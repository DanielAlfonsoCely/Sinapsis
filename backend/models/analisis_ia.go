package models

import "time"

// AnalysisType representa los cuatro modelos MONAI soportados.
// Valores en snake_case para coincidir exactamente con el contrato del
// microservicio (PROJECT_ARCHITECTURE §9).
type AnalysisType string

const (
	AnalysisTypeSpleenSegmentation   AnalysisType = "ct_spleen_segmentation"
	AnalysisTypeLungNodule           AnalysisType = "ct_lung_nodule_detection"
	AnalysisTypeBrainTumor           AnalysisType = "mri_brain_tumor_segmentation"
	AnalysisTypeBreastDensity        AnalysisType = "xr_breast_density_classification"
)

// AnalysisRequest es el mensaje que el backend publica a la cola
// RABBITMQ_REQUEST_QUEUE para que el microservicio ejecute la inferencia.
// Contrato exacto: PROJECT_ARCHITECTURE §9.
type AnalysisRequest struct {
	RequestID     string       `json:"request_id"`
	StudyID       string       `json:"study_id"`
	PatientRef    string       `json:"patient_ref"`
	AnalysisType  AnalysisType `json:"analysis_type"`
	ImageURI      string       `json:"image_uri"`
	RequestedBy   string       `json:"requested_by"`
	CorrelationID string       `json:"correlation_id"`
	IssuedAt      time.Time    `json:"issued_at"`
}

// ModelRef identifica el bundle MONAI que ejecutó la inferencia.
type ModelRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Artifact es un artefacto de salida producido por la inferencia (p.ej. máscara).
type Artifact struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// AnalysisResult es el mensaje que el microservicio publica de vuelta al exchange
// RABBITMQ_RESULT_EXCHANGE. El consumer del backend lo recibe y persiste en sugerencia_ia.
// Contrato exacto: PROJECT_ARCHITECTURE §9.
type AnalysisResult struct {
	RequestID   string            `json:"request_id"`
	StudyID     string            `json:"study_id"`
	Status      string            `json:"status"` // "succeeded" | "failed"
	Model       ModelRef          `json:"model"`
	Artifacts   []Artifact        `json:"artifacts"`
	Metrics     map[string]any    `json:"metrics"`
	Error       map[string]string `json:"error,omitempty"` // {"code":..., "message":...} si failed
	ProcessedAt time.Time         `json:"processed_at"`
	DurationMs  int               `json:"duration_ms"`
}

// SolicitarAnalisisRequest es el body del endpoint
// POST /api/v1/examenes/:id/analisis-ia.
type SolicitarAnalisisRequest struct {
	AnalysisType AnalysisType `json:"analysis_type" binding:"required"`
}

// SugerenciaIAResponse es la respuesta de GET /api/v1/sugerencias-ia/:id.
type SugerenciaIAResponse struct {
	ID                   string         `json:"id"`
	ExaminagenID         string         `json:"examinagen_id"`
	HistoriaClinicaID    string         `json:"historia_clinica_id"`
	RequestID            *string        `json:"request_id,omitempty"`
	CorrelationID        *string        `json:"correlation_id,omitempty"`
	EstadoProcesamiento  string         `json:"estado_procesamiento"`
	ModeloIAUtilizado    string         `json:"modelo_ia_utilizado"`
	ConfianzaPrediccion  *float64       `json:"confianza_prediccion,omitempty"`
	DescripcionHallazgo  *string        `json:"descripcion_hallazgo,omitempty"`
	DiagnosticoSugerido  *string        `json:"diagnostico_sugerido,omitempty"`
	Metricas             map[string]any `json:"metricas,omitempty"`
	FechaAnalisis        *time.Time     `json:"fecha_analisis,omitempty"`
	EstadoRevision       string         `json:"estado_revision"`
	ObservacionesMedico  *string        `json:"observaciones_medico,omitempty"`
	FechaRevision        *time.Time     `json:"fecha_revision,omitempty"`
	MedicoRevisorID      *string        `json:"medico_revisor_id,omitempty"`
}

// RevisionRequest es el body del endpoint
// PATCH /api/v1/sugerencias-ia/:id/revision.
type RevisionRequest struct {
	EstadoRevision      string  `json:"estado_revision" binding:"required,oneof=revisada rechazada"`
	ObservacionesMedico *string `json:"observaciones_medico"`
}
