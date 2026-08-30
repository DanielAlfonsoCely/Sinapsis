package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sinapsis-backend/services"
)

// DifyHandler expone el endpoint de chat para pacientes.
// El frontend llama a POST /api/v1/chat con la pregunta y
// (opcionalmente) el conversation_id para continuar una sesión.
type DifyHandler struct {
	dify *services.DifyService
}

func NewDifyHandler(dify *services.DifyService) *DifyHandler {
	return &DifyHandler{dify: dify}
}

type chatRequest struct {
	Question       string `json:"question" binding:"required"`
	ConversationID string `json:"conversation_id"` // opcional; "" = nueva sesión
}

type chatResponse struct {
	Answer         string `json:"answer"`
	ConversationID string `json:"conversation_id"`
}

// Chat recibe una pregunta del paciente y devuelve la respuesta de Dify.
//
// El userID viene del token JWT (lo pone middleware.RequireAuth en el contexto).
// Así Dify asocia la conversación al paciente correcto sin exponer el UUID
// directamente en el body.
func (h *DifyHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El campo 'question' es requerido"})
		return
	}

	// El middleware RequireAuth deja el user_id en el contexto como string.
	userID, _ := c.Get("user_id")
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		userIDStr = "paciente-anonimo"
	}

	result, err := h.dify.Chat(c.Request.Context(), userIDStr, req.Question, req.ConversationID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "No se pudo conectar con el asistente de IA"})
		return
	}

	c.JSON(http.StatusOK, chatResponse{
		Answer:         result.Answer,
		ConversationID: result.ConversationID,
	})
}
