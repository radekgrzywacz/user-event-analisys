package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	contracts "user-event-analisys/contracts/events"

	"synthetic-data-generator/internal/adapters"
	"synthetic-data-generator/internal/env"
	"synthetic-data-generator/internal/user"

	"github.com/brianvoe/gofakeit/v7"
)

type Generator struct {
	userService *user.UserService
	faker       *gofakeit.Faker
}

func NewGenerator(us *user.UserService, faker *gofakeit.Faker) *Generator {
	return &Generator{
		userService: us,
		faker:       faker,
	}
}

func (g *Generator) CreateGoodEvent(userId int, eventType EventType) (Event, error) {
	user, err := g.userService.GetUserById(userId)
	if err != nil {
		return Event{}, fmt.Errorf("Failed to get user: %w", err)
	}

	event := Event{
		UserId:    user.ID,
		Type:      eventType,
		Timestamp: time.Now(),
		Metadata: Metadata{
			IP:        user.IP,
			UserAgent: user.UserAgent,
			Country:   user.Country,
		},
		Additional: make(map[string]interface{}),
	}

	switch eventType {
	case EventPayment:
		event.Additional["value"] = g.faker.Price(10, 10000)
		event.Additional["currency"] = "EUR"
		event.Additional["merchant"] = g.faker.Company()
	case EventLogin:
		event.Additional["device"] = g.faker.AppName()
		event.Additional["source"] = []string{"WEB", "MOBILE"}[g.faker.Number(0, 1)]
	case EventLogout:
		event.Additional["duration"] = g.faker.Number(30, 600)
	default:
		event.Additional["source"] = "WEB"
	}

	return event, nil
}

func (g *Generator) CreateRandomEvent(userId int, eventType EventType) (Event, error) {
	user, err := g.userService.GetUserById(userId)
	if err != nil {
		return Event{}, fmt.Errorf("Failed to get user: %w", err)
	}

	event := Event{
		UserId:    user.ID,
		Type:      eventType,
		Timestamp: time.Now(),
		Metadata: Metadata{
			IP:        g.faker.IPv4Address(),
			UserAgent: g.faker.UserAgent(),
			Country:   g.faker.Country(),
		},
		Additional: make(map[string]interface{}),
	}

	switch eventType {
	case EventPayment:
		event.Additional["value"] = g.faker.Price(1000, 100000)
		event.Additional["currency"] = g.faker.CurrencyShort()
		event.Additional["merchant"] = g.faker.Company()
	case EventLogin:
		event.Additional["device"] = g.faker.AppName()
		event.Additional["source"] = []string{"WEB", "API", "TOR", "VPN"}[g.faker.Number(0, 3)]
	case EventLogout:
		event.Additional["duration"] = g.faker.Number(0, 5) // podejrzanie krótka sesja
	default:
		event.Additional["source"] = "UNKNOWN"
	}

	return event, nil
}

func (g *Generator) SendEvent(event Event) error {
	payload := contracts.UserActivityPayload{
		UserID:     event.UserId,
		Type:       contracts.ActivityType(event.Type),
		Timestamp:  event.Timestamp,
		SessionID:  event.SessionId,
		Metadata:   contracts.UserMetadata(event.Metadata),
		Additional: event.Additional,
	}

	envelope, err := adapters.UserActivityToEnvelope(payload, event.SessionId, nil)
	if err != nil {
		return fmt.Errorf("could not build envelope: %w", err)
	}

	e, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("could not marshal envelope: %w", err)
	}

	body := bytes.NewBuffer(e)
	url := fmt.Sprintf("http://%s:%s/ingestor", env.GetEnvString("INGESTOR_URL", "http-ingestor"), env.GetEnvString("INGESTOR_PORT", "8081"))

	resp, err := http.Post(url, "application/json; charset=utf-8", body)
	if err != nil {
		return fmt.Errorf("could not send an event: %w", err)
	}

	defer resp.Body.Close()

	return nil
}
