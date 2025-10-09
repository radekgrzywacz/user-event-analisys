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
		Metadata:  meta,
	})
}
