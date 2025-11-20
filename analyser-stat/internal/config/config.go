package config

import (
	"analyser/internal/env"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	Kafka *kgo.Client
	Redis *redis.Client
	// Pg    *store.Queries
}

func setupKafka() (*kgo.Client, error) {
	broker := env.GetEnvString("KAFKA_URL", "localhost:9092")
	topic := env.GetEnvString("KAFKA_TOPIC", "events")
	group := env.GetEnvString("KAFKA_CONSUMER_GROUP", "analyser-stat")

	cl, err := kgo.NewClient(kgo.SeedBrokers(broker),
		kgo.ConsumeTopics(topic),
		kgo.ConsumerGroup(group),
	)
	if err != nil {
		return nil, fmt.Errorf("Unable to create consumer client: %v", err)
	}

	return cl, nil
}

func setupRedis() *redis.Client {
	url := env.GetEnvString("REDIS_URL", "localhost:6379")
	return redis.NewClient(&redis.Options{
		Addr: url,
		DB:   0,
	})
}

func SetupConfig() (*Config, error) {
	kafka, err := setupKafka()
	if err != nil {
		return nil, fmt.Errorf("Error configuring the app: %w", err)
	}

	redis := setupRedis()
	// pg, err := setupPostgres()
	// if err != nil {
	// 	return nil, fmt.Errorf("Error setting up Postgres: %w", err)
	// }

	return &Config{
		Kafka: kafka,
		Redis: redis,
		// Pg:    pg,
	}, nil
}
