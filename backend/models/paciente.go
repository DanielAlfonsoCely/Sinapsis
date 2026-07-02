package models

import (
	"time"

	"github.com/google/uuid"
)

// PacienteListItem es lo que se muestra en la tabla de /pacientes.
// UltimaConsulta y ProximaCita pueden ser nil si el paciente aún no tiene consultas.
type PacienteListItem struct {
	ID               uuid.UUID  `json:"id"`
	NumeroDocumento  string     `json:"numero_documento"`
	TipoDocumento    string     `json:"tipo_documento"`
	NombrePaciente   string     `json:"nombre_paciente"`
	ApellidosPaciente string    `json:"apellidos_paciente"`
	Telefono         *string    `json:"telefono"`
	Email            *string    `json:"email"`
	UltimaConsulta   *time.Time `json:"ultima_consulta"`
	ProximaCita      *time.Time `json:"proxima_cita"`
	Estado           bool       `json:"estado"`
}

// Paciente es el detalle completo, tal como está en la tabla paciente.
type Paciente struct {
	ID                   uuid.UUID `json:"id"`
	UsuarioID            uuid.UUID `json:"usuario_id"`
	NumeroDocumento      string    `json:"numero_documento"`
	TipoDocumento        string    `json:"tipo_documento"`
	NombrePaciente       string    `json:"nombre_paciente"`
	ApellidosPaciente    string    `json:"apellidos_paciente"`
	FechaNacimiento      time.Time `json:"fecha_nacimiento"`
	Sexo                 *string   `json:"sexo"`
	TipoSangre           *string   `json:"tipo_sangre"`
	Alergias             *string   `json:"alergias"`
	Direccion            *string   `json:"direccion"`
	Telefono             *string   `json:"telefono"`
	Email                *string   `json:"email"`
	ContactoEmergencia   *string   `json:"contacto_emergencia"`
	TelefonoEmergencia   *string   `json:"telefono_emergencia"`
	AntecedentesMedicos  *string   `json:"antecedentes_medicos"`
	MedicamentosActuales *string   `json:"medicamentos_actuales"`
	EstadoCivil          *string   `json:"estado_civil"`
	Ocupacion            *string   `json:"ocupacion"`
	Aseguradora          *string   `json:"aseguradora"`
	NumeroAfiliacion     *string   `json:"numero_afiliacion"`
	FechaRegistro        time.Time `json:"fecha_registro"`
	Estado               bool      `json:"estado"`
}
