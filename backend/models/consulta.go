package models

import (
	"time"

	"github.com/google/uuid"
)

// CreateConsultaRequest es el payload para registrar una nueva consulta médica
// dentro de la historia clínica de un paciente (HU-03).
//
// Sigue el estándar de historia clínica en Colombia (Res. 1995 de 1999):
// motivo de consulta, anamnesis (enfermedad actual), revisión por sistemas,
// signos vitales, examen físico, diagnóstico con código CIE-10 y plan de manejo.
type CreateConsultaRequest struct {
	PacienteID string `json:"paciente_id" binding:"required,uuid"`

	TipoConsulta   *string `json:"tipo_consulta"`
	MotivoConsulta string  `json:"motivo_consulta" binding:"required"`

	Anamnesis         *string `json:"anamnesis"`
	RevisionSistemas  *string `json:"revision_sistemas"`
	ExamenFisico      *string `json:"examen_fisico"`
	HallazgosClinicos *string `json:"hallazgos_clinicos"`

	// Signos vitales
	PresionArterial        *string  `json:"presion_arterial"`
	FrecuenciaCardiaca     *int     `json:"frecuencia_cardiaca"`
	FrecuenciaRespiratoria *int     `json:"frecuencia_respiratoria"`
	Temperatura            *float64 `json:"temperatura"`
	SaturacionOxigeno      *int     `json:"saturacion_oxigeno"`
	PesoKg                 *float64 `json:"peso_kg"`
	TallaCm                *float64 `json:"talla_cm"`

	DiagnosticoPrincipal    *string `json:"diagnostico_principal"`
	DiagnosticoCIE10        *string `json:"diagnostico_cie10"`
	PlanManejo              *string `json:"plan_manejo"`
	ProcedimientosIndicados *string `json:"procedimientos_indicados"`
	ObservacionesMedico     *string `json:"observaciones_medico"`

	ProximaCita *string `json:"proxima_cita"` // YYYY-MM-DD, opcional
}

// ConsultaListItem es cada entrada de la historia clínica de un paciente (HU-04),
// incluyendo el médico que atendió la consulta.
type ConsultaListItem struct {
	ID uuid.UUID `json:"id"`

	TipoConsulta   *string `json:"tipo_consulta"`
	MotivoConsulta string  `json:"motivo_consulta"`

	Anamnesis         *string `json:"anamnesis"`
	RevisionSistemas  *string `json:"revision_sistemas"`
	ExamenFisico      *string `json:"examen_fisico"`
	HallazgosClinicos *string `json:"hallazgos_clinicos"`

	PresionArterial        *string  `json:"presion_arterial"`
	FrecuenciaCardiaca     *int     `json:"frecuencia_cardiaca"`
	FrecuenciaRespiratoria *int     `json:"frecuencia_respiratoria"`
	Temperatura            *float64 `json:"temperatura"`
	SaturacionOxigeno      *int     `json:"saturacion_oxigeno"`
	PesoKg                 *float64 `json:"peso_kg"`
	TallaCm                *float64 `json:"talla_cm"`

	DiagnosticoPrincipal    *string `json:"diagnostico_principal"`
	DiagnosticoCIE10        *string `json:"diagnostico_cie10"`
	PlanManejo              *string `json:"plan_manejo"`
	ProcedimientosIndicados *string `json:"procedimientos_indicados"`
	ObservacionesMedico     *string `json:"observaciones_medico"`

	ProximaCita    *time.Time `json:"proxima_cita"`
	FechaConsulta  time.Time  `json:"fecha_consulta"`
	EstadoConsulta string     `json:"estado_consulta"`

	MedicoNombre       string `json:"medico_nombre"`
	MedicoEspecialidad string `json:"medico_especialidad"`
}
