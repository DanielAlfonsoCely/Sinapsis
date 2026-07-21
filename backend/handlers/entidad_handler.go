package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"sinapsis-backend/db/repositories"
	"sinapsis-backend/models"
	"sinapsis-backend/services"
)

type EntidadHandler struct {
	service *services.EntidadService
}

func NewEntidadHandler(service *services.EntidadService) *EntidadHandler {
	return &EntidadHandler{service: service}
}

// actorIDFromContext extrae el user_id crudo del contexto (para auditoría),
// igual que en FormulaHandler.
func (h *EntidadHandler) actorIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		return uuid.Nil, false
	}
	return userID, true
}

// Create maneja POST /api/v1/entidades.
// Registra una nueva entidad de salud. Solo accesible por admin.
func (h *EntidadHandler) Create(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	var req models.CreateEntidadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entidad, err := h.service.Create(c.Request.Context(), actorID, req)
	if err != nil {
		if errors.Is(err, repositories.ErrEntidadDuplicada) {
			c.JSON(http.StatusConflict, gin.H{"error": "ya existe una entidad con ese NIT"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create entidad"})
		return
	}

	c.JSON(http.StatusCreated, entidad)
}

// List maneja GET /api/v1/entidades.
func (h *EntidadHandler) List(c *gin.Context) {
	entidades, err := h.service.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidades"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entidades": entidades, "total": len(entidades)})
}

// ListAdmin maneja GET /api/v1/admin/entidades.
// Solo accesible por admin_plataforma (protegido por RequireAdmin middleware).
func (h *EntidadHandler) ListAdmin(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	q := c.DefaultQuery("q", "")

	entidades, err := h.service.ListAdmin(c.Request.Context(), actorID, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidades"})
		return
	}

	c.JSON(http.StatusOK, models.AdminEntidadListResponse{
		Entidades: entidades,
		Total:     len(entidades),
	})
}

// GetByIDAdmin maneja GET /api/v1/admin/entidades/:id.
// Solo accesible por admin_plataforma (protegido por RequireAdmin middleware).
func (h *EntidadHandler) GetByIDAdmin(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	detalle, err := h.service.GetByIDAdmin(c.Request.Context(), actorID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "entidad no encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch entidad"})
		return
	}

	c.JSON(http.StatusOK, detalle)
}

// Stats maneja GET /api/v1/admin/stats.
// Devuelve métricas globales de la plataforma para el dashboard admin.
func (h *EntidadHandler) Stats(c *gin.Context) {
	actorID, ok := h.actorIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}

	stats, err := h.service.Stats(c.Request.Context(), actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_consultas":         stats.TotalConsultas,
		"total_pacientes_activos": stats.TotalPacientesActivos,
		"total_usuarios_activos":  stats.TotalUsuariosActivos,
	})
}
