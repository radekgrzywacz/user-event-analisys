package main

import (
	"analyser/internal/analyser"
	"analyser/internal/config"
	"analyser/internal/event"
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/twmb/franz-go/pkg/kgo"
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
	commitChan := make(chan *kgo.Record, 1000)

	go commitLoop(ctx, config.Kafka, commitChan)

	for {
		fetches := config.Kafka.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				record := record
				go func() {
					event, err := event.ParseEvent(record.Value)
					if err != nil {
						log.Println("error parsing record")
						return
					}
					err = analyser.Process(event, config.Redis)
					if err != nil {
						log.Printf("Processing error: %v", err)
						return
					}

					commitChan <- record
				}()
			}
		})
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
			toCommit = append(toCommit, record)
		case <-ticker.C:
			if len(toCommit) > 0 {
				if err := client.CommitRecords(ctx, toCommit...); err != nil {
					log.Printf("Commit error: %v", err)
				}
				toCommit = nil
			}
		}
	}
}
