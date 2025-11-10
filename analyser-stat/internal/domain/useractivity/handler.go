package useractivity

import (
	"fmt"

	"analyser/internal/event"
	"analyser/internal/processor"

	"github.com/redis/go-redis/v9"
	contracts "user-event-analisys/contracts/events"
)

type Handler struct {
	domain          string
	canonicalDomain string
}

func NewHandler() Handler {
	return Handler{
		domain:          contracts.DomainUserActivity,
		canonicalDomain: contracts.DomainUserActivity,
	}
}

func NewHandlerWithDomain(domain string) Handler {
	if domain == "" {
		domain = contracts.DomainUserActivity
	}
	return Handler{
		domain:          domain,
		canonicalDomain: contracts.DomainUserActivity,
	}
}

func (h Handler) Domain() string {
	return h.domain
}

func (h Handler) Handle(envelope contracts.Envelope, rdb *redis.Client, producer *processor.Producer) error {
	envelope.Domain = h.canonicalDomain

	ev, err := event.ParseEvent(envelope)
	if err != nil {
		return fmt.Errorf("parse user activity payload: %w", err)
	}

	return Process(ev, rdb, producer)

}
