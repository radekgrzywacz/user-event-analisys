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

	rdb.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", time.Now().Add(-72*time.Hour).Unix()))
	rdb.Expire(ctx, key, 72*time.Hour)

	rdb.SAdd(ctx, fmt.Sprintf("user:%d:ips", event.UserId), event.Metadata.IP)
	rdb.Expire(ctx, fmt.Sprintf("user:%d:ips", event.UserId), 14*24*time.Hour)

	rdb.SAdd(ctx, fmt.Sprintf("user:%d:user_agents", event.UserId), event.Metadata.UserAgent)
	rdb.Expire(ctx, fmt.Sprintf("user:%d:user_agents", event.UserId), 14*24*time.Hour)

	rdb.SAdd(ctx, fmt.Sprintf("user:%d:countries", event.UserId), event.Metadata.Country)
	rdb.Expire(ctx, fmt.Sprintf("user:%d:countries", event.UserId), 14*24*time.Hour)

	rdb.HIncrBy(ctx, fmt.Sprintf("user:%d:activity_hours", event.UserId), strconv.Itoa(event.Timestamp.Hour()), 1)
	rdb.Expire(ctx, fmt.Sprintf("user:%d:activity_hours", event.UserId), 14*24*time.Hour)

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

	var cur, prev2, prev1 event.Event
	json.Unmarshal([]byte(events[0]), &cur)
	json.Unmarshal([]byte(events[1]), &prev1)
	json.Unmarshal([]byte(events[2]), &prev2)

	histogramKey := fmt.Sprintf("user:%d:transitions", cur.UserId)
	globalHistogramKey := "global:transitions"
	firstOrderTransitionKey := fmt.Sprintf("%s->%s", prev1.Type, cur.Type)
	rdb.HIncrBy(ctx, histogramKey, firstOrderTransitionKey, 1)
	rdb.HIncrBy(ctx, globalHistogramKey, firstOrderTransitionKey, 1)

	secondOrderTransitionKey := fmt.Sprintf("%s->%s->%s", prev2.Type, prev1.Type, cur.Type)
	rdb.HIncrBy(ctx, histogramKey, secondOrderTransitionKey, 1)
	rdb.HIncrBy(ctx, globalHistogramKey, secondOrderTransitionKey, 1)

	return nil
}

func analyseCached(event event.Event, rdb *redis.Client) (AnalyseResult, error) {
	if result := checkStoredUserData(event, rdb); result.Anomaly {
		return result, nil
	}

	result, err := checkMarkovAnomaly(event, rdb)
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

	currentCount := float64(countEventTypeOccuranceInWindow(eventType, events))

	alpha := 0.3
	ema, err := updateEMA(rdb, e.UserId, eventType, currentCount, alpha)
	if err != nil {
		return AnalyseResult{}, err
	}

	deviation := math.Abs(currentCount - ema)
	emaStd, err := updateEMAStdDev(rdb, e.UserId, eventType, deviation, alpha)
	if err != nil {
		return AnalyseResult{}, err
	}

	threshold := 2.5
	if emaStd < 1 {
		threshold = 4.0
	}
	if deviation > threshold*emaStd && emaStd > 0 {
		// TODO: Add rolling calibration
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "ema_outlier",
			Message:     fmt.Sprintf("Event frequency for %s deviates from baseline: %.2f vs EMA=%.2f (σ=%.2f)", eventType, currentCount, ema, emaStd),

			Timestamp: time.Now(),
		}, nil
	}
	return AnalyseResult{}, nil
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

func checkMarkovAnomaly(e event.Event, rdb *redis.Client) (AnalyseResult, error) {
	prev1, prev2, err := getMarkovHistory(e, rdb)
	if err != nil {
		return AnalyseResult{}, err
	}

	userHistogramKey := fmt.Sprintf("user:%d:transitions", e.UserId)
	const (
		userBaseline   = "user history"
		globalBaseline = "global history"
	)

	result, hasData, err := checkSecondOrderMarkovAnomaly(e, rdb, userHistogramKey, userBaseline, prev1, prev2)
	if err != nil {
		return AnalyseResult{}, err
	}
	if result.Anomaly {
		return result, nil
	}
	if hasData {
		return AnalyseResult{}, nil
	}

	result, hasData, err = checkFirstOrderMarkovAnomaly(e, rdb, userHistogramKey, userBaseline, prev1)
	if err != nil {
		return AnalyseResult{}, err
	}
	if result.Anomaly {
		return result, nil
	}
	if hasData {
		return AnalyseResult{}, nil
	}

	result, hasData, err = checkSecondOrderMarkovAnomaly(e, rdb, "global:transitions", globalBaseline, prev1, prev2)
	if err != nil {
		return AnalyseResult{}, err
	}
	if result.Anomaly {
		return result, nil
	}
	if hasData {
		return AnalyseResult{}, nil
	}

	result, _, err = checkFirstOrderMarkovAnomaly(e, rdb, "global:transitions", globalBaseline, prev1)
	if err != nil {
		return AnalyseResult{}, err
	}
	if result.Anomaly {
		return result, nil
	}

	return AnalyseResult{}, nil
}

