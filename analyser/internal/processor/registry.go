package processor

import (
	"errors"
	"fmt"

	"analyser/internal/store"

	"github.com/redis/go-redis/v9"
	contracts "user-event-analisys/contracts/events"
)

var ErrUnknownDomain = errors.New("unknown domain")

type Handler interface {
	Domain() string
	Handle(envelope contracts.Envelope, rdb *redis.Client, pg *store.Queries) error
}

type Registry struct {
	handlers map[string]Handler
}

func NewRegistry(handlers ...Handler) *Registry {
	reg := &Registry{handlers: make(map[string]Handler, len(handlers))}
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		reg.handlers[handler.Domain()] = handler
	}
	return reg
}

func (r *Registry) Handle(envelope contracts.Envelope, rdb *redis.Client, pg *store.Queries) error {
	handler, ok := r.handlers[envelope.Domain]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownDomain, envelope.Domain)
	}
	return handler.Handle(envelope, rdb, pg)
}
