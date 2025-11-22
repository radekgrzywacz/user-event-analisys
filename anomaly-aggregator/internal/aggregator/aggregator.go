package aggregator

import (
	"context"
	"log"
	"sync"
	"time"

	"anomaly-aggregator/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
)

type Aggregator struct {
	db    *store.Queries
	cache *AggregatorCache
}

type AggregatorCache struct {
	mu   sync.Mutex
	data map[string]*AggregatorEntry
}

type AggregatorEntry struct {
	ML        *MLResult
	Stat      []StatResult
	LastEvent time.Time
}

type MLResult struct {
	UserID       int
	SessionID    string
	Anomaly      bool
	Score        float64
	Threshold    float64
	EventCount   int
	UniqueEvents int
}

type StatResult struct {
	UserID      int
	SessionID   string
	Anomaly     bool
	AnomalyType string
	Timestamp   time.Time
}

// Konstruktor
func NewAggregator(db *store.Queries) *Aggregator {
	return &Aggregator{
		db: db,
		cache: &AggregatorCache{
			data: make(map[string]*AggregatorEntry),
		},
	}
}

// Odbiór danych z ML
func (f *Aggregator) OnMLResult(ctx context.Context, res MLResult) error {
	f.cache.mu.Lock()
	entry, ok := f.cache.data[res.SessionID]
	if !ok {
		entry = &AggregatorEntry{}
		f.cache.data[res.SessionID] = entry
	}
	entry.ML = &res
	entry.LastEvent = time.Now()
	f.cache.mu.Unlock()

	return f.tryAggregate(ctx, res.SessionID)
}

// Odbiór danych ze Stat
func (f *Aggregator) OnStatResult(ctx context.Context, res StatResult) error {
	f.cache.mu.Lock()
	entry, ok := f.cache.data[res.SessionID]
	if !ok {
		entry = &AggregatorEntry{}
		f.cache.data[res.SessionID] = entry
	}
	entry.Stat = append(entry.Stat, res)
	entry.LastEvent = time.Now()
	f.cache.mu.Unlock()

	return f.tryAggregate(ctx, res.SessionID)
}

// Próba połączenia danych (jeśli obie części już są)
func (f *Aggregator) tryAggregate(ctx context.Context, sessionID string) error {
	f.cache.mu.Lock()
	entry, ok := f.cache.data[sessionID]
	f.cache.mu.Unlock()
	if !ok || entry.ML == nil || len(entry.Stat) == 0 {
		return nil // niepełne dane – czekamy
	}

	// Agregacja statystyczna
	var anyStatAnomaly bool
	anomalyTypes := make(map[string]struct{})
	for _, s := range entry.Stat {
		if s.Anomaly {
			anyStatAnomaly = true
			anomalyTypes[s.AnomalyType] = struct{}{}
		}
	}

	var anomalyList []string
	for k := range anomalyTypes {
		anomalyList = append(anomalyList, k)
	}

	params := store.InsertAggregatedResultParams{
		SessionID:    entry.ML.SessionID,
		UserID:       int32(entry.ML.UserID),
		MlAnomaly:    pgtype.Bool{Bool: entry.ML.Anomaly, Valid: true},
		MlScore:      pgtype.Float8{Float64: entry.ML.Score, Valid: true},
		MlThreshold:  pgtype.Float8{Float64: entry.ML.Threshold, Valid: true},
		StatAnomaly:  pgtype.Bool{Bool: anyStatAnomaly, Valid: true},
		AnomalyType:  pgtype.Text{String: joinStrings(anomalyList, ", "), Valid: len(anomalyList) > 0},
		EventCount:   pgtype.Int4{Int32: int32(entry.ML.EventCount), Valid: true},
		UniqueEvents: pgtype.Int4{Int32: int32(entry.ML.UniqueEvents), Valid: true},
	}

	id, err := f.db.InsertAggregatedResult(ctx, params)
	if err != nil {
		return err
	}
	log.Printf("[Aggregator] Aggregated saved id=%d for session=%s", id, sessionID)

	// czyścimy z cache po zapisaniu
	f.cache.mu.Lock()
	delete(f.cache.data, sessionID)
	f.cache.mu.Unlock()
	return nil
}

func (f *Aggregator) StartCleanup(interval time.Duration) {
	go func() {
		for range time.Tick(interval) {
			now := time.Now()
			f.cache.mu.Lock()
			for sid, e := range f.cache.data {
				if now.Sub(e.LastEvent) > 5*time.Minute {
					delete(f.cache.data, sid)
				}
			}
			f.cache.mu.Unlock()
		}
	}()
}

func joinStrings(list []string, sep string) string {
	out := ""
	for i, v := range list {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}
