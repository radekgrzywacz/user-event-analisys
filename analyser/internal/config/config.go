package config

import (
	"analyser/internal/env"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Config struct {
	Kafka *kgo.Client
}

func setupKafka() (*kgo.Client, error) {
	broker := env.GetEnvString("KAFKA_URL", "localhost:9092")

	cl, err := kgo.NewClient(kgo.SeedBrokers(broker), kgo.ConsumeTopics("events"))
	if err != nil {
		return nil, fmt.Errorf("Unable to create consumer client: %v", err)
	}

	return cl, nil
}

func SetupConfig() (*Config, error) {
	kafka, err := setupKafka()
	if err != nil {
		return nil, fmt.Errorf("Error configuring the app: %w", err)
	}

	return &Config{
		Kafka: kafka,
	}, nil
}
