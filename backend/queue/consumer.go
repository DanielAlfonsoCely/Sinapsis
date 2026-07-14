package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

	descripcion, diagnostico, confianza := deriveHallazgos(r.Model.Name, r.Metrics)

	_, err = c.pool.Exec(ctx, `
		UPDATE sugerencia_ia
		SET
			estado_procesamiento = $1,
			modelo_ia_utilizado  = $2,
			metricas             = $3,
			fecha_analisis       = $4,
			descripcion_hallazgo = $5,
			diagnostico_sugerido = $6,
			confianza_prediccion = $7
		WHERE request_id = $8
	`, estado, modelo, metricasJSON, r.ProcessedAt, descripcion, diagnostico, confianza, r.RequestID)

	return err
}

// deriveHallazgos traduce las métricas crudas del microservicio (que varían
// por tipo de bundle -- ver microservice/.../output_extractors.py) a un
// resumen legible para el médico. Se guarda en columnas propias de
// sugerencia_ia para que la historia clínica pueda mostrarlo sin tener que
// interpretar el JSON de metricas.
//
// Campos reales por tipo de bundle:
//   - Segmentación (bazo, tumor cerebral): { volume_voxels }
//   - Detección de nódulo pulmonar:        { lesion_count, max_diameter_mm }
//   - Clasificación (densidad mamaria):    { predicted_class, probability }
func deriveHallazgos(modelName string, metrics map[string]any) (descripcion, diagnostico *string, confianza *float64) {
	if metrics == nil {
		return nil, nil, nil
	}

	if voxels, ok := asFloat(metrics["volume_voxels"]); ok {
		esBazo := strings.Contains(modelName, "spleen")
		var d, dg string
		if esBazo {
			d = fmt.Sprintf("Segmentación de bazo completada: %.0f vóxeles identificados en la región segmentada.", voxels)
			dg = "Segmentación de bazo (MONAI spleen_ct_segmentation)"
		} else {
			d = fmt.Sprintf("Segmentación completada: %.0f vóxeles identificados en la región segmentada.", voxels)
			dg = "Segmentación de tumor cerebral (MONAI brats_mri_segmentation)"
		}
		return &d, &dg, nil
	}

	if lesionCount, ok := asFloat(metrics["lesion_count"]); ok {
		maxDiameter, _ := asFloat(metrics["max_diameter_mm"])
		var d, dg string
		if lesionCount > 0 {
			d = fmt.Sprintf("%.0f nódulo(s) pulmonar(es) detectado(s), diámetro máximo %.1f mm.", lesionCount, maxDiameter)
			dg = "Hallazgo nodular pulmonar sugerido por IA"
		} else {
			d = "No se detectaron nódulos pulmonares en la imagen analizada."
			dg = "Sin hallazgos nodulares pulmonares"
		}
		return &d, &dg, nil
	}

	if predictedClass, ok := metrics["predicted_class"].(string); ok {
		probability, _ := asFloat(metrics["probability"])
		conf := probability * 100
		d := fmt.Sprintf("Densidad mamaria clasificada como categoría %s (confianza %.0f%%).", predictedClass, conf)
		dg := fmt.Sprintf("Densidad mamaria BI-RADS clase %s", predictedClass)
		return &d, &dg, &conf
	}

	return nil, nil, nil
}

// asFloat extrae un float64 de un valor any proveniente de json.Unmarshal
// (los números JSON se decodifican como float64 en map[string]any).
func asFloat(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}
