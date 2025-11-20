package useractivity

import (
	"analyser/internal/event"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func putEventToRedis(event event.Event, rdb *redis.Client) error {
	if err := addEventToRedis(event, rdb); err != nil {
		return fmt.Errorf("Redis insert failed: %w", err)
	}
	if err := recordTransition(event, rdb); err != nil {
		return fmt.Errorf("Redis transition record failed: %w", err)
	}

	return nil
}
