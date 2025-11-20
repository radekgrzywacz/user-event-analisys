package config

import (
	"anomaly-aggregator/internal/env"
	"anomaly-aggregator/internal/store"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	KafkaStat *kgo.Client
	KafkaMl   *kgo.Client
	KafkaRaw  *kgo.Client
	Pg        *store.Queries
}

func setupPostgres() (*store.Queries, error) {
	url := env.GetEnvString("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/user_event_analysis_db?sslmode=disable")

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to PostgreSQL: %w", err)
	}

	return store.New(pool), nil
}

func setupKafka(broker, topic, group string) (*kgo.Client, error) {
	cl, err := kgo.NewClient(kgo.SeedBrokers(broker),
		kgo.ConsumeTopics(topic),
		kgo.ConsumerGroup(group),
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to create consumer client: %v", err)
	}

	return cl, nil
}

func SetupConfig() (Config, error) {
	broker := env.GetEnvString("KAFKA_URL", "localhost:9092")
	topic := env.GetEnvString("KAFKA_TOPIC_STAT", "stat_out")
	group := env.GetEnvString("KAFKA_CONSUMER_GROUP", "aggregator")
	stat, err := setupKafka(broker, topic, group)
	if err != nil {
		return Config{}, fmt.Errorf("Could not set up Kafka Stat consumer: %w", err)
	}

	topic = env.GetEnvString("KAFKA_TOPIC_STAT", "ml_out")
	ml, err := setupKafka(broker, topic, group)
	if err != nil {
		return Config{}, fmt.Errorf("Could not set up Kafka Ml consumer: %w", err)
	}

	topic = env.GetEnvString("KAFKA_TOPIC_STAT", "events")
	raw, err := setupKafka(broker, topic, group)
	if err != nil {
		return Config{}, fmt.Errorf("Could not set up Kafka Ml consumer: %w", err)
	}

	pg, err := setupPostgres()
	if err != nil {
		return Config{}, fmt.Errorf("Could not set up Postgres: %w", err)
	}

	return Config{stat, ml, raw, pg}, nil
}
