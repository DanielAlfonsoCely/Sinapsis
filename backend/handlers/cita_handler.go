package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
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
// La agenda el PACIENTE autenticado. Puede agendar con:
//   - su médico tratante (general), siempre; o
//   - un especialista, solo si su médico general autorizó esa especialidad (remisión).
// Agendar con un especialista NO cambia el médico tratante del paciente.
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
	medicoID, err := uuid.Parse(req.MedicoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "medico_id inválido"})
		return
	}

	fechaHora, err := time.Parse("2006-01-02T15:04", req.FechaHora)
	if err != nil {
		fechaHora, err = time.Parse("2006-01-02T15:04:05", req.FechaHora)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fecha_hora debe tener formato YYYY-MM-DDTHH:MM"})
			return
		}
	}

	ctx := context.Background()

	// Solo un paciente agenda sus citas.
	var pacienteID uuid.UUID
	var tratanteID *uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT p.id, hc.medico_tratante_id
		 FROM paciente p JOIN historia_clinica hc ON hc.paciente_id = p.id
		 WHERE p.usuario_id = $1`, userID,
	).Scan(&pacienteID, &tratanteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un paciente puede agendar sus citas"})
			return
		}
		log.Printf("lookup paciente error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify paciente"})
		return
	}

	// El médico destino: puede ser el tratante o un especialista.
	var especialidad string
	if err := h.pool.QueryRow(ctx,
		`SELECT especialidad FROM medico WHERE id = $1 AND estado = true`, medicoID,
	).Scan(&especialidad); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "el médico no existe"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	esTratante := tratanteID != nil && *tratanteID == medicoID
	if !esTratante {
		// Debe existir una remisión autorizada para esa especialidad.
		var autorizado bool
		if err := h.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM remision
			   WHERE paciente_id = $1 AND estado = 'autorizada' AND especialidad = $2)`,
			pacienteID, especialidad,
		).Scan(&autorizado); err != nil {
			log.Printf("check remision error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify autorización"})
			return
		}
		if !autorizado {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "no tienes autorización para agendar con esta especialidad",
			})
			return
		}
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
		"es_tratante": esTratante,
		"especialidad": strings.TrimSpace(especialidad),
	})
}
