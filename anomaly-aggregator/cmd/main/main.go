package main

import (
	"anomaly-aggregator/internal/config"
	"anomaly-aggregator/internal/consumers"
	"context"
	"log"
	"time"
)

func main() {
	cfg, err := config.SetupConfig()
	if err != nil {
		log.Panicf("Could not setup configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	rawEventCons := consumers.NewRawEventConsumer(cfg.KafkaRaw, cfg.Pg)
	go func() {
		if err := rawEventCons.ConsumeTopic(ctx, rawEventCons.InsertEvent); err != nil {
			log.Printf("Raw events consumer error: %v", err)
		}
	}()
	<-ctx.Done()
	log.Println("Context canceled, shutting down gracefully")

}
