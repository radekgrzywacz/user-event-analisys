package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type ResultEvent struct {
	UserId      int    `json:"user_id"`
	SessionId   string `json:"session_id"`
	EventType   string `json:"event_type"`
	Anomaly     bool   `json:"anomaly"`
	AnomalyType string `json:"anomaly_type"`
	Message     string `json:"message"`
	Timestamp   string `json:"timestamp"`
	Source      string `json:"source"`
}

type Producer struct {
	client *kgo.Client
	topic  string
}

func NewProducer(client *kgo.Client, topic string) *Producer {
	return &Producer{client: client, topic: topic}
}

func (p *Producer) PublishResult(ctx context.Context, result ResultEvent) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("Kafka publish: marshal error: %v", err)
	}

	record := &kgo.Record{
		Topic:     p.topic,
		Value:     data,
		Timestamp: time.Now(),
	}

	err = p.client.ProduceSync(ctx, record).FirstErr()
	if err != nil {
		return fmt.Errorf("Kafka publish error: %v", err)
	} else {
		log.Printf("Published result for user %d (%s) to topic %s", result.UserId, result.EventType, p.topic)
		return nil
	}
}
