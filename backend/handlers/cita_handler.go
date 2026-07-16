package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
	"sinapsis-backend/services"
)

type CitaHandler struct {
	service *services.CitaService
}

func NewCitaHandler(service *services.CitaService) *CitaHandler {
	return &CitaHandler{service: service}
}

// CitasHoy maneja GET /api/v1/citas/hoy.
// Retorna las citas del médico autenticado para el día de hoy con datos del paciente.
//
// Nota de alcance: esta lectura NO se audita -- es la agenda propia del médico,
// no la consulta del expediente de un paciente específico.
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

	ctx := c.Request.Context()

	medicoID, err := h.service.ResolveMedicoID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrSoloMedico) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede ver sus citas"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	citas, err := h.service.CitasHoy(ctx, medicoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch citas"})
		return
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

	// La cita debe caer en una franja válida: de 06:00 a 19:30, en intervalos de media hora.
	if !horarioValido(fechaHora) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "la hora debe estar entre las 06:00 y las 19:30, en intervalos de media hora",
		})
		return
	}

	ctx := c.Request.Context()

	res, err := h.service.Create(ctx, userID, userID, medicoID, fechaHora, req.Motivo)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrSoloPaciente):
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un paciente puede agendar sus citas"})
		case errors.Is(err, repositories.ErrMedicoNoExiste):
			c.JSON(http.StatusNotFound, gin.H{"error": "el médico no existe"})
		case errors.Is(err, repositories.ErrSinAutorizacion):
			c.JSON(http.StatusForbidden, gin.H{"error": "no tienes autorización para agendar con esta especialidad"})
		case errors.Is(err, repositories.ErrCitaActivaExistente):
			c.JSON(http.StatusConflict, gin.H{"error": "ya tienes una cita activa con este médico"})
		case errors.Is(err, repositories.ErrHorarioOcupado):
			c.JSON(http.StatusConflict, gin.H{"error": "ese horario ya fue tomado, elige otro"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cita"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           res.CitaID,
		"paciente_id":  res.PacienteID,
		"fecha_hora":   fechaHora,
		"es_tratante":  res.EsTratante,
		"especialidad": strings.TrimSpace(res.Especialidad),
	})
}

// Disponibilidad maneja GET /api/v1/citas/disponibilidad?medico_id=...&fecha=YYYY-MM-DD
// Devuelve las franjas horarias del día (06:00 a 19:30, cada media hora) para
// ese médico, marcando cuáles ya están ocupadas por otro paciente.
func (h *CitaHandler) Disponibilidad(c *gin.Context) {
	if _, exists := c.Get("user_id"); !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	medicoID, err := uuid.Parse(c.Query("medico_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "medico_id inválido"})
		return
	}

	// Usamos offset fijo -05:00 (Colombia) para evitar LoadLocation, igual que en Create.
	bogota := time.FixedZone("America/Bogota", -5*60*60)
	fecha, err := time.ParseInLocation("2006-01-02", c.Query("fecha"), bogota)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha debe tener formato YYYY-MM-DD"})
		return
	}
	diaSiguiente := fecha.AddDate(0, 0, 1)

	ctx := c.Request.Context()

	existeMedico, err := h.service.ExisteMedicoActivo(ctx, medicoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}
	if !existeMedico {
		c.JSON(http.StatusNotFound, gin.H{"error": "el médico no existe"})
		return
	}

	ocupados, err := h.service.HorariosOcupados(ctx, medicoID, fecha, diaSiguiente)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch disponibilidad"})
		return
	}

	type Horario struct {
		Hora       string `json:"hora"`
		Disponible bool   `json:"disponible"`
	}
	horarios := make([]Horario, 0, len(slotsDelDia()))
	for _, slot := range slotsDelDia() {
		horarios = append(horarios, Horario{Hora: slot, Disponible: !ocupados[slot]})
	}

	c.JSON(http.StatusOK, gin.H{
		"medico_id": medicoID,
		"fecha":     fecha.Format("2006-01-02"),
		"horarios":  horarios,
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

	ctx := c.Request.Context()

	medicoID, err := h.service.ResolveMedicoID(ctx, userID)
	if err != nil {
		if errors.Is(err, repositories.ErrSoloMedico) {
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

	citas, err := h.service.CitasSemana(ctx, medicoID, lunes, domingo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch citas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"citas":        citas,
		"semana_desde": lunes.Format("2006-01-02"),
		"semana_hasta": domingo.AddDate(0, 0, -1).Format("2006-01-02"),
	})
}
