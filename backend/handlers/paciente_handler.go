package handlers

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"net/http"
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
// q es opcional: filtra por nombre, apellidos o número de documento.
// Solo devuelve los pacientes cuyo médico tratante es el médico autenticado
// (Opción A: cada médico ve únicamente sus pacientes).
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

	var medicoID uuid.UUID
	err = h.pool.QueryRow(context.Background(),
		`SELECT id FROM medico WHERE usuario_id = $1`, userID,
	).Scan(&medicoID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede listar pacientes"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	rows, err := h.pool.Query(
		context.Background(),
		`SELECT p.id, p.numero_documento, p.tipo_documento, p.nombre_paciente, p.apellidos_paciente,
		        p.telefono, p.email,
		        (SELECT MAX(co.fecha_consulta) FROM consulta co WHERE co.paciente_id = p.id) AS ultima_consulta,
		        (SELECT MIN(co.proxima_cita) FROM consulta co
		           WHERE co.paciente_id = p.id AND co.proxima_cita >= CURRENT_DATE) AS proxima_cita,
		        EXISTS (SELECT 1 FROM cita ci
		           WHERE ci.paciente_id = p.id AND ci.estado = 'programada'
		             AND ci.fecha_hora::date = CURRENT_DATE) AS tiene_cita_hoy,
		        p.estado
		 FROM paciente p
		 JOIN historia_clinica hc ON hc.paciente_id = p.id
		 WHERE hc.medico_tratante_id = $1
		   AND ($2 = ''
		    OR p.nombre_paciente ILIKE '%' || $2 || '%'
		    OR p.apellidos_paciente ILIKE '%' || $2 || '%'
		    OR p.numero_documento ILIKE '%' || $2 || '%')
		 ORDER BY p.nombre_paciente, p.apellidos_paciente`,
		medicoID, q,
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
			&p.Telefono, &p.Email, &p.UltimaConsulta, &p.ProximaCita, &p.TieneCitaHoy, &p.Estado,
		); err != nil {
			log.Printf("scan paciente error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read pacientes"})
			return
		}
		// proxima_cita = la próxima cita futura más cercana entre sus consultas
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

// GetByID maneja GET /api/v1/pacientes/:id
func (h *PacienteHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid paciente id"})
		return
	}

	var p models.Paciente
	err = h.pool.QueryRow(
		context.Background(),
		`SELECT id, usuario_id, numero_documento, tipo_documento, nombre_paciente, apellidos_paciente,
		        fecha_nacimiento, sexo, tipo_sangre, alergias, direccion, telefono, email,
		        contacto_emergencia, telefono_emergencia, antecedentes_medicos, medicamentos_actuales,
		        estado_civil, ocupacion, aseguradora, numero_afiliacion, fecha_registro, estado
		 FROM paciente WHERE id = $1`,
		id,
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
		log.Printf("get paciente error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch paciente"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// ListMedicos maneja GET /api/v1/medicos.
// Devuelve los médicos disponibles como destino de una remisión/transferencia.
func (h *PacienteHandler) ListMedicos(c *gin.Context) {
	rows, err := h.pool.Query(context.Background(),
		`SELECT m.id, u.nombre_usuario, u.apellidos, m.especialidad
		 FROM medico m
		 JOIN usuario u ON u.id = m.usuario_id
		 WHERE m.estado = true
		 ORDER BY u.nombre_usuario, u.apellidos`,
	)
	if err != nil {
		log.Printf("list medicos error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch médicos"})
		return
	}
	defer rows.Close()

	medicos := make([]models.MedicoListItem, 0)
	for rows.Next() {
		var m models.MedicoListItem
		var nombre, apellidos string
		if err := rows.Scan(&m.ID, &nombre, &apellidos, &m.Especialidad); err != nil {
			log.Printf("scan medico error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read médicos"})
			return
		}
		m.Nombre = nombre + " " + apellidos
		medicos = append(medicos, m)
	}

	c.JSON(http.StatusOK, gin.H{"medicos": medicos})
}

// Transfer maneja POST /api/v1/pacientes/:id/transferir.
// Reasigna el médico tratante del paciente. Solo el médico tratante actual puede
// remitir a su paciente; tras la transferencia deja de verlo y el destino lo ve.
func (h *PacienteHandler) Transfer(c *gin.Context) {
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

	var req models.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	medicoDestinoID, err := uuid.Parse(req.MedicoDestinoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "medico_destino_id inválido"})
		return
	}

	ctx := context.Background()

	// Médico que solicita la transferencia.
	var medicoOrigenID uuid.UUID
	err = h.pool.QueryRow(ctx, `SELECT id FROM medico WHERE usuario_id = $1`, userID).Scan(&medicoOrigenID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusForbidden, gin.H{"error": "solo un médico puede remitir pacientes"})
			return
		}
		log.Printf("lookup medico error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico"})
		return
	}

	if medicoOrigenID == medicoDestinoID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el paciente ya está asignado a ese médico"})
		return
	}

	// El destino debe existir y estar activo.
	var exists2 bool
	if err := h.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM medico WHERE id = $1 AND estado = true)`, medicoDestinoID,
	).Scan(&exists2); err != nil {
		log.Printf("check destino error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify médico destino"})
		return
	}
	if !exists2 {
		c.JSON(http.StatusNotFound, gin.H{"error": "el médico destino no existe"})
		return
	}

	// Solo el médico tratante actual puede transferir a su paciente.
	tag, err := h.pool.Exec(ctx,
		`UPDATE historia_clinica
		    SET medico_tratante_id = $1, fecha_actualizacion = NOW()
		  WHERE paciente_id = $2 AND medico_tratante_id = $3`,
		medicoDestinoID, pacienteID, medicoOrigenID,
	)
	if err != nil {
		log.Printf("transfer error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to transfer paciente"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "el paciente no está bajo su cuidado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "transferido", "medico_destino_id": medicoDestinoID})
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
