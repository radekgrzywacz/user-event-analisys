package analyser

import (
	"analyser/internal/event"
	"analyser/internal/store"
	"analyser/internal/user"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

func saveEventToPostgres(pg *store.Queries, ev event.Event) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := user.EnsureUserExists(pg, ctx, int64(ev.UserId)); err != nil {
		return 0, fmt.Errorf("Cannot ensure user existance: %w", err)
	}
	meta, _ := json.Marshal(ev.Metadata)

	return pg.InsertEvent(ctx, store.InsertEventParams{
		UserID:    int32(ev.UserId),
		EventType: string(ev.Type),
		Timestamp: pgtype.Timestamptz{Time: ev.Timestamp, Valid: true},
		Ip:        pgtype.Text{String: ev.Metadata.IP, Valid: ev.Metadata.IP != ""},
		UserAgent: pgtype.Text{String: ev.Metadata.UserAgent, Valid: ev.Metadata.UserAgent != ""},
		Country:   pgtype.Text{String: ev.Metadata.Country, Valid: ev.Metadata.Country != ""},
		SessionID: pgtype.Text{String: ev.SessionId, Valid: ev.SessionId != ""},
		Metadata:  meta,
	})
}

func saveAnomalyToPostgres(pg *store.Queries, ev event.Event, res AnalyseResult) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	eventId, err := saveEventToPostgres(pg, ev)
	if err != nil {
		return 0, fmt.Errorf("Could not save an event: %w", err)
	}

	details, err := json.Marshal(map[string]interface{}{
		"message": res.Message,
	})
	if err != nil {
		return 0, fmt.Errorf("Could not marshal anomaly message: %w", err)
	}

	return pg.InsertAnomaly(ctx, store.InsertAnomalyParams{
		UserID:      int32(ev.UserId),
		EventID:     pgtype.Int8{Int64: eventId, Valid: true},
		AnomalyType: res.AnomalyType,
		Details:     details,
		DetectedAt:  pgtype.Timestamptz{Time: res.Timestamp, Valid: true},
	})
}

func putEventToRedis(event event.Event, rdb *redis.Client) error {
	if err := addEventToRedis(event, rdb); err != nil {
		return fmt.Errorf("Redis insert failed: %w", err)
	}
	if err := recordTransition(event, rdb); err != nil {
		return fmt.Errorf("Redis transition record failed: %w", err)
	}

	return nil
}
