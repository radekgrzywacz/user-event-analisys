package analyser

import (
	"analyser/internal/event"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

func Process(event event.Event, rdb *redis.Client) error {
	if err := addEventToRedis(event, rdb); err != nil {
		return fmt.Errorf("%d", err)
	}

	return nil
}

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

	return nil
}

type AnalyseResult struct {
	Anomaly   bool   `json:"anomaly"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

func analyseCached(event event.Event, rdb *redis.Client) AnalyseResult {
	var result AnalyseResult
	result = checkStoredUserData(event, rdb)

	return result
}

func checkStoredUserData(event event.Event, rdb *redis.Client) AnalyseResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := rdb.SIsMember(ctx, fmt.Sprintf("user:%d:ips", event.UserId), event.Metadata.IP)
	if res.Val() == true {
		return AnalyseResult{
			Anomaly:   res.Val(),
			Message:   "IP address never seen before",
			Timestamp: time.Now().Format("20060102150405"),
		}
	}
	res = rdb.SIsMember(ctx, fmt.Sprintf("user:%d:user_agents", event.UserId), event.Metadata.UserAgent)
	if res.Val() == true {
		return AnalyseResult{
			Anomaly:   res.Val(),
			Message:   "User agent never seen before",
			Timestamp: time.Now().Format("20060102150405"),
		}
	}
	res = rdb.SIsMember(ctx, fmt.Sprintf("user:%d:countries", event.UserId), event.Metadata.Country)
	if res.Val() == true {
		return AnalyseResult{
			Anomaly:   res.Val(),
			Message:   "Country never seen before",
			Timestamp: time.Now().Format("20060102150405"),
		}
	}
	return AnalyseResult{}
}

func checkEventZScore(eventType event.EventType, e event.Event, rdb *redis.Client) (AnalyseResult, error) {
	now := time.Now()
	start := now.Add(-1 * time.Hour).Unix()
	end := now.Unix()
	events, err := getEventsFromWindow(e.UserId, rdb, start, end)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Could not get events from the time window")
	}
	eventsCount := countEventTypeOccuranceInWindow(eventType, events)
	previousAverage, err := countAverageOccurancesInPreviousWindows(eventType, 10, e.UserId, rdb)
	if err != nil {
		fmt.Errorf("%w", err)
	}

	stdDev, err := countStdDevOfOccourancesInPreviousWindows(eventType, 10, e.UserId, rdb)
	if err != nil {
		return AnalyseResult{}, fmt.Errorf("Could not calculate standard deviation: %w", err)
	}
	if stdDev == 0 {
		return AnalyseResult{
			Anomaly:   false,
			Message:   "Standard deviation is zero; insufficient variance for Z-score analysis",
			Timestamp: time.Now().Format("20060102150405"),
		}, nil
	}

	zScore := float64(eventsCount-int(previousAverage)) / stdDev
	return AnalyseResult{
		Anomaly:   math.Abs(zScore) > 3,
		Message:   fmt.Sprintf("Z-Score for %s: %.2f", eventType, zScore),
		Timestamp: time.Now().Format("20060102150405"),
	}, nil
}
