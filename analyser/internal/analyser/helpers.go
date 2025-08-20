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

// Time calculated in unix values
func getEventsFromWindow(userId int, rdb *redis.Client, start, end int64) ([]event.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d", userId)

	rawEvents, err := rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", start),
		Max: fmt.Sprintf("%d", end),
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch events: %w", err)
	}

	var events []event.Event
	for _, raw := range rawEvents {
		var e event.Event
		if err := json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, fmt.Errorf("Failed to deserialize event: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

func countEventTypeOccuranceInWindow(t event.EventType, events []event.Event) int {
	count := 0
	for _, event := range events {
		if event.Type == t {
			count++
		}
	}

	return count
}

func countAverageOccurancesInPreviousWindows(
	eventType event.EventType,
	windowsAmount, userId int,
	rdb *redis.Client,
) (float32, error) {
	if windowsAmount == 0 {
		return 0, fmt.Errorf("windowsAmount cannot be zero")
	}

	now := time.Now()
	totalOccurrences := 0.0

	for i := 2; i <= windowsAmount+1; i++ {
		start := now.Add(-time.Duration(i) * time.Hour).Unix()
		end := now.Add(-time.Duration(i-1) * time.Hour).Unix()

		events, err := getEventsFromWindow(userId, rdb, start, end)
		if err != nil {
			return 0, fmt.Errorf("could not get events for window %d: %w", i, err)
		}

		totalOccurrences += float64(countEventTypeOccuranceInWindow(eventType, events))
	}

	return float32(totalOccurrences) / float32(windowsAmount), nil
}

func countStdDevOfOccourancesInPreviousWindows(
	eventType event.EventType,
	windowsAmount, userId int,
	rdb *redis.Client,
) (float64, error) {
	now := time.Now()
	var counts []float64

	for i := 2; i < windowsAmount+1; i++ {
		start := now.Add(-time.Duration(i) * time.Hour).Unix()
		end := now.Add(-time.Duration(i-1) * time.Hour).Unix()

		events, err := getEventsFromWindow(userId, rdb, start, end)
		if err != nil {
			return 0, fmt.Errorf("Could not get events for window %d: %w", i, err)
		}

		count := countEventTypeOccuranceInWindow(eventType, events)
		counts = append(counts, float64(count))
	}

	var sum float64
	for _, c := range counts {
		sum += c
	}
	mean := sum / float64(len(counts))

	var variance float64
	for _, c := range counts {
		variance += (c - mean) * (c + mean)
	}
	variance /= float64(len(counts))

	return math.Sqrt(variance), nil
}
