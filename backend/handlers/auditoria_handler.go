package handlers

import (
	"log"
	"net/http"
	"sinapsis-backend/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sinapsis-backend/services"
)

type AuditoriaHandler struct {
	service *services.AuditService
}

func NewAuditoriaHandler(service *services.AuditService) *AuditoriaHandler {
	return &AuditoriaHandler{service: service}
}

func (h *AuditoriaHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit inválido"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset inválido"})
		return
	}

	entries, total, err := h.service.ListRecent(c.Request.Context(), limit, offset)
	if err != nil {
		log.Printf("list audit error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch auditoría"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"registros": entries,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})

}

func (h *AuditoriaHandler) LookCritical(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit inválido"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset inválido"})
		return
	}

	entries, total, err := h.service.LookCritical(c.Request.Context(), limit, offset)
	if err != nil {
		log.Printf("list audit error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch auditoría"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"registros": entries,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

func (h *AuditoriaHandler) RegistrarExportacion(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	usuarioID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	var req struct {
		TablaAfectada string  `json:"tabla_afectada" binding:"required"`
		Detalles      *string `json:"detalles"`
		RegistroID    *string `json:"registro_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var registroID *uuid.UUID
	if req.RegistroID != nil && *req.RegistroID != "" {
		parsed, err := uuid.Parse(*req.RegistroID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "registro_id inválido"})
			return
		}
		registroID = &parsed
	}

	ip := c.ClientIP()

	if err := h.service.Record(
		c.Request.Context(),
		usuarioID,
		models.AuditExport,
		req.TablaAfectada,
		registroID,
		&ip,
		req.Detalles,
		models.High,
	); err != nil {
		log.Printf("record export audit error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record auditoría"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "registrado"})
}
