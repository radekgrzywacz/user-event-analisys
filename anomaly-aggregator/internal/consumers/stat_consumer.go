package consumers

import (
	"context"
	"encoding/json"
	"log"

	"anomaly-aggregator/internal/events"
	"anomaly-aggregator/internal/store"

	"github.com/twmb/franz-go/pkg/kgo"
)

type StatConsumer struct {
	client *kgo.Client
	pg     *store.Queries
}

func NewStatConsumer(client *kgo.Client, pg *store.Queries) *StatConsumer {
	return &StatConsumer{client: client, pg: pg}
}

func (c *StatConsumer) ConsumeTopic(ctx context.Context, handle func(events.StatResult) error) error {
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
				if err := handle(msg); err != nil {
					log.Printf("Error handling stat record: %v", err)
				}
			}
		})
	}
}
