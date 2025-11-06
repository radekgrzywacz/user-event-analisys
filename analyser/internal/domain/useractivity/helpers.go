package useractivity

import (
	"analyser/internal/event"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
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

func countOccuranceTimeStdDev(e event.Event, rdb *redis.Client) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vals, err := rdb.HGetAll(ctx, fmt.Sprintf("user:%d:activity_hours", e.UserId)).Result()
	if err != nil {
		return 0, fmt.Errorf("Could not get activity hours of user with id: %d", e.UserId)
	}

	hourlyCounts := make(map[int]int)
	for k, v := range vals {
		intKey, err := strconv.Atoi(k)
		if err != nil {
			log.Printf("Could not convert the key: %v", err)
			continue
		}

		intVal, err := strconv.Atoi(v)
		if err != nil {
			log.Printf("Could not convert the value: %v", err)
			continue
		}

		hourlyCounts[intKey] = intVal
	}

	var total, count float64
	for h := 0; h < 24; h++ {
		v := float64(hourlyCounts[h])
		total += v
		count++
	}

	mean := total / count

	var variance float64
	for h := 0; h < 24; h++ {
		v := float64(hourlyCounts[h])
		variance += math.Pow(v-mean, 2)
	}

	stdDev := math.Sqrt(variance / count)
	return stdDev, nil
}

func updateEMA(rdb *redis.Client, userId int, eventType event.EventType, current float64, alpha float64) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d:ema:%s", userId, eventType)
	prevStr, err := rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		if err := rdb.Set(ctx, key, current, 30*24*time.Hour).Err(); err != nil {
			return 0, fmt.Errorf("Failed to set initial EMA: %w", err)
		}
		return current, nil
	} else if err != nil {
		return 0, fmt.Errorf("Could not get previous EMA: %w", err)
	}

	prevEMA, err := strconv.ParseFloat(prevStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid previous EMA value: %w", err)
	}

	newEma := alpha*current + (1-alpha)*prevEMA
	if err := rdb.Set(ctx, key, newEma, 30*24*time.Hour).Err(); err != nil {
		return 0, fmt.Errorf("Could not store new EMA: %w", err)
	}

	return newEma, nil
}

func updateEMAStdDev(rdb *redis.Client, userId int, eventType event.EventType, deviation, alpha float64) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:%d:ema_std:%s", userId, eventType)
	prevStr, err := rdb.Get(ctx, key).Result()

	if err == redis.Nil {
		rdb.Set(ctx, key, deviation, 30*24*time.Hour)
		return deviation, nil
	} else if err != nil {
		return 0, fmt.Errorf("Could not get previous EMA stddev: %w", err)
	}

	prevStd, err := strconv.ParseFloat(prevStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid previous EMA stddev value: %w", err)
	}

	newStd := alpha*deviation + (1-alpha)*prevStd
	if err := rdb.Set(ctx, key, newStd, 30*24*time.Hour).Err(); err != nil {
		return 0, fmt.Errorf("Could not store new EMA stddev: %w", err)
	}

	return newStd, nil
}
