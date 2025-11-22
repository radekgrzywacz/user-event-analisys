package consumers

import (
	"anomaly-aggregator/internal/aggregator"
	"anomaly-aggregator/internal/events"
	"anomaly-aggregator/internal/store"
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/twmb/franz-go/pkg/kgo"
)

type MLConsumer struct {
	client *kgo.Client
	pg     *store.Queries
	fusion *aggregator.Aggregator
}

func NewMLConsumer(client *kgo.Client, pg *store.Queries, fusion *aggregator.Aggregator) *MLConsumer {
	return &MLConsumer{client: client, pg: pg, fusion: fusion}
}

func (c *MLConsumer) ConsumeTopic(ctx context.Context) error {
	log.Println("[Kafka] Listening for ML results...")

	for {
		fetches := c.client.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				var msg events.MLResult
				if err := json.Unmarshal(record.Value, &msg); err != nil {
					log.Printf("Error decoding ML result: %v", err)
					continue
				}

				if err := c.insertMLResult(msg); err != nil {
					log.Printf("Error inserting ML result: %v", err)
				}

				if err := c.fusion.OnMLResult(ctx, aggregator.MLResult{
					UserID:       msg.UserID,
					SessionID:    msg.SessionID,
					Anomaly:      msg.Anomaly,
					Score:        msg.Score,
					Threshold:    msg.Threshold,
					EventCount:   msg.EventCount,
					UniqueEvents: msg.UniqueEvents,
				}); err != nil {
					log.Printf("Fusion.OnMLResult error: %v", err)
				}
			}
		})
	}
}

func (c *MLConsumer) insertMLResult(payload events.MLResult) error {
	ctx := context.Background()
	_, err := c.pg.InsertMLResult(ctx, store.InsertMLResultParams{
		UserID:     int32(payload.UserID),
		SessionID:  payload.SessionID,
		Anomaly:    payload.Anomaly,
		Score:      pgtype.Float8{Float64: payload.Score, Valid: true},
		Threshold:  pgtype.Float8{Float64: payload.Threshold, Valid: true},
		EventCount: pgtype.Int4{Int32: int32(payload.EventCount), Valid: true},
	})
	log.Println("Inserted ml")
	return err
}
