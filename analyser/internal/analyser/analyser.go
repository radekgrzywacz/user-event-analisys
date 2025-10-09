package analyser

import (
	"analyser/internal/event"
	"analyser/internal/store"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type AnalyseResult struct {
	Anomaly   bool   `json:"anomaly"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// TODO: Dodać do redisa:
// 1. Zbyt częste logowania
// 2. Szybkość zmian eventów
// 3. Odległość geograficzna po IP
// 4. Za duzo failed loginow

func Process(event event.Event, rdb *redis.Client, pg *store.Queries) error {
	// result, err := analyseCached(event, rdb)
	// if err != nil {
	// 	return fmt.Errorf("Error analysing cached data: %w", err)
	// }
	// if result.Anomaly {
	// 	// TODO: Process anomaly
	// }
	// result, err = analyseStatistics(event, rdb)
	// if err != nil {
	// 	return fmt.Errorf("%w", err)
	// }
	// if result.Anomaly {
	// 	// TODO: Process anomaly
	// }

	if err := addEventToRedis(event, rdb); err != nil {
		return fmt.Errorf("Redis insert failed: %w", err)
	}
	if err := recordTransition(event, rdb); err != nil {
		return fmt.Errorf("Redis transition record failed: %w", err)
	}

	if _, err := saveEventToPostgres(pg, event); err != nil {
		return fmt.Errorf("Postgres insert failed: %w", err)
	}

	return nil
}
