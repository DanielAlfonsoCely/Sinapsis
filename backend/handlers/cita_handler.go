package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

type CitaHandler struct {
	pool *pgxpool.Pool
}

func NewCitaHandler(pool *pgxpool.Pool) *CitaHandler {
	return &CitaHandler{pool: pool}
}

// Create maneja POST /api/v1/citas.
// El médico autenticado agenda una cita a uno de sus pacientes. Una cita
// 'programada' para hoy es la que habilita el botón de Consulta en la lista.
func (h *CitaHandler) Create(c *gin.Context) {
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

	var req models.CreateCitaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pacienteID, err := uuid.Parse(req.PacienteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "paciente_id inválido"})
		return
	}

	// Acepta "2006-01-02T15:04" (datetime-local) o el formato con segundos.
	fechaHora, err := time.Parse("2006-01-02T15:04", req.FechaHora)
	if err != nil {
		fechaHora, err = time.Parse("2006-01-02T15:04:05", req.FechaHora)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fecha_hora debe tener formato YYYY-MM-DDTHH:MM"})
			return
		}
	}

	ctx := context.Background()

	var medicoID uuid.UUID
	err = h.pool.QueryRow(ctx, `SELECT id FROM medico WHERE usuario_id = $1`, userID).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede agendar citas"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// La cita solo puede agendarse para un paciente del propio médico tratante.
	var esTratante bool
	if err := h.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM historia_clinica
		   WHERE paciente_id = $1 AND medico_tratante_id = $2)`,
		pacienteID, medicoID,
	).Scan(&esTratante); err != nil {
		log.Printf("check tratante error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify paciente"})
		return
	}
	if !esTratante {
		c.JSON(http.StatusForbidden, gin.H{"error": "el paciente no está bajo su cuidado"})
		return
	}

	var citaID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`INSERT INTO cita (paciente_id, medico_id, fecha_hora, motivo, estado)
		 VALUES ($1, $2, $3, $4, 'programada')
		 RETURNING id`,
		pacienteID, medicoID, fechaHora, req.Motivo,
	).Scan(&citaID)
	if err != nil {
		log.Printf("create cita error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cita"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          citaID,
		"paciente_id": pacienteID,
		"fecha_hora":  fechaHora,
	})
}
