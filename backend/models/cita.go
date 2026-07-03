package models

// CreateCitaRequest es el payload para agendar una cita a un paciente (agenda mínima).
// Una cita 'programada' para hoy es la que habilita el botón de Consulta.
type CreateCitaRequest struct {
	PacienteID string  `json:"paciente_id" binding:"required,uuid"`
	FechaHora  string  `json:"fecha_hora" binding:"required"` // ISO 8601, ej: 2026-07-15T10:30
	Motivo     *string `json:"motivo"`
}
