package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DifyService encapsula las llamadas a la API REST de Dify.
// Usa el endpoint /chat-messages en modo blocking: envía la pregunta
// y espera la respuesta completa antes de devolver.
type DifyService struct {
	baseURL    string // ej: http://host.docker.internal/v1
	apiKey     string // API key del chatbot en Dify
	httpClient *http.Client
}

func NewDifyService(baseURL, apiKey string) *DifyService {
	return &DifyService{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 180 * time.Second, // Ollama local puede tardar
		},
	}
}

// --- Estructuras de la API de Dify ---

type difyRequest struct {
	Inputs         map[string]any `json:"inputs"`
	Query          string         `json:"query"`
	ResponseMode   string         `json:"response_mode"` // "blocking"
	ConversationID string         `json:"conversation_id,omitempty"`
	User           string         `json:"user"`
}

type difyResponse struct {
	Answer         string `json:"answer"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

// ChatResult es lo que devuelve DifyService al handler.
type ChatResult struct {
	Answer         string `json:"answer"`
	ConversationID string `json:"conversation_id"`
}

// Chat envía una pregunta a Dify y devuelve la respuesta en modo blocking.
//
// userID: identificador del usuario (ej: UUID del paciente). Dify lo usa
// para asociar la conversación, pero no autentica — la auth es solo el API key.
//
// conversationID: si es "" Dify inicia una nueva conversación; si se pasa
// un ID previo, continúa esa sesión (historial incluido).
func (s *DifyService) Chat(ctx context.Context, userID, question, conversationID string) (*ChatResult, error) {
	payload := difyRequest{
		Inputs:         map[string]any{},
		Query:          question,
		ResponseMode:   "blocking",
		ConversationID: conversationID,
		User:           userID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("dify: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/chat-messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("dify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dify: http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("dify: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dify: status %d: %s", resp.StatusCode, string(respBody))
	}

	var difyResp difyResponse
	if err := json.Unmarshal(respBody, &difyResp); err != nil {
		return nil, fmt.Errorf("dify: unmarshal response: %w", err)
	}

	return &ChatResult{
		Answer:         difyResp.Answer,
		ConversationID: difyResp.ConversationID,
	}, nil
}
