package analyser

import (
	"analyser/internal/event"
	"context"
	"encoding/json"
	"fmt"
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
	rdb.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", cutoff))

	rdb.SAdd(ctx, fmt.Sprintf("user:%d:ips", event.UserId), event.Metadata.IP)
	rdb.SAdd(ctx, fmt.Sprintf("user:%d:user_agents", event.UserId), event.Metadata.UserAgent)
	rdb.SAdd(ctx, fmt.Sprintf("user:%d:countries", event.UserId), event.Metadata.Country)

	return nil
}
