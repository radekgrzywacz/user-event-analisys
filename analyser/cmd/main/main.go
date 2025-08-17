package main

import (
	"analyser/internal/config"
	"context"
	"fmt"
	"log"
	"os"

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

	ctx := context.Background()

	for {
		fetches := config.Kafka.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				fmt.Printf("Received message: topic=%s key %s value %s offset %s\n",
					record.Topic, string(record.Key), string(record.Value), fmt.Sprint(record.Offset))
			}
		})
	}

}
