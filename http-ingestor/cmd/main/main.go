package main

import (
	"context"
	"errors"
	"fmt"
	"http-ingestor/internal/env"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/twmb/franz-go/pkg/kgo"
)

type App struct {
	client *kgo.Client
}

// Getting proper env - docker or .env
func init() {
	if os.Getenv("RUNNING_IN_DOCKER") == "" {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Println("No .env file found (this is fine in Docker)")
		}
	}
}

func setupKafka() (*kgo.Client, error) {
	kafka := env.GetEnvString("KAFKA_URL", "asd")
	cl, err := kgo.NewClient(kgo.SeedBrokers(kafka),
		kgo.DefaultProduceTopic("events"))
	if err != nil {
		return nil, fmt.Errorf("Could not create a kafka client: %d", err)
	}

	return cl, nil
}

func (app *App) getDataForKafka(w http.ResponseWriter, r *http.Request) {
	cl, err := setupKafka()
	if err != nil {
		panic(err)
	}
	defer cl.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	app.client.Produce(ctx, &kgo.Record{
		Topic: "events",
		Value: body,
	}, func(_ *kgo.Record, err error) {
		defer wg.Done()
		if err != nil {
			log.Printf("Kafka produce error: %v", err)
		}
	})
	wg.Wait()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, `{"message": "Data received successfully"}`)
}

func healthcheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, `{"message": "Healthchecked successfully"}`)
}

func main() {
	client, err := setupKafka()
	if err != nil {
		log.Fatalf("Error creating kafka client: %v", err)
	}

	app := App{client}

	http.HandleFunc("/ingestor", app.getDataForKafka)
	http.HandleFunc("/healthcheck", healthcheck)

	log.Print("Ingestor server starting...")
	err = http.ListenAndServe(":8081", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
