package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

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
