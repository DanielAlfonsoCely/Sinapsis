package audit

import (
	"context"

	"sinapsis-backend/models"
)

// Observer es cualquier componente que reacciona a un evento auditable.
type Observer interface {
	Notify(ctx context.Context, entry models.AuditLogEntry)
}

// Publisher mantiene la lista de observers y los notifica a todos.
type Publisher struct {
	observers []Observer
}

func NewPublisher(observers ...Observer) *Publisher {
	return &Publisher{observers: observers}
}

func (p *Publisher) Publish(ctx context.Context, entry models.AuditLogEntry) {
	for _, o := range p.observers {
		o.Notify(ctx, entry)
	}
}