func getMarkovHistory(e event.Event, rdb *redis.Client) (*event.Event, *event.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d", e.UserId)
	rawEvents, err := rdb.ZRevRange(ctx, key, 0, 1).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("Could not retrieve recent events: %w", err)
	}

	if len(rawEvents) == 0 {
		return nil, nil, nil
	}

	var prev1 event.Event
	if err := json.Unmarshal([]byte(rawEvents[0]), &prev1); err != nil {
		return nil, nil, fmt.Errorf("Could not unmarshal recent event: %w", err)
	}

	if len(rawEvents) == 1 {
		return &prev1, nil, nil
	}

	var prev2 event.Event
	if err := json.Unmarshal([]byte(rawEvents[1]), &prev2); err != nil {
		return &prev1, nil, fmt.Errorf("Could not unmarshal second recent event: %w", err)
	}

	return &prev1, &prev2, nil
}

func checkFirstOrderMarkovAnomaly(e event.Event, rdb *redis.Client, histogramKey, baseline string, prev *event.Event) (AnalyseResult, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prev == nil {
		return AnalyseResult{}, false, nil
	}

	prefix := fmt.Sprintf("%s->", prev.Type)
	transition := fmt.Sprintf("%s->%s", prev.Type, e.Type)
	allTransitions, err := rdb.HGetAll(ctx, histogramKey).Result()
	if err != nil {
		return AnalyseResult{}, false, fmt.Errorf("Could not fetch transition data: %w", err)
	}

	total := 0
	count := 0
	for k, v := range allTransitions {
		if strings.HasPrefix(k, prefix) && strings.Count(k, "->") == 1 {
			c, _ := strconv.Atoi(v)
			total += c
			if k == transition {
				count = c
			}
		}
	}

	// < 20 to avoid false positive
	if total < 20 {
		return AnalyseResult{}, false, nil
	}

	probability := float64(count) / float64(total)

	threshold := math.Max(0.01, 1.0/math.Sqrt(float64(total)))
	if total < 50 {
		threshold = 0.02
	} else if total > 200 {
		threshold = 0.05
	}

	if probability < threshold {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "markov_low_probability",
			Message:     formatMarkovMessage(fmt.Sprintf("Unusual transition %s->%s", prev.Type, e.Type), baseline),
			Timestamp:   time.Now(),
		}, true, nil
	}

	return AnalyseResult{}, true, nil
}

func checkSecondOrderMarkovAnomaly(e event.Event, rdb *redis.Client, histogramKey, baseline string, prev1, prev2 *event.Event) (AnalyseResult, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if prev1 == nil || prev2 == nil {
		return AnalyseResult{}, false, nil
	}

	prefix := fmt.Sprintf("%s->%s->", prev2.Type, prev1.Type)
	transition := fmt.Sprintf("%s->%s->%s", prev2.Type, prev1.Type, e.Type)
	allTransitions, err := rdb.HGetAll(ctx, histogramKey).Result()
	if err != nil {
		return AnalyseResult{}, false, fmt.Errorf("Could not fetch transition data: %w", err)
	}

	total := 0
	count := 0
	for k, v := range allTransitions {
		if strings.HasPrefix(k, prefix) && strings.Count(k, "->") == 2 {
			c, _ := strconv.Atoi(v)
			total += c
			if k == transition {
				count = c
			}
		}
	}

	// < 20 to avoid false positive
	if total < 20 {
		return AnalyseResult{}, false, nil
	}

	probability := float64(count) / float64(total)

	threshold := math.Max(0.01, 1.0/math.Sqrt(float64(total)))
	if total < 50 {
		threshold = 0.02
	} else if total > 200 {
		threshold = 0.05
	}

	if probability < threshold {
		return AnalyseResult{
			Anomaly:     true,
			AnomalyType: "markov_low_probability",
			Message:     formatMarkovMessage(fmt.Sprintf("Unusual transition %s->%s->%s", prev2.Type, prev1.Type, e.Type), baseline),
			Timestamp:   time.Now(),
		}, true, nil
	}

	return AnalyseResult{}, true, nil
}

func formatMarkovMessage(base, baseline string) string {
	if baseline == "" {
		return base
	}
	return fmt.Sprintf("%s based on %s", base, baseline)
}
