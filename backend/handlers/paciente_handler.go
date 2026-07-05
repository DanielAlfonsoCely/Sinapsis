package handlers

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"sinapsis-backend/models"
)

type PacienteHandler struct {
	pool *pgxpool.Pool
}

func NewPacienteHandler(pool *pgxpool.Pool) *PacienteHandler {
	return &PacienteHandler{pool: pool}
}

// List maneja GET /api/v1/pacientes?q=texto
// Si el médico es de triage (especialidad contiene "triage"), ve TODOS los
// pacientes de su entidad. Si no, ve solo sus propios pacientes y los que
// tienen cita con él (especialista vía remisión).
func (h *PacienteHandler) List(c *gin.Context) {
	q := c.Query("q")

	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	var medicoID, entidadID uuid.UUID
	var especialidad string
	err = h.pool.QueryRow(context.Background(),
		`SELECT id, entidad_id, especialidad FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID, &entidadID, &especialidad)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede listar pacientes"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	// Detectar si es médico de triage (lógica de negocio en Go, no en DB).
	esTriage := strings.Contains(strings.ToLower(especialidad), "triage") ||
		strings.Contains(strings.ToLower(especialidad), "triagge")

	rows, err := h.pool.Query(
		context.Background(),
		`SELECT p.id, p.numero_documento, p.tipo_documento, p.nombre_paciente, p.apellidos_paciente,
		        p.telefono, p.email,
		        (SELECT MAX(co.fecha_consulta) FROM consulta co WHERE co.paciente_id = p.id) AS ultima_consulta,
		        (SELECT MIN(co.proxima_cita) FROM consulta co
		           WHERE co.paciente_id = p.id AND co.proxima_cita >= CURRENT_DATE) AS proxima_cita,
		        EXISTS (SELECT 1 FROM cita ci
		           WHERE ci.paciente_id = p.id AND ci.medico_id = $1 AND ci.estado = 'programada'
		             AND ci.fecha_hora::date = CURRENT_DATE) AS tiene_cita_hoy,
		        (hc.medico_tratante_id = $1) AS es_tratante,
		        p.estado
		 FROM paciente p
		 JOIN historia_clinica hc ON hc.paciente_id = p.id
		 WHERE (
		   ($3 = true AND hc.entidad_id = $4)
		   OR
		   ($3 = false AND (
		     hc.medico_tratante_id = $1
		     OR EXISTS (SELECT 1 FROM cita ci2 WHERE ci2.paciente_id = p.id AND ci2.medico_id = $1)
		   ))
		 )
		   AND ($2 = ''
		    OR p.nombre_paciente ILIKE '%' || $2 || '%'
		    OR p.apellidos_paciente ILIKE '%' || $2 || '%'
		    OR p.numero_documento ILIKE '%' || $2 || '%')
		 ORDER BY es_tratante DESC, p.nombre_paciente, p.apellidos_paciente`,
		medicoID, q, esTriage, entidadID,
	)
	if err != nil {
		log.Printf("list pacientes error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pacientes"})
		return
	}
	defer rows.Close()

	pacientes := make([]models.PacienteListItem, 0)
	for rows.Next() {
		var p models.PacienteListItem
		if err := rows.Scan(
			&p.ID, &p.NumeroDocumento, &p.TipoDocumento, &p.NombrePaciente, &p.ApellidosPaciente,
			&p.Telefono, &p.Email, &p.UltimaConsulta, &p.ProximaCita, &p.TieneCitaHoy, &p.EsTratante, &p.Estado,
		); err != nil {
			log.Printf("scan paciente error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read pacientes"})
			return
		}
		pacientes = append(pacientes, p)
	}

	if err := rows.Err(); err != nil {
		log.Printf("rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch pacientes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pacientes": pacientes,
		"total":     len(pacientes),
		"es_triage": esTriage,
	})
}

// Me maneja GET /api/v1/pacientes/me.
// El propio paciente autenticado consulta sus datos, resolviendo el usuario_id
// del JWT (a diferencia de GetByID, que recibe el id del paciente por URL).
func (h *PacienteHandler) Me(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	var p models.Paciente
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT id, usuario_id, numero_documento, tipo_documento, nombre_paciente, apellidos_paciente,
		        fecha_nacimiento, sexo, tipo_sangre, alergias, direccion, telefono, email,
		        contacto_emergencia, telefono_emergencia, antecedentes_medicos, medicamentos_actuales,
		        estado_civil, ocupacion, aseguradora, numero_afiliacion, fecha_registro, estado
		 FROM paciente WHERE usuario_id = $1`,
		userID,
	).Scan(
		&p.ID, &p.UsuarioID, &p.NumeroDocumento, &p.TipoDocumento, &p.NombrePaciente, &p.ApellidosPaciente,
		&p.FechaNacimiento, &p.Sexo, &p.TipoSangre, &p.Alergias, &p.Direccion, &p.Telefono, &p.Email,
		&p.ContactoEmergencia, &p.TelefonoEmergencia, &p.AntecedentesMedicos, &p.MedicamentosActuales,
		&p.EstadoCivil, &p.Ocupacion, &p.Aseguradora, &p.NumeroAfiliacion, &p.FechaRegistro, &p.Estado,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "paciente not found"})
			return
		}
		log.Printf("get me error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch paciente"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// GetByID maneja GET /api/v1/pacientes/:id (autenticado).
// tiene_cita_hoy es específico del médico autenticado: indica si ESE médico tiene
// una cita programada para hoy con el paciente (gate de consulta para tratante y
// especialista por igual).
func (h *PacienteHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	// Médico autenticado (si lo es); si no, tiene_cita_hoy quedará en false.
	var medicoID uuid.UUID
	if userIDRaw, ok := c.Get("user_id"); ok {
		if userID, err := uuid.Parse(userIDRaw.(string)); err == nil {
			_ = h.pool.QueryRow(context.Background(),
				`SELECT id FROM medico WHERE usuario_id = $1`, userID,
			).Scan(&medicoID)
		}
	}

	var p models.Paciente
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT id, usuario_id, numero_documento, tipo_documento, nombre_paciente, apellidos_paciente,
		        fecha_nacimiento, sexo, tipo_sangre, alergias, direccion, telefono, email,
		        contacto_emergencia, telefono_emergencia, antecedentes_medicos, medicamentos_actuales,
		        estado_civil, ocupacion, aseguradora, numero_afiliacion, fecha_registro, estado,
		        EXISTS (SELECT 1 FROM cita ci
		           WHERE ci.paciente_id = paciente.id AND ci.medico_id = $2
		             AND ci.estado = 'programada'
		             AND ci.fecha_hora::date = CURRENT_DATE) AS tiene_cita_hoy
		 FROM paciente WHERE id = $1`,
		id, medicoID,
	).Scan(
		&p.ID, &p.UsuarioID, &p.NumeroDocumento, &p.TipoDocumento, &p.NombrePaciente, &p.ApellidosPaciente,
		&p.FechaNacimiento, &p.Sexo, &p.TipoSangre, &p.Alergias, &p.Direccion, &p.Telefono, &p.Email,
		&p.ContactoEmergencia, &p.TelefonoEmergencia, &p.AntecedentesMedicos, &p.MedicamentosActuales,
		&p.EstadoCivil, &p.Ocupacion, &p.Aseguradora, &p.NumeroAfiliacion, &p.FechaRegistro, &p.Estado,
		&p.TieneCitaHoy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "paciente not found"})
			return
		}
		log.Printf("get paciente error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch paciente"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// ListEspecialidades maneja GET /api/v1/especialidades.
// Devuelve las especialidades disponibles (excluye Medicina General) para que el
// médico general elija qué autorizar a su paciente.
func (h *PacienteHandler) ListEspecialidades(c *gin.Context) {
	rows, err := h.pool.Query(context.Background(),
		`SELECT DISTINCT especialidad FROM medico
		 WHERE estado = true AND especialidad NOT ILIKE 'Medicina General'
		 ORDER BY especialidad`,
	)
	if err != nil {
		log.Printf("list especialidades error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch especialidades"})
		return
	}
	defer rows.Close()

	especialidades := make([]string, 0)
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read especialidades"})
			return
		}
		especialidades = append(especialidades, e)
	}

	c.JSON(http.StatusOK, gin.H{"especialidades": especialidades})
}

// AutorizarEspecialidad maneja POST /api/v1/pacientes/:id/remisiones.
// El médico tratante (general) autoriza a su paciente a consultar una especialidad.
// NO cambia el médico tratante: el especialista atenderá al paciente temporalmente.
func (h *PacienteHandler) AutorizarEspecialidad(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	pacienteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	var req models.AutorizarEspecialidadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()

	// Solo el médico tratante del paciente puede autorizar.
	var medicoID uuid.UUID
	err = h.pool.QueryRow(ctx, `SELECT id FROM medico WHERE usuario_id = $1`, userID).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede autorizar"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	var esTratante bool
	if err := h.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM historia_clinica
		   WHERE paciente_id = $1 AND medico_tratante_id = $2)`,
		pacienteID, medicoID,
	).Scan(&esTratante); err != nil {
		log.Printf("check tratante error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify paciente"})
		return
	}
	if !esTratante {
		c.JSON(http.StatusForbidden, gin.H{"error": "el paciente no está bajo su cuidado"})
		return
	}

	// Debe existir al menos un especialista de esa especialidad.
	var hayEspecialista bool
	if err := h.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM medico WHERE estado = true AND especialidad = $1)`,
		req.Especialidad,
	).Scan(&hayEspecialista); err != nil {
		log.Printf("check especialidad error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify especialidad"})
		return
	}
	if !hayEspecialista {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no hay especialistas de esa especialidad"})
		return
	}

	var remisionID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`INSERT INTO remision (paciente_id, medico_remitente_id, especialidad, motivo, estado)
		 VALUES ($1, $2, $3, $4, 'autorizada')
		 RETURNING id`,
		pacienteID, medicoID, req.Especialidad, req.Motivo,
	).Scan(&remisionID)
	if err != nil {
		log.Printf("create remision error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create remisión"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           remisionID,
		"especialidad": req.Especialidad,
	})
}

// MiAgenda maneja GET /api/v1/mi/agenda (para el paciente autenticado).
// Devuelve su médico tratante, las especialidades autorizadas con sus especialistas,
// y sus próximas citas.
func (h *PacienteHandler) MiAgenda(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	ctx := context.Background()

	var pacienteID uuid.UUID
	var tratanteID *uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT p.id, hc.medico_tratante_id
		 FROM paciente p JOIN historia_clinica hc ON hc.paciente_id = p.id
		 WHERE p.usuario_id = $1`, userID,
	).Scan(&pacienteID, &tratanteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un paciente puede ver su agenda"})
			return
		}
		log.Printf("lookup paciente error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch agenda"})
		return
	}

	// Médico tratante (general).
	var tratante *models.MedicoListItem
	if tratanteID != nil {
		var m models.MedicoListItem
		var nombre, apellidos string
		if err := h.pool.QueryRow(ctx,
			`SELECT m.id, u.nombre_usuario, u.apellidos, m.especialidad
			 FROM medico m JOIN usuario u ON u.id = m.usuario_id WHERE m.id = $1`,
			*tratanteID,
		).Scan(&m.ID, &nombre, &apellidos, &m.Especialidad); err == nil {
			m.Nombre = nombre + " " + apellidos
			tratante = &m
		}
	}

	// Especialidades autorizadas + especialistas de cada una.
	remRows, err := h.pool.Query(ctx,
		`SELECT DISTINCT especialidad FROM remision
		 WHERE paciente_id = $1 AND estado = 'autorizada' ORDER BY especialidad`,
		pacienteID,
	)
	if err != nil {
		log.Printf("list remisiones error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch remisiones"})
		return
	}
	type autorizacion struct {
		Especialidad  string                  `json:"especialidad"`
		Especialistas []models.MedicoListItem `json:"especialistas"`
	}
	autorizaciones := make([]autorizacion, 0)
	var especialidades []string
	for remRows.Next() {
		var e string
		if err := remRows.Scan(&e); err != nil {
			remRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read remisiones"})
			return
		}
		especialidades = append(especialidades, e)
	}
	remRows.Close()

	for _, e := range especialidades {
		espRows, err := h.pool.Query(ctx,
			`SELECT m.id, u.nombre_usuario, u.apellidos, m.especialidad
			 FROM medico m JOIN usuario u ON u.id = m.usuario_id
			 WHERE m.estado = true AND m.especialidad = $1
			 ORDER BY u.nombre_usuario`, e,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch especialistas"})
			return
		}
		especialistas := make([]models.MedicoListItem, 0)
		for espRows.Next() {
			var m models.MedicoListItem
			var nombre, apellidos string
			if err := espRows.Scan(&m.ID, &nombre, &apellidos, &m.Especialidad); err != nil {
				espRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read especialistas"})
				return
			}
			m.Nombre = nombre + " " + apellidos
			especialistas = append(especialistas, m)
		}
		espRows.Close()
		autorizaciones = append(autorizaciones, autorizacion{Especialidad: e, Especialistas: especialistas})
	}

	// Próximas citas del paciente.
	citaRows, err := h.pool.Query(ctx,
		`SELECT ci.id, u.nombre_usuario, u.apellidos, m.especialidad, ci.fecha_hora, ci.motivo, ci.estado
		 FROM cita ci
		 JOIN medico m ON m.id = ci.medico_id
		 JOIN usuario u ON u.id = m.usuario_id
		 WHERE ci.paciente_id = $1
		 ORDER BY ci.fecha_hora DESC`, pacienteID,
	)
	if err != nil {
		log.Printf("list citas error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch citas"})
		return
	}
	defer citaRows.Close()
	citas := make([]gin.H, 0)
	for citaRows.Next() {
		var id uuid.UUID
		var nombre, apellidos, especialidad, estado string
		var fechaHora time.Time
		var motivo *string
		if err := citaRows.Scan(&id, &nombre, &apellidos, &especialidad, &fechaHora, &motivo, &estado); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read citas"})
			return
		}
		citas = append(citas, gin.H{
			"id":            id,
			"medico_nombre": nombre + " " + apellidos,
			"especialidad":  especialidad,
			"fecha_hora":    fechaHora,
			"motivo":        motivo,
			"estado":        estado,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"medico_tratante": tratante,
		"autorizaciones":  autorizaciones,
		"citas":           citas,
	})
}

