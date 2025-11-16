package main

import (
	"analyser/internal/config"
	"analyser/internal/domain/useractivity"
	"analyser/internal/processor"
	"context"
	"errors"
	"log"
	"os"
	"time"

	contracts "user-event-analisys/contracts/events"

	"github.com/joho/godotenv"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	CommitInterval = 3 * time.Second
	JobBufferSize  = 1000
)

func init() {
	if os.Getenv("RUNNING_IN_DOCKER") == "" {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Println("No .env file found (this is fine in Docker)")
		}
	}
}

func main() {
	cfg, err := config.SetupConfig()
	if err != nil {
		log.Panic(err)
	}
	defer cfg.Kafka.Close()
	defer cfg.Redis.Close()

	ctx := context.Background()

	commitChan := make(chan *kgo.Record, JobBufferSize)
	producer := processor.NewProducer(cfg.Kafka, "analyser-result")

	registry := processor.NewRegistry(
		useractivity.NewHandler(),
	)

	go commitLoop(ctx, cfg.Kafka, commitChan)

	log.Println("Analyser-Stat started (sequential mode, Kafka keyed by user_id)")

	for {
		fetches := cfg.Kafka.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				envelope, err := contracts.ParseEnvelope(record.Value)
				if err != nil {
					log.Printf("error parsing envelope: %v", err)
					continue
				}

				if err := registry.Handle(envelope, cfg.Redis, producer); err != nil {
					if errors.Is(err, processor.ErrUnknownDomain) {
						log.Printf("skipping unsupported domain %q", envelope.Domain)
						commitChan <- record
						continue
					}
					log.Printf("Processing error: %v", err)
					continue
				}

				// Record processed successfully — mark for commit
				commitChan <- record
			}
		})
	}
}

func commitLoop(ctx context.Context, client *kgo.Client, commitChan chan *kgo.Record) {
	var toCommit []*kgo.Record
	ticker := time.NewTicker(CommitInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case record := <-commitChan:
			if record != nil {
				toCommit = append(toCommit, record)
			}
		case <-ticker.C:
			if len(toCommit) > 0 {
				if err := client.CommitRecords(ctx, toCommit...); err != nil {
					log.Printf("Commit error: %v", err)
				} else {
					log.Printf("Committed %d records", len(toCommit))
				}
				toCommit = nil
			}
		}
	}
}
