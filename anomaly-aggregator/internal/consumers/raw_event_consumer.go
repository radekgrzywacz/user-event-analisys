package consumers

import (
	"anomaly-aggregator/internal/store"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/twmb/franz-go/pkg/kgo"

	contracts "user-event-analisys/contracts/events"
)

type RawEventConsumer struct {
	client *kgo.Client
	pg     *store.Queries
}

func NewRawEventConsumer(client *kgo.Client, pg *store.Queries) *RawEventConsumer {
	return &RawEventConsumer{client: client, pg: pg}
}

func (c *RawEventConsumer) ConsumeTopic(ctx context.Context, handle func(contracts.UserActivityPayload) error) error {
	log.Println("[Kafka] Listening for raw events...")

	for {
		fetches := c.client.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				if err := c.handleRecord(record, handle); err != nil {
					log.Printf("Error handling record: %v", err)
				}
			}
		})
	}
}

func (c *RawEventConsumer) handleRecord(record *kgo.Record, handle func(contracts.UserActivityPayload) error) error {
	raw := record.Value

	envelope, err := contracts.ParseEnvelope(raw)
	if err != nil {
		return fmt.Errorf("parse envelope: %w", err)
	}

	if envelope.Domain != contracts.DomainUserActivity {
		log.Printf("Skipping domain: %s", envelope.Domain)
		return nil
	}

	payload, err := envelope.UserActivityPayload()
	if err != nil {
		return fmt.Errorf("decode user activity: %w", err)
	}

	if err := handle(payload); err != nil {
		return fmt.Errorf("handle payload: %w", err)
	}
	return nil
}

func (c *RawEventConsumer) InsertEvent(payload contracts.UserActivityPayload) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	additional, err := json.Marshal(payload.Additional)
	if err != nil {
		return fmt.Errorf("marshal additional: %w", err)
	}

	ev := store.InsertEventParams{
		UserID:    int32(payload.UserID),
		EventType: string(payload.Type),

		Timestamp: pgtype.Timestamptz{
			Time:  payload.Timestamp.UTC(),
			Valid: true,
		},

		Ip: pgtype.Text{
			String: payload.Metadata.IP,
			Valid:  true,
		},
		UserAgent: pgtype.Text{
			String: payload.Metadata.UserAgent,
			Valid:  true,
		},
		Country: pgtype.Text{
			String: payload.Metadata.Country,
			Valid:  true,
		},
		SessionID: pgtype.Text{
			String: payload.SessionID,
			Valid:  true,
		},
		Metadata: additional,
	}

	_, err = c.pg.InsertEvent(ctx, ev)
	if err != nil {
		return fmt.Errorf("insert raw event: %w", err)
	}

	return nil
}
