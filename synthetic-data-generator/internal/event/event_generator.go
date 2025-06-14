package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"synthetic-data-generator/internal/env"
	"synthetic-data-generator/internal/user"
	"time"

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
	}

	return event, nil
}

func (g *Generator) SendEvent(event Event) error {
	e, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("Could not marshal an event: %w", err)
	}

	body := bytes.NewBuffer(e)
	url := fmt.Sprintf("http://%s:%s/ingestor", env.GetEnvString("INGESTOR_URL", "http-ingestor"), env.GetEnvString("INGESTOR_PORT", "8081"))

	resp, err := http.Post(url, "application/json; charset=utf-8", body)
	if err != nil {
		return fmt.Errorf("Could not send an event: %w", err)
	}

	defer resp.Body.Close()

	log.Printf("Status received from server is: %s", resp.Status)
	log.Printf("StatusCode received from server is: %d", resp.StatusCode)
	log.Printf("Content Type received from Server is: %s", resp.Header["Content-Type"][0])

	return nil
}
