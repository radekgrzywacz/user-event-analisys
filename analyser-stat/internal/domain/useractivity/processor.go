package useractivity

import (
	"analyser/internal/event"
	"analyser/internal/processor"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type AnalyseResult struct {
	Anomaly     bool      `json:"anomaly"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	AnomalyType string    `json:"anomaly_type"`
}

// TODO: Dodać do redisa:
// 1. Zbyt częste logowania
// 2. Szybkość zmian eventów
// 3. Odległość geograficzna po IP
// 4. Za duzo failed loginow

// TODO: Poprawić uczenie sie markova, zeby zapobiegać data poisoning.

// TODO: Dodać heartbeat do 'klienta' w celu tworzenia wykresu uptime serwisu

func Process(event event.Event, rdb *redis.Client, producer *processor.Producer) error {
	ctx := context.Background()
	result, err := analyseCached(event, rdb)
	if err != nil {
		return fmt.Errorf("error analysing cached data: %w", err)
	}

	if result.Anomaly {
		if err := putEventToRedis(event, rdb); err != nil {
			return fmt.Errorf("redis insert failed for anomaly event: %w", err)
		}
		producer.PublishResult(ctx, processor.ResultEvent{
			UserId:      event.UserId,
			SessionId:   event.SessionId,
			EventType:   string(event.Type),
			Anomaly:     true,
			AnomalyType: result.AnomalyType,
			Message:     result.Message,
			Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
			Source:      "stat",
		})
		log.Printf("Result produced")
		return nil
	}

	result, err = analyseStatistics(event, rdb)
	if err != nil {
		return fmt.Errorf("error analysing statistics: %w", err)
	}

	if result.Anomaly {
		if err := putEventToRedis(event, rdb); err != nil {
			return fmt.Errorf("redis insert failed for anomaly event: %w", err)
		}
		producer.PublishResult(ctx, processor.ResultEvent{
			UserId:      event.UserId,
			SessionId:   event.SessionId,
			EventType:   string(event.Type),
			Anomaly:     true,
			AnomalyType: result.AnomalyType,
			Message:     result.Message,
			Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
			Source:      "stat",
		})
		return nil
	}

	if err := putEventToRedis(event, rdb); err != nil {
		return fmt.Errorf("redis insert failed for normal event: %w", err)
	}
	producer.PublishResult(ctx, processor.ResultEvent{
		UserId:      event.UserId,
		SessionId:   event.SessionId,
		EventType:   string(event.Type),
		Anomaly:     false,
		AnomalyType: "",
		Message:     "No anomaly deteceted",
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		Source:      "stat",
	})
	return nil
}
