package analyser

import (
	"analyser/internal/event"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func addEventToRedis(event event.Event, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d", event.UserId)

	serialized, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("Could not serialize event: %w", err)
	}

	z := redis.Z{
		Score:  float64(event.Timestamp.Unix()),
		Member: serialized,
	}
	if err := rdb.ZAdd(ctx, key, z).Err(); err != nil {
		return fmt.Errorf("Failed to add event to redis: %w", err)
	}

	cutoff := time.Now().Add(-24 * time.Hour * 3).Unix()
	if err := rdb.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", cutoff)).Err(); err != nil {
		return fmt.Errorf("Failed to clean up old events: %w", err)
	}

	rdb.SAdd(ctx, fmt.Sprintf("user:%d:ips", event.UserId), event.Metadata.IP)
	rdb.SAdd(ctx, fmt.Sprintf("user:%d:user_agents", event.UserId), event.Metadata.UserAgent)
	rdb.SAdd(ctx, fmt.Sprintf("user:%d:countries", event.UserId), event.Metadata.Country)

	rdb.HIncrBy(ctx, fmt.Sprintf("user:%d:activity_hours", event.UserId), strconv.Itoa(event.Timestamp.Hour()), 1)

	return nil
}

func recordTransition(e event.Event, rdb *redis.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d", e.UserId)
	events, err := rdb.ZRevRange(ctx, key, 0, 2).Result()
	if err != nil || len(events) < 2 {
		return nil
	}

	var prev2, prev1 event.Event
	json.Unmarshal([]byte(events[1]), &prev2)
	json.Unmarshal([]byte(events[0]), &prev1)

	transitionKey := fmt.Sprintf("%s->%s->%s", prev2.Type, prev1.Type, e.Type)
	histogramKey := fmt.Sprintf("user:%d:transitions", e.UserId)

	return rdb.HIncrBy(ctx, histogramKey, transitionKey, 1).Err()
}

func analyseCached(event event.Event, rdb *redis.Client) (AnalyseResult, error) {
	if result := checkStoredUserData(event, rdb); result.Anomaly {
		return result, nil
	}

	result, err := checkForValidEventTransition(event, rdb)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Error checking event transition: %w", err)
	}
	if result.Anomaly {
		return result, nil
	}

	result, err = checkMarkovAnomaly(event, rdb)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Error checking Markov anomaly: %w", err)
	}
	if result.Anomaly {
		return result, nil
	}

	return AnalyseResult{}, nil
}

func checkStoredUserData(event event.Event, rdb *redis.Client) AnalyseResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := rdb.SIsMember(ctx, fmt.Sprintf("user:%d:ips", event.UserId), event.Metadata.IP)
	if !res.Val() {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "new_ip",
			Message:     "IP address never seen before",
			Timestamp:   time.Now(),
		}
	}
	res = rdb.SIsMember(ctx, fmt.Sprintf("user:%d:user_agents", event.UserId), event.Metadata.UserAgent)
	if !res.Val() {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "new_user_agent",
			Message:     "User agent never seen before",
			Timestamp:   time.Now(),
		}
	}
	res = rdb.SIsMember(ctx, fmt.Sprintf("user:%d:countries", event.UserId), event.Metadata.Country)
	if !res.Val() {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "new_country",
			Message:     "Country never seen before",
			Timestamp:   time.Now(),
		}
	}
	return AnalyseResult{}
}

func checkEventZScore(eventType event.EventType, e event.Event, rdb *redis.Client, windowSize time.Duration) (AnalyseResult, error) {
	now := time.Now()
	start := now.Add(-1 * windowSize).Unix()
	end := now.Unix()

	events, err := getEventsFromWindow(e.UserId, rdb, start, end)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Could not get events from the time window")
	}

	eventsCount := countEventTypeOccuranceInWindow(eventType, events)
	previousAverage, err := countAverageOccurancesInPreviousWindows(eventType, 10, e.UserId, rdb, windowSize)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("%w", err)
	}

	stdDev, err := countStdDevOfOccourancesInPreviousWindows(eventType, 10, e.UserId, rdb, windowSize)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Could not calculate standard deviation: %w", err)
	}
	if stdDev == 0 {
		return AnalyseResult{
			Anomaly:     false,
			AnomalyType: "no_variance",
			Message:     "Standard deviation is zero; insufficient variance for Z-score analysis",
			Timestamp:   time.Now(),
		}, nil
	}

	zScore := float64(eventsCount-int(previousAverage)) / stdDev
	return AnalyseResult{
		Anomaly:     math.Abs(zScore) > 3,
		AnomalyType: "zscore_outlier",
		Message:     fmt.Sprintf("Z-Score for %s: %.2f", eventType, zScore),
		Timestamp:   time.Now(),
	}, nil
}

