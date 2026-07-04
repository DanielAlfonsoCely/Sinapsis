package models

import (
	"time"

	"github.com/google/uuid"
)

// Medicamento es cada ítem recetado dentro de una fórmula médica.
type Medicamento struct {
	Nombre     string `json:"nombre" binding:"required"`
	Dosis      string `json:"dosis"`
	Frecuencia string `json:"frecuencia"`
	Duracion   string `json:"duracion"`
	Cantidad   string `json:"cantidad"`
}

// CreateFormulaRequest es el payload para registrar una fórmula médica (HU-06).
// Va ligada a la consulta donde se recetó (RN: se prescribe dentro de un encuentro).
type CreateFormulaRequest struct {
	PacienteID       string        `json:"paciente_id" binding:"required,uuid"`
	ConsultaID       string        `json:"consulta_id" binding:"required,uuid"`
	Medicamentos     []Medicamento `json:"medicamentos" binding:"required,min=1,dive"`
	Indicaciones     *string       `json:"indicaciones"`
	Contraindicaciones *string     `json:"contraindicaciones"`
	FechaVencimiento *string       `json:"fecha_vencimiento"` // YYYY-MM-DD, opcional
}

// FormulaListItem es cada fórmula del paciente, con el médico que la prescribió
// y la consulta a la que pertenece (para enlazar con la historia clínica).
type FormulaListItem struct {
	ID                 uuid.UUID     `json:"id"`
	ConsultaID         *uuid.UUID    `json:"consulta_id"`
	Medicamentos       []Medicamento `json:"medicamentos"`
	Indicaciones       *string       `json:"indicaciones"`
	Contraindicaciones *string       `json:"contraindicaciones"`
	FechaPrescripcion  time.Time     `json:"fecha_prescripcion"`
	FechaVencimiento   *time.Time    `json:"fecha_vencimiento"`
	EstadoFormula      string        `json:"estado_formula"`
	MedicoNombre       string        `json:"medico_nombre"`
}
