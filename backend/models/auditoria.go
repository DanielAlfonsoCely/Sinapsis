package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuditOperation string

const (
	AuditCreate           AuditOperation = "crear"
	AuditUpdate           AuditOperation = "actualizar"
	AuditDelete           AuditOperation = "eliminar"
	AuditConsult          AuditOperation = "consultar"
	AuditExport           AuditOperation = "exportar"
	AuditChangePermission AuditOperation = "cambiar_permisos"
	AuditUseAI            AuditOperation = "usar_ia"
)

type ImportanceLevel string

const (
	SeverityCritical ImportanceLevel = "CRITICAL"
	SeverityHigh     ImportanceLevel = "HIGH"
	SeverityMedium   ImportanceLevel = "MEDIUM"
	SeverityLow      ImportanceLevel = "LOW"
)

// TablasSensibles son las tablas cuya modificación eleva la gravedad.
var tablasSensibles = map[string]bool{
	"usuario":          true,
	"medico":           true,
	"paciente":         true,
	"historia_clinica": true,
}

// ClassifyGravedad determina la gravedad automáticamente a partir de la
// operación y la tabla afectada.
func ClassifyGravedad(operacion AuditOperation, tabla string) ImportanceLevel {
	sensible := tablasSensibles[strings.ToLower(tabla)]

	switch operacion {
	case AuditDelete:
		if sensible {
			return SeverityCritical
		}
		return SeverityHigh
	case AuditChangePermission:
		return SeverityCritical
	case AuditUpdate:
		if sensible {
			return SeverityHigh
		}
		return SeverityMedium
	case AuditCreate:
		if sensible {
			return SeverityMedium
		}
		return SeverityLow
	case AuditExport:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

type AuditLogEntry struct {
	ID                uuid.UUID       `json:"id"`
	UsuarioID         uuid.UUID       `json:"usuario_id"`
	UsuarioNombre     string          `json:"usuario_nombre"`
	UsuarioEmail      string          `json:"usuario_email"`
	TipoOperacion     AuditOperation  `json:"tipo_operacion"`
	TablaAfectada     string          `json:"tabla_afectada"`
	RegistroID        *uuid.UUID      `json:"registro_id"`
	ValoresAnteriores json.RawMessage `json:"valores_anteriores"`
	ValoresNuevos     json.RawMessage `json:"valores_nuevos"`
	IPOrigen          *string         `json:"ip_origen"`
	Detalles          *string         `json:"detalles"`
	FechaOperacion    time.Time       `json:"fecha_operacion"`
	Gravedad          ImportanceLevel `json:"gravedad"`
}