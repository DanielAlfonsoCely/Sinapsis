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

	// PreDiagnostico es la impresión clínica inicial del médico, registrada
	// ANTES de ver cualquier sugerencia de IA (RF-12/RN-007). Es opcional al
	// crear la consulta -- puede completarse después vía
	// PATCH /consultas/:id/pre-diagnostico, pero es obligatorio antes de
	// solicitar o visualizar hallazgos de análisis IA.
	PreDiagnostico *string `json:"pre_diagnostico"`

	ProximaCita *string `json:"proxima_cita"` // YYYY-MM-DD, opcional

	// Fórmula médica emitida durante la consulta (HU-06). Opcional: si viene con
	// medicamentos, se registra en la misma transacción, ligada a esta consulta.
	Medicamentos        []Medicamento `json:"medicamentos" binding:"omitempty,dive"`
	FormulaIndicaciones *string       `json:"formula_indicaciones"`
}

// UpdatePreDiagnosticoRequest es el payload para registrar/actualizar el
// pre-diagnóstico de una consulta ya creada (RF-12/RN-007).
type UpdatePreDiagnosticoRequest struct {
	PreDiagnostico string `json:"pre_diagnostico" binding:"required"`
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
	PreDiagnostico          *string `json:"pre_diagnostico"`

	ProximaCita    *time.Time `json:"proxima_cita"`
	FechaConsulta  time.Time  `json:"fecha_consulta"`
	EstadoConsulta string     `json:"estado_consulta"`

	MedicoNombre       string `json:"medico_nombre"`
	MedicoEspecialidad string `json:"medico_especialidad"`

	Anexos        []AnexoItem        `json:"anexos"`
	SugerenciasIA []SugerenciaIAItem `json:"sugerencias_ia"`
}

// AnexoItem es un resultado/imagen adjunto a una consulta (HU-07).
type AnexoItem struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Tipo   string `json:"tipo"`
}

// SugerenciaIAItem es el resumen de una sugerencia de análisis IA asociada a
// un examen de esta consulta, para mostrarla en la historia clínica -- así el
// resultado de "usar la sugerencia" (marcarla como revisada) queda visible en
// el expediente, no solo en la pantalla de análisis IA.
type SugerenciaIAItem struct {
	ID                  string  `json:"id"`
	ExaminagenID        string  `json:"examinagen_id"`
	ModeloIAUtilizado   string  `json:"modelo_ia_utilizado"`
	EstadoProcesamiento string  `json:"estado_procesamiento"`
	DiagnosticoSugerido *string `json:"diagnostico_sugerido"`
	DescripcionHallazgo *string `json:"descripcion_hallazgo"`
	EstadoRevision      string  `json:"estado_revision"`
}
