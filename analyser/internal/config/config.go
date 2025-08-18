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
}

func setupKafka() (*kgo.Client, error) {
	broker := env.GetEnvString("KAFKA_URL", "localhost:9092")

	cl, err := kgo.NewClient(kgo.SeedBrokers(broker), kgo.ConsumeTopics("events"))
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

	return &Config{
		Kafka: kafka,
		Redis: redis,
	}, nil
}