const tempPasswordChars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

// generateTempPassword genera una contraseña temporal aleatoria y criptográficamente segura.
func generateTempPassword(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(tempPasswordChars))))
		if err != nil {
			return "", err
		}
		result[i] = tempPasswordChars[n.Int64()]
	}
	return string(result), nil
}

// Create maneja POST /api/v1/pacientes (HU-02).
// Registra en una sola transacción: la cuenta de usuario del paciente (con
// contraseña temporal generada y "enviada" por correo), el paciente y su
// historia clínica, vinculada a la entidad del médico autenticado.
func (h *PacienteHandler) Create(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing session"})
		return
	}
	userID, err := uuid.Parse(userIDRaw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
		return
	}

	var req models.CreatePacienteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fechaNacimiento, err := time.Parse("2006-01-02", req.FechaNacimiento)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha_nacimiento debe tener formato YYYY-MM-DD"})
		return
	}

	ctx := context.Background()

	// El médico autenticado determina la entidad dueña de la historia clínica (RN-003)
	// y queda como su médico tratante (Opción A: cada paciente pertenece a un médico).
	var medicoID, medicoEntidadID uuid.UUID
	err = h.pool.QueryRow(ctx,
		`SELECT id, entidad_id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID, &medicoEntidadID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede registrar pacientes"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	tempPassword, err := generateTempPassword(12)
	if err != nil {
		log.Printf("generate password error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create credentials"})
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		log.Printf("begin tx error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	var usuarioID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario, fecha_creacion, fecha_actualizacion)
		 VALUES ($1, $2, $3, $4, 'paciente', NOW(), NOW())
		 RETURNING id`,
		req.NombrePaciente, req.ApellidosPaciente, req.Email, string(hashedPassword),
	).Scan(&usuarioID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "ya existe un usuario con ese email"})
			return
		}
		log.Printf("create usuario error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create usuario"})
		return
	}

	var pacienteID uuid.UUID
	var fechaRegistro time.Time
	err = tx.QueryRow(ctx,
		`INSERT INTO paciente (usuario_id, numero_documento, tipo_documento, nombre_paciente, apellidos_paciente,
		                       fecha_nacimiento, sexo, telefono, email, direccion, fecha_registro, estado)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), true)
		 RETURNING id, fecha_registro`,
		usuarioID, req.NumeroDocumento, req.TipoDocumento, req.NombrePaciente, req.ApellidosPaciente,
		fechaNacimiento, req.Sexo, req.Telefono, req.Email, req.Direccion,
	).Scan(&pacienteID, &fechaRegistro)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "ya existe un paciente con ese número de documento"})
			return
		}
		log.Printf("create paciente error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create paciente"})
		return
	}

	var historiaClinicaID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO historia_clinica (paciente_id, entidad_id, medico_tratante_id, fecha_creacion, fecha_actualizacion)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 RETURNING id`,
		pacienteID, medicoEntidadID, medicoID,
	).Scan(&historiaClinicaID)
	if err != nil {
		log.Printf("create historia_clinica error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create historia clínica"})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("commit error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save paciente"})
		return
	}

	// Envío de correo simulado: aún no hay proveedor SMTP configurado.
	log.Printf("[EMAIL SIMULADO] Para: %s | Asunto: Bienvenido a SINAPSIS | Contraseña temporal: %s",
		req.Email, tempPassword)

	c.JSON(http.StatusCreated, gin.H{
		"id":                 pacienteID,
		"numero_documento":   req.NumeroDocumento,
		"nombre_paciente":    req.NombrePaciente,
		"apellidos_paciente": req.ApellidosPaciente,
		// temp_password: solo se expone mientras el envío de correo esté simulado.
		// Quitar este campo en cuanto haya un proveedor SMTP real.
		"temp_password":       tempPassword,
		"email":               req.Email,
		"historia_clinica_id": historiaClinicaID,
		"fecha_registro":      fechaRegistro,
	})
}
