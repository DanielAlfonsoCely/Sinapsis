package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"sinapsis-backend/models"
)

// Publisher publica mensajes AnalysisRequest a la cola de solicitudes de IA.
type Publisher struct {
	ch           *amqp.Channel
	requestQueue string
}

// NewPublisher crea un Publisher sobre el canal AMQP ya abierto.
func NewPublisher(ch *amqp.Channel, requestQueue string) *Publisher {
	return &Publisher{ch: ch, requestQueue: requestQueue}
}

// Publish serializa req como JSON y lo publica a la cola de solicitudes.
// delivery_mode=Persistent para que RabbitMQ sobreviva reinicios.
func (p *Publisher) Publish(ctx context.Context, req models.AnalysisRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal AnalysisRequest: %w", err)
	}

	if err := p.ch.PublishWithContext(ctx,
		"",             // exchange vacío → publica directamente a la cola
		p.requestQueue, // routing key = nombre de la cola
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish AnalysisRequest: %w", err)
	}

	return nil
}
