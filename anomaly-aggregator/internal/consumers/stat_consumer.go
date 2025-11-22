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

type StatConsumer struct {
	client *kgo.Client
	pg     *store.Queries
	fusion *aggregator.Aggregator
}

func NewStatConsumer(client *kgo.Client, pg *store.Queries, fusion *aggregator.Aggregator) *StatConsumer {
	return &StatConsumer{client: client, pg: pg, fusion: fusion}
}

func (c *StatConsumer) ConsumeTopic(ctx context.Context) error {
	log.Println("[Kafka] Listening for stat results...")

	for {
		fetches := c.client.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				var msg events.StatResult
				if err := json.Unmarshal(record.Value, &msg); err != nil {
					log.Printf("Error decoding stat result: %v", err)
					continue
				}

				if err := c.insertStatResult(msg); err != nil {
					log.Printf("Error inserting stat result: %v", err)
				}

				if err := c.fusion.OnStatResult(ctx, aggregator.StatResult{
					UserID:      msg.UserID,
					SessionID:   msg.SessionID,
					Anomaly:     msg.Anomaly,
					AnomalyType: msg.AnomalyType,
					Timestamp:   msg.Timestamp,
				}); err != nil {
					log.Printf("Fusion.OnStatResult error: %v", err)
				}
			}
		})
	}
}

func (c *StatConsumer) insertStatResult(payload events.StatResult) error {
	ctx := context.Background()
	_, err := c.pg.InsertStatResult(ctx, store.InsertStatResultParams{
		UserID:      int32(payload.UserID),
		SessionID:   payload.SessionID,
		EventType:   pgtype.Text{String: payload.EventType, Valid: true},
		Anomaly:     payload.Anomaly,
		AnomalyType: pgtype.Text{String: payload.AnomalyType, Valid: true},
		Message:     pgtype.Text{String: payload.Message, Valid: true},
		Timestamp:   pgtype.Timestamptz{Time: payload.Timestamp.UTC(), Valid: true},
		Source:      pgtype.Text{String: "stat", Valid: true},
	})
	log.Println("Inserted stat")
	return err
}
