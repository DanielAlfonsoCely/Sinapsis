package queue

import (
	"context"
	"encoding/json"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"sinapsis-backend/models"
)

// Consumer escucha la cola de resultados IA y persiste cada AnalysisResult
// en la tabla sugerencia_ia.
type Consumer struct {
	ch          *amqp.Channel
	resultQueue string
	pool        *pgxpool.Pool
}

// NewConsumer crea un Consumer con el canal AMQP y el pool de BD.
func NewConsumer(ch *amqp.Channel, resultQueue string, pool *pgxpool.Pool) *Consumer {
	return &Consumer{ch: ch, resultQueue: resultQueue, pool: pool}
}

// Start arranca el consumo en goroutine. Retorna cuando ctx es cancelado o la
// entrega del canal AMQP se cierra.
func (c *Consumer) Start(ctx context.Context) {
	msgs, err := c.ch.Consume(
		c.resultQueue,
		"sinapsis-backend", // consumer tag
		false,              // auto-ack → manejamos ack manualmente
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,
	)
	if err != nil {
		log.Printf("amqp consumer: error al registrar consumer: %v", err)
		return
	}

	log.Printf("amqp consumer: escuchando resultados IA en cola %q", c.resultQueue)

	for {
		select {
		case <-ctx.Done():
			log.Println("amqp consumer: contexto cancelado, deteniendo consumer")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Println("amqp consumer: canal cerrado")
				return
			}
			c.handle(ctx, msg)
		}
	}
}

// handle procesa un mensaje AnalysisResult individual.
func (c *Consumer) handle(ctx context.Context, msg amqp.Delivery) {
	var result models.AnalysisResult
	if err := json.Unmarshal(msg.Body, &result); err != nil {
		log.Printf("amqp consumer: JSON inválido, descartando mensaje: %v", err)
		msg.Nack(false, false) // dead-letter, no reencolar
		return
	}

	if err := c.persist(ctx, result); err != nil {
		log.Printf("amqp consumer: error persistiendo resultado request_id=%s: %v", result.RequestID, err)
		msg.Nack(false, true) // reencolar para reintento
		return
	}

	msg.Ack(false)
	log.Printf("amqp consumer: resultado procesado request_id=%s status=%s duration_ms=%d",
		result.RequestID, result.Status, result.DurationMs)
}

// persist actualiza la fila de sugerencia_ia correspondiente al request_id.
// El handler POST ya insertó la fila con estado_procesamiento='enviado';
// aquí la actualizamos con el resultado del microservicio.
func (c *Consumer) persist(ctx context.Context, r models.AnalysisResult) error {
	// Serializar metricas + artifacts como JSONB combinado.
	metricas := map[string]any{
		"metrics":   r.Metrics,
		"artifacts": r.Artifacts,
	}
	metricasJSON, err := json.Marshal(metricas)
	if err != nil {
		return err
	}

	estado := "completado"
	if r.Status == "failed" {
		estado = "fallido"
	}

	// modelo_ia_utilizado: usar nombre del bundle tal como lo devuelve el microservicio.
	modelo := r.Model.Name
	if r.Model.Version != "" {
		modelo += " v" + r.Model.Version
	}

	_, err = c.pool.Exec(ctx, `
		UPDATE sugerencia_ia
		SET
			estado_procesamiento = $1,
			modelo_ia_utilizado  = $2,
			metricas             = $3,
			fecha_analisis       = $4
		WHERE request_id = $5
	`, estado, modelo, metricasJSON, r.ProcessedAt, r.RequestID)

	return err
}
