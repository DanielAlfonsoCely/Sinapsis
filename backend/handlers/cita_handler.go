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

// CitasHoy maneja GET /api/v1/citas/hoy.
// Retorna las citas del médico autenticado para el día de hoy con datos del paciente.
func (h *CitaHandler) CitasHoy(c *gin.Context) {
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

	var medicoID uuid.UUID
	err = h.pool.QueryRow(context.Background(),
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede ver sus citas"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	rows, err := h.pool.Query(context.Background(),
		`SELECT ci.id, ci.fecha_hora, ci.estado,
		        p.id, p.nombre_paciente, p.apellidos_paciente, p.numero_documento
		 FROM cita ci
		 JOIN paciente p ON p.id = ci.paciente_id
		 WHERE ci.medico_id = $1
		   AND ci.fecha_hora::date = CURRENT_DATE
		 ORDER BY ci.fecha_hora`,
		medicoID,
	)
	if err != nil {
		log.Printf("list citas hoy error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch citas"})
		return
	}
	defer rows.Close()

	type PacienteResumen struct {
		ID                string `json:"id"`
		NombrePaciente    string `json:"nombre_paciente"`
		ApellidosPaciente string `json:"apellidos_paciente"`
		NumeroDocumento   string `json:"numero_documento"`
	}
	type CitaItem struct {
		ID        string          `json:"id"`
		FechaHora time.Time       `json:"fecha_hora"`
		Estado    string          `json:"estado"`
		Paciente  PacienteResumen `json:"paciente"`
	}

	citas := make([]CitaItem, 0)
	for rows.Next() {
		var ci CitaItem
		if err := rows.Scan(
			&ci.ID, &ci.FechaHora, &ci.Estado,
			&ci.Paciente.ID, &ci.Paciente.NombrePaciente, &ci.Paciente.ApellidosPaciente, &ci.Paciente.NumeroDocumento,
		); err != nil {
			log.Printf("scan cita error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read citas"})
			return
		}
		citas = append(citas, ci)
	}

	total := len(citas)
	completadas := 0
	pendientes := 0
	for _, ci := range citas {
		if ci.Estado == "completada" {
			completadas++
		} else if ci.Estado == "programada" || ci.Estado == "en_curso" {
			pendientes++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"citas":       citas,
		"total":       total,
		"completadas": completadas,
		"pendientes":  pendientes,
	})
}

// Create maneja POST /api/v1/citas.
// La agenda el PACIENTE autenticado. Puede agendar con:
//   - su médico tratante (general), siempre; o
//   - un especialista, solo si su médico general autorizó esa especialidad (remisión).
//
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

	// Usamos offset fijo -05:00 (Colombia) para evitar LoadLocation que falla en Docker sin tzdata.
	bogota := time.FixedZone("America/Bogota", -5*60*60)
	fechaHora, err := time.ParseInLocation("2006-01-02T15:04", req.FechaHora, bogota)
	if err != nil {
		fechaHora, err = time.ParseInLocation("2006-01-02T15:04:05", req.FechaHora, bogota)
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
		"id":           citaID,
		"paciente_id":  pacienteID,
		"fecha_hora":   fechaHora,
		"es_tratante":  esTratante,
		"especialidad": strings.TrimSpace(especialidad),
	})
}

// CitasSemana maneja GET /api/v1/citas/semana?fecha=YYYY-MM-DD
// Retorna todas las citas del médico autenticado para la semana que contiene la fecha dada.
// Si no se pasa ?fecha, usa la semana actual.
func (h *CitaHandler) CitasSemana(c *gin.Context) {
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

	var medicoID uuid.UUID
	err = h.pool.QueryRow(context.Background(),
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede ver su agenda"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// Calcular inicio de semana (lunes) a partir de ?fecha o hoy.
	// Usamos offset fijo -05:00 (Colombia) para no depender de tzdata en Docker.
	fechaParam := c.Query("fecha")
	bogota := time.FixedZone("America/Bogota", -5*60*60)
	var ref time.Time
	if fechaParam != "" {
		ref, err = time.ParseInLocation("2006-01-02", fechaParam, bogota)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fecha debe tener formato YYYY-MM-DD"})
			return
		}
	} else {
		ref = time.Now().In(bogota)
	}
	// Retroceder al lunes en hora Bogotá
	weekday := int(ref.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	lunes := ref.AddDate(0, 0, -(weekday - 1))
	lunes = time.Date(lunes.Year(), lunes.Month(), lunes.Day(), 0, 0, 0, 0, bogota)
	domingo := lunes.AddDate(0, 0, 7)

	rows, err := h.pool.Query(context.Background(),
		`SELECT ci.id, ci.fecha_hora, ci.estado, ci.motivo,
		        p.id, p.nombre_paciente, p.apellidos_paciente, p.numero_documento
		 FROM cita ci
		 JOIN paciente p ON p.id = ci.paciente_id
		 WHERE ci.medico_id = $1
		   AND ci.fecha_hora >= $2
		   AND ci.fecha_hora < $3
		 ORDER BY ci.fecha_hora`,
		medicoID, lunes, domingo,
	)
	if err != nil {
		log.Printf("list citas semana error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch citas"})
		return
	}
	defer rows.Close()

	type PacienteResumen struct {
		ID                string `json:"id"`
		NombrePaciente    string `json:"nombre_paciente"`
		ApellidosPaciente string `json:"apellidos_paciente"`
		NumeroDocumento   string `json:"numero_documento"`
	}
	type CitaItem struct {
		ID        string          `json:"id"`
		FechaHora time.Time       `json:"fecha_hora"`
		Estado    string          `json:"estado"`
		Motivo    *string         `json:"motivo"`
		Paciente  PacienteResumen `json:"paciente"`
	}

	citas := make([]CitaItem, 0)
	for rows.Next() {
		var ci CitaItem
		if err := rows.Scan(
			&ci.ID, &ci.FechaHora, &ci.Estado, &ci.Motivo,
			&ci.Paciente.ID, &ci.Paciente.NombrePaciente, &ci.Paciente.ApellidosPaciente, &ci.Paciente.NumeroDocumento,
		); err != nil {
			log.Printf("scan cita semana error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read citas"})
			return
		}
		citas = append(citas, ci)
	}

	c.JSON(http.StatusOK, gin.H{
		"citas":        citas,
		"semana_desde": lunes.Format("2006-01-02"),
		"semana_hasta": domingo.AddDate(0, 0, -1).Format("2006-01-02"),
	})
}
