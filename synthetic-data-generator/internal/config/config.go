package config

import (
	"database/sql"
	"flag"
	"log"
	"synthetic-data-generator/internal/db"
	"synthetic-data-generator/internal/env"
)

type Config struct {
	Flags Flags
	DB   *sql.DB
}

type Flags struct {
	UsersCount        int
	DurationInSeconds int
	Concurrency       int
	AnomalyRate       float64
	Endpoint          string
}

func SetupConfig() Config {
	db, err := db.Connect()
	if err != nil {
		log.Fatalf("Error while connecting to the database: %q", err)
	}

	return Config{
		Flags: parseFlags(),
		DB:    db,
	}
}

func parseFlags() Flags {
	var flags Flags

	flag.IntVar(&flags.UsersCount, "users", 20, "Number of users to simulate")
	flag.IntVar(&flags.DurationInSeconds, "duration", 120, "Duration of the simulation in seconds")
	flag.IntVar(&flags.Concurrency, "concurrency", 5, "Number of concurrent simulated events")
	flag.Float64Var(&flags.AnomalyRate, "anomaly-rate", 0.2, "Fraction of events that are anomalies (0.0 - 1.0)")
	flag.StringVar(&flags.Endpoint, "endpoint", env.GetEnvString("INGESTOR_API_URL", "http://localhost:8080/events"), "API endpoint to send events to")

	flag.Parse()

	if flags.AnomalyRate < 0.0 || flags.AnomalyRate > 1.0 {
		log.Fatal("Anomaly rate must be between 0.0 and 1.0!")
	}

	return flags
}
