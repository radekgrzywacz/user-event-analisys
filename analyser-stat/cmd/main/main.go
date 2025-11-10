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
	WorkerCount   = 20   // liczba gorutyn przetwarzających rekordy
	JobBufferSize = 1000 // bufor kanału dla rekordów
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
	config, err := config.SetupConfig()
	if err != nil {
		log.Panic(err)
	}
	defer config.Kafka.Close()
	defer config.Redis.Close()

	ctx := context.Background()
	jobs := make(chan *kgo.Record, JobBufferSize)
	commitChan := make(chan *kgo.Record, JobBufferSize)

	producer := processor.NewProducer(config.Kafka, "analyser-result")

	registry := processor.NewRegistry(
		useractivity.NewHandler(),
	)

	for i := range WorkerCount {
		go worker(ctx, i, jobs, commitChan, registry, producer, config)
	}

	go commitLoop(ctx, config.Kafka, commitChan)

	log.Printf("Analyser started with %d workers", WorkerCount)

	for {
		fetches := config.Kafka.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				select {
				case jobs <- record:
					// record passed to worker
				case <-ctx.Done():
					return
				default:
					log.Printf("Job channel full, dropping record (topic=%s, partition=%d)", p.Topic, p.Partition)
				}
			}
		})
	}
}

func worker(ctx context.Context, id int, jobs <-chan *kgo.Record, commitChan chan<- *kgo.Record, registry *processor.Registry, producer *processor.Producer, cfg *config.Config) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("[worker %d] stopping", id)
			return
		case record := <-jobs:
			if record == nil {
				continue
			}

			envelope, err := contracts.ParseEnvelope(record.Value)
			if err != nil {
				log.Printf("[worker %d] error parsing envelope: %v", id, err)
				continue
			}

			if err := registry.Handle(envelope, cfg.Redis, producer); err != nil {
				if errors.Is(err, processor.ErrUnknownDomain) {
					log.Printf("[worker %d] skipping unsupported domain %q", id, envelope.Domain)
					commitChan <- record
					continue
				}
				log.Printf("[worker %d] processing error: %v", id, err)
				continue
			}

			commitChan <- record
		}
	}
}

func commitLoop(ctx context.Context, client *kgo.Client, commitChan chan *kgo.Record) {
	var toCommit []*kgo.Record
	ticker := time.NewTicker(3 * time.Second)
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