func checkEventTypesZScore(e event.Event, rdb *redis.Client) (AnalyseResult, error) {
	var eventTypeWindow = map[event.EventType]time.Duration{
		event.EventLogin:         15 * time.Minute,
		event.EventPayment:       10 * time.Minute,
		event.EventLogout:        1 * time.Hour,
		event.EventFailedLogin:   15 * time.Minute,
		event.EventPasswordReset: 2 * time.Hour,
		event.EventOther:         2 * time.Hour,
	}

	window, ok := eventTypeWindow[e.Type]
	if !ok {
		window = 1 * time.Hour
	}

	result, err := checkEventZScore(e.Type, e, rdb, window)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("%w", err)
	}

	return result, nil
}

func analyseStatistics(event event.Event, rdb *redis.Client) (AnalyseResult, error) {
	result, err := checkEventTypesZScore(event, rdb)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Error calculating ZScore: %w", err)
	}
	if result.Anomaly {
		return result, nil
	}

	result, err = checkTimeDeviation(event, rdb)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Error calculating time deviation: %w", err)
	}
	if result.Anomaly {
		return result, nil
	}

	return AnalyseResult{}, err
}

func checkTimeDeviation(e event.Event, rdb *redis.Client) (AnalyseResult, error) {
	hour := e.Timestamp.Hour()
	stdDev, err := countOccuranceTimeStdDev(e, rdb)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Error calculating hour std dev: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hourKey := fmt.Sprintf("user:%d:activity_hours", e.UserId)
	val, err := rdb.HGet(ctx, hourKey, strconv.Itoa(hour)).Int()
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Could not get event count for hour %d: %w", hour, err)
	}

	if float64(val) < stdDev {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "time_deviation",
			Message:     fmt.Sprintf("Activity at unusual hour %d. Count=%d < stdDev=%.2f", hour, val, stdDev),
			Timestamp:   time.Now(),
		}, nil
	}

	return AnalyseResult{
		Anomaly:   false,
		Message:   "",
		Timestamp: time.Now(),
	}, nil
}

func checkForValidEventTransition(e event.Event, rdb *redis.Client) (AnalyseResult, error) {
	var allowedTransitions = map[event.EventType][]event.EventType{
		event.EventLogin:         {event.EventPayment, event.EventLogout, event.EventFailedLogin},
		event.EventPayment:       {event.EventLogout, event.EventOther},
		event.EventLogout:        {event.EventLogin},
		event.EventFailedLogin:   {event.EventLogin, event.EventPasswordReset},
		event.EventPasswordReset: {event.EventLogin},
		event.EventOther:         {event.EventLogout, event.EventLogin},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d", e.UserId)
	events, err := rdb.ZRevRange(ctx, key, 0, 0).Result()
	if err != nil || len(events) == 0 {
		return AnalyseResult{}, fmt.Errorf("Could not retrieve last events: %w", err)
	}

	var lastEvent event.Event
	if err := json.Unmarshal([]byte(events[0]), &lastEvent); err != nil {
		return AnalyseResult{}, nil
	}

	allowedNext, ok := allowedTransitions[lastEvent.Type]
	if !ok {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "unknown_transition_rule",
			Message:     fmt.Sprintf("No transition rule form previous event type %s", lastEvent.Type),
			Timestamp:   time.Now(),
		}, nil
	}

	for _, next := range allowedNext {
		if next == e.Type {
			return AnalyseResult{}, nil
		}
	}

	return AnalyseResult{
		Anomaly:     true,
		AnomalyType: "invalid_transition",
		Message:     fmt.Sprintf("Disallowed transition: %s->%s", lastEvent.Type, e.Type),
		Timestamp:   time.Now(),
	}, nil
}

func checkMarkovAnomaly(e event.Event, rdb *redis.Client) (AnalyseResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d", e.UserId)
	events, err := rdb.ZRevRange(ctx, key, 0, 2).Result()
	if err != nil || len(events) < 2 {
		return AnalyseResult{}, nil
	}

	var prev2, prev1 event.Event
	json.Unmarshal([]byte(events[1]), &prev2)
	json.Unmarshal([]byte(events[0]), &prev1)

	histogramKey := fmt.Sprintf("user:%d:transitions", e.UserId)

	prefix := fmt.Sprintf("%s->%s->%s", prev2.Type, prev1.Type, e.Type)
	allTransitions, err := rdb.HGetAll(ctx, histogramKey).Result()
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Could not fetch transition data: %w", err)
	}

	total := 0
	count := 0
	for k, v := range allTransitions {
		if strings.HasPrefix(k, prefix) {
			c, _ := strconv.Atoi(v)
			total += c
			if strings.HasSuffix(k, string(e.Type)) {
				count = c
			}
		}
	}

	if total == 0 {
		return AnalyseResult{}, nil
	}

	probability := float64(count) / float64(total)
	if probability < 0.05 {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "markov_low_probability",
			Message:     fmt.Sprintf("Unusual transition %s->%s->%s", prev2.Type, prev1.Type, e.Type),
			Timestamp:   time.Now(),
		}, nil
	}

	return AnalyseResult{}, nil
}
