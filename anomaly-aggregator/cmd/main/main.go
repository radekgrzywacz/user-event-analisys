package main

import (
	"anomaly-aggregator/internal/aggregator"
	"anomaly-aggregator/internal/config"
	"anomaly-aggregator/internal/consumers"
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.SetupConfig()
	if err != nil {
		log.Fatalf("Could not setup configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		log.Println("[Fusion] Shutdown signal received — stopping all consumers…")
		cancel()
	}()

	agg := aggregator.NewAggregator(cfg.Pg)
	agg.StartCleanup(5 * time.Minute)

	rawCons := consumers.NewRawEventConsumer(cfg.KafkaRaw, cfg.Pg)
	statCons := consumers.NewStatConsumer(cfg.KafkaStat, cfg.Pg, agg)
	mlCons := consumers.NewMLConsumer(cfg.KafkaMl, cfg.Pg, agg)

	wg := &sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		if err := rawCons.ConsumeTopic(ctx, rawCons.InsertEvent); err != nil {
			log.Printf("[RawConsumer] error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := statCons.ConsumeTopic(ctx); err != nil {
			log.Printf("[StatConsumer] error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := mlCons.ConsumeTopic(ctx); err != nil {
			log.Printf("[MLConsumer] error: %v", err)
		}
	}()

	log.Println("[Aggregator] Aggregator with cache started. Waiting for messages…")
	wg.Wait()
	log.Println("[Aggregator] Graceful shutdown complete.")
}
