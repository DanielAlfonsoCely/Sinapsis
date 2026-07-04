package models

// CreateCitaRequest es el payload para que un paciente agende una cita.
// El paciente sale del JWT; solo indica con qué médico y cuándo.
type CreateCitaRequest struct {
	MedicoID  string  `json:"medico_id" binding:"required,uuid"`
	FechaHora string  `json:"fecha_hora" binding:"required"` // ISO local, ej: 2026-07-15T10:30
	Motivo    *string `json:"motivo"`
}

// AutorizarEspecialidadRequest es el payload para que un médico general autorice
// a su paciente a consultar una especialidad (remisión). No cambia el tratante.
type AutorizarEspecialidadRequest struct {
	Especialidad string  `json:"especialidad" binding:"required"`
	Motivo       *string `json:"motivo"`
}
