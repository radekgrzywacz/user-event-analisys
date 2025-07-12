package main

import (
	"context"
	"fmt"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

func main() {
	brokers := "localhost:9092"

	cl, err := kgo.NewClient(kgo.SeedBrokers(brokers), kgo.ConsumeTopics("events"))
	if err != nil {
		log.Fatalf("Unable to create consumer client: %v", err)
	}
	defer cl.Close()

	ctx := context.Background()

	for {
		fetches := cl.PollFetches(ctx)
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			for _, record := range p.Records {
				fmt.Printf("Received message: topic=%s key %s value %s offset %s\n",
					record.Topic, string(record.Key), string(record.Value), fmt.Sprint(record.Offset))
			}
		})
	}
}
