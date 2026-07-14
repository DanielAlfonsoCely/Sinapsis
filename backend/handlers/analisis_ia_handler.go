package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
	"sinapsis-backend/queue"
)

// AnalisisIAHandler gestiona los tres endpoints del módulo de análisis IA:
//
//   POST   /api/v1/examenes/:id/analisis-ia   — solicitar análisis
//   GET    /api/v1/sugerencias-ia/:id          — consultar estado/resultado
//   PATCH  /api/v1/sugerencias-ia/:id/revision — revisión médica
type AnalisisIAHandler struct {
	pool      *pgxpool.Pool
	publisher *queue.Publisher // nil si RABBITMQ_URL no está configurado
}

func NewAnalisisIAHandler(pool *pgxpool.Pool, publisher *queue.Publisher) *AnalisisIAHandler {
	return &AnalisisIAHandler{pool: pool, publisher: publisher}
}

// validAnalysisTypes son los cuatro tipos de análisis soportados por el microservicio.
var validAnalysisTypes = map[models.AnalysisType]bool{
	models.AnalysisTypeSpleenSegmentation: true,
	models.AnalysisTypeLungNodule:         true,
	models.AnalysisTypeBrainTumor:         true,
	models.AnalysisTypeBreastDensity:      true,
}

// SolicitarAnalisis maneja POST /api/v1/examenes/:id/analisis-ia (HU-IA-01).
// El médico que solicitó el examen (examinagen.medico_solicitante_id) pide que el
// microservicio MONAI ejecute un análisis sobre la imagen ya almacenada.
func (h *AnalisisIAHandler) SolicitarAnalisis(c *gin.Context) {
	if h.publisher == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "módulo de análisis IA no disponible"})
		return
	}

	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	examinagenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "examinagen id inválido"})
		return
	}

	var req models.SolicitarAnalisisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validAnalysisTypes[req.AnalysisType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "analysis_type inválido",
			"allowed": []string{
				string(models.AnalysisTypeSpleenSegmentation),
				string(models.AnalysisTypeLungNodule),
				string(models.AnalysisTypeBrainTumor),
				string(models.AnalysisTypeBreastDensity),
			},
		})
		return
	}

	ctx := context.Background()

	// Resolver médico autenticado.
	var medicoID uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede solicitar análisis IA"})
			return
		}
		log.Printf("analisis_ia lookup medico: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// Leer el examen: debe existir, pertenecer a este médico y tener imagen.
	var pacienteID, historiaClinicaID uuid.UUID
	var urlImagen *string
	err = h.pool.QueryRow(ctx, `
		SELECT paciente_id, historia_clinica_id, url_imagen
		FROM examinagen
		WHERE id = $1 AND medico_solicitante_id = $2
	`, examinagenID, medicoID).Scan(&pacienteID, &historiaClinicaID, &urlImagen)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "examen no encontrado o no autorizado"})
			return
		}
		log.Printf("analisis_ia lookup examinagen: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find examen"})
		return
	}
	if urlImagen == nil || *urlImagen == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "el examen no tiene imagen asociada"})
		return
	}

	// Construir el mensaje de solicitud.
	requestID := uuid.New()
	correlationID := uuid.New()
	now := time.Now().UTC()

	amqpReq := models.AnalysisRequest{
		RequestID:     requestID.String(),
		StudyID:       examinagenID.String(),
		PatientRef:    pacienteID.String(),
		AnalysisType:  req.AnalysisType,
		ImageURI:      *urlImagen,
		RequestedBy:   medicoID.String(),
		CorrelationID: correlationID.String(),
		IssuedAt:      now,
	}

	// Insertar sugerencia_ia en estado 'enviado' dentro de una transacción,
	// y publicar el mensaje AMQP solo si el INSERT fue exitoso.
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		log.Printf("analisis_ia begin tx: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	var sugerenciaID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO sugerencia_ia
			(examinagen_id, historia_clinica_id, request_id, correlation_id,
			 estado_procesamiento, modelo_ia_utilizado)
		VALUES ($1, $2, $3, $4, 'enviado', '')
		RETURNING id
	`, examinagenID, historiaClinicaID, requestID, correlationID).Scan(&sugerenciaID)
	if err != nil {
		log.Printf("analisis_ia insert sugerencia_ia: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create sugerencia"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("analisis_ia commit: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit"})
		return
	}

	// Publicar a RabbitMQ después del commit (si falla, la fila queda en 'enviado'
	// pero no hay resultado; el médico puede reintentar).
	if err := h.publisher.Publish(ctx, amqpReq); err != nil {
		log.Printf("analisis_ia publish: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue analysis"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"id":         sugerenciaID,
		"request_id": requestID,
		"estado":     "enviado",
	})
}

// GetSugerencia maneja GET /api/v1/sugerencias-ia/:id.
// Devuelve el estado de procesamiento y, si ya completó, los resultados del modelo.
func (h *AnalisisIAHandler) GetSugerencia(c *gin.Context) {
	sugerenciaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	ctx := context.Background()

	// Verificar que el médico autenticado tenga acceso (es el solicitante del examen).
	userIDRaw, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDRaw.(string))

	var medicoID uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "acceso no autorizado"})
			return
		}
		log.Printf("analisis_ia get medico: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	row := h.pool.QueryRow(ctx, `
		SELECT
			s.id, s.examinagen_id, s.historia_clinica_id,
			s.request_id, s.correlation_id,
			s.estado_procesamiento, s.modelo_ia_utilizado,
			s.confianza_prediccion, s.descripcion_hallazgo, s.diagnostico_sugerido,
			s.metricas, s.fecha_analisis,
			s.estado_revision, s.observaciones_medico, s.fecha_revision, s.medico_revisor_id
		FROM sugerencia_ia s
		JOIN examinagen e ON e.id = s.examinagen_id
		WHERE s.id = $1 AND e.medico_solicitante_id = $2
	`, sugerenciaID, medicoID)

	var resp models.SugerenciaIAResponse
	var requestID, correlationID, medicoRevisorID *uuid.UUID
	var metricasRaw []byte

	err = row.Scan(
		&resp.ID, &resp.ExaminagenID, &resp.HistoriaClinicaID,
		&requestID, &correlationID,
		&resp.EstadoProcesamiento, &resp.ModeloIAUtilizado,
		&resp.ConfianzaPrediccion, &resp.DescripcionHallazgo, &resp.DiagnosticoSugerido,
		&metricasRaw, &resp.FechaAnalisis,
		&resp.EstadoRevision, &resp.ObservacionesMedico, &resp.FechaRevision, &medicoRevisorID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sugerencia no encontrada"})
			return
		}
		log.Printf("analisis_ia get sugerencia: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sugerencia"})
		return
	}

	if requestID != nil {
		s := requestID.String()
		resp.RequestID = &s
	}
	if correlationID != nil {
		s := correlationID.String()
		resp.CorrelationID = &s
	}
	if medicoRevisorID != nil {
		s := medicoRevisorID.String()
		resp.MedicoRevisorID = &s
	}
	if len(metricasRaw) > 0 {
		if err := json.Unmarshal(metricasRaw, &resp.Metricas); err != nil {
			log.Printf("analisis_ia unmarshal metricas: %v", err)
		}
	}

	c.JSON(http.StatusOK, resp)
}

// Revision maneja PATCH /api/v1/sugerencias-ia/:id/revision.
// El médico marca la sugerencia como revisada o rechazada y opcionalmente agrega
// sus observaciones. Solo se puede revisar si el análisis ya completó.
func (h *AnalisisIAHandler) Revision(c *gin.Context) {
	sugerenciaID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req models.RevisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	userIDRaw, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDRaw.(string))

	var medicoID uuid.UUID
	if err := h.pool.QueryRow(ctx,
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede revisar sugerencias IA"})
			return
		}
		log.Printf("analisis_ia revision medico: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// Verificar que la sugerencia exista, pertenezca al médico y esté completada.
	var estadoProcesamiento string
	err = h.pool.QueryRow(ctx, `
		SELECT s.estado_procesamiento
		FROM sugerencia_ia s
		JOIN examinagen e ON e.id = s.examinagen_id
		WHERE s.id = $1 AND e.medico_solicitante_id = $2
	`, sugerenciaID, medicoID).Scan(&estadoProcesamiento)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "sugerencia no encontrada"})
			return
		}
		log.Printf("analisis_ia revision lookup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch sugerencia"})
		return
	}
	if estadoProcesamiento != "completado" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "solo se puede revisar un análisis completado",
			"estado": estadoProcesamiento,
		})
		return
	}

	_, err = h.pool.Exec(ctx, `
		UPDATE sugerencia_ia
		SET
			estado_revision      = $1,
			observaciones_medico = $2,
			fecha_revision       = NOW(),
			medico_revisor_id    = $3
		WHERE id = $4
	`, req.EstadoRevision, req.ObservacionesMedico, medicoID, sugerenciaID)
	if err != nil {
		log.Printf("analisis_ia revision update: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update revision"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             sugerenciaID,
		"estado_revision": req.EstadoRevision,
	})
}
