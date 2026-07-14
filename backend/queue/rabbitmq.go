// Package queue gestiona la conexión AMQP con RabbitMQ y declara la topología
// de intercambios/colas que usa el microservicio de IA (sinapsis.ai).
//
// Topología (espejo de microservice/src/sinapsis_ai/main.py):
//
//	Exchange sinapsis.ai (topic) ← el microservicio publica AnalysisResult aquí
//	Cola   ai.analysis.requests  ← el backend publica AnalysisRequest aquí
//	Cola   backend.ai.results    ← el backend consume AnalysisResult desde aquí
//	  bind: sinapsis.ai / ai.analysis.result
package queue

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection envuelve la conexión AMQP y el canal activo.
type Connection struct {
	conn    *amqp.Connection
	Channel *amqp.Channel
}

// Connect establece la conexión AMQP, abre un canal y declara la topología
// necesaria para el flujo backend ↔ microservicio IA.
//
// Nota: la cola de solicitudes (ai.analysis.requests) la declara el microservicio
// Python con su DLX; el backend solo publica en ella sin declararla.
func Connect(url, resultExchange, resultRoutingKey, resultQueue string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", err)
	}

	// Exchange de resultados (el microservicio publica aquí; el backend consume).
	// El microservicio lo declara como "direct" (main.py:64-68), mantenemos consistencia.
	if err := ch.ExchangeDeclare(
		resultExchange,
		"direct",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare result exchange %q: %w", resultExchange, err)
	}

	// Cola propia del backend para consumir resultados.
	if _, err := ch.QueueDeclare(
		resultQueue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare result queue %q: %w", resultQueue, err)
	}

	// Bind: resultados publicados por el microservicio llegan a la cola del backend.
	if err := ch.QueueBind(
		resultQueue,
		resultRoutingKey,
		resultExchange,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("bind result queue: %w", err)
	}

	log.Printf("amqp: conectado a %s (exchange=%s, resultQueue=%s)", url, resultExchange, resultQueue)
	return &Connection{conn: conn, Channel: ch}, nil
}

// Close cierra el canal y la conexión AMQP.
func (c *Connection) Close() {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
