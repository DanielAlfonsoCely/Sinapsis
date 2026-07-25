package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/db"
	"sinapsis-backend/models"
)

var (
	ErrDuplicateEmail     = errors.New("usuario duplicate email")
	ErrDuplicateDocumento = errors.New("numero_documento already registered")
	ErrNoUpdateFields     = errors.New("no update fields provided")
	ErrUsuarioNotFound    = errors.New("usuario not found")
)

type UsuarioFilters struct {
	Search string
	Rol    string
	Estado *bool
	Limit  int
	Offset int
}

type UsuarioRepository struct {
	pool *pgxpool.Pool
}

func NewUsuarioRepository(pool *pgxpool.Pool) *UsuarioRepository {
	return &UsuarioRepository{pool: pool}
}

// ErrEntidadRequired se devuelve cuando el rol requiere entidad_id y no llegó.
var ErrEntidadRequired = errors.New("entidad_id is required for this role")

// Create inserta la fila base en `usuario` y, según tipo_usuario, la fila
// extendida correspondiente (medico, paciente, administrador_entidad o
// administrador_plataforma), todo en una sola transacción.
func (r *UsuarioRepository) Create(ctx context.Context, req models.CreateUsuarioAdminRequest, hashedPassword string) (models.Usuario, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Usuario{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var user models.Usuario
	err = tx.QueryRow(ctx,
		`INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario, estado, fecha_creacion, fecha_actualizacion)
		 VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
		 RETURNING id, nombre_usuario, apellidos, email, tipo_usuario, estado, fecha_creacion, fecha_actualizacion`,
		req.NombreUsuario, req.Apellidos, req.Email, hashedPassword, req.TipoUsuario,
	).Scan(
		&user.ID,
		&user.NombreUsuario,
		&user.Apellidos,
		&user.Email,
		&user.TipoUsuario,
		&user.Estado,
		&user.FechaCreacion,
		&user.FechaActualizacion,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return models.Usuario{}, ErrDuplicateEmail
		}
		return models.Usuario{}, fmt.Errorf("create usuario: %w", err)
	}

	switch req.TipoUsuario {
	case "medico":
		if req.EntidadID == "" {
			return models.Usuario{}, ErrEntidadRequired
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO medico (usuario_id, numero_documento, especialidad, numero_colegiado, experiencia_anios, entidad_id, estado, fecha_registro)
			 VALUES ($1, $2, $3, $4, $5, $6, true, NOW())`,
			user.ID, req.NumeroDocumento, req.Especialidad, req.NumeroColegiado, req.ExperienciaAnios, req.EntidadID,
		)
		if err != nil {
			if isUniqueViolation(err) {
				return models.Usuario{}, fmt.Errorf("create medico: %w", ErrDuplicateDocumento)
			}
			return models.Usuario{}, fmt.Errorf("create medico: %w", err)
		}

	case "paciente":
		fechaNacimiento, ferr := time.Parse("2006-01-02", req.FechaNacimiento)
		if ferr != nil {
			return models.Usuario{}, fmt.Errorf("fecha_nacimiento inválida: %w", ferr)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO paciente (usuario_id, numero_documento, tipo_documento, nombre_paciente, apellidos_paciente,
			                       fecha_nacimiento, sexo, telefono, email, fecha_registro, estado)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), true)`,
			user.ID, req.NumeroDocumento, req.TipoDocumento, req.NombreUsuario, req.Apellidos,
			fechaNacimiento, req.Sexo, req.Telefono, req.Email,
		)
		if err != nil {
			if isUniqueViolation(err) {
				return models.Usuario{}, fmt.Errorf("create paciente: %w", ErrDuplicateDocumento)
			}
			return models.Usuario{}, fmt.Errorf("create paciente: %w", err)
		}

	case "admin_entidad":
		if req.EntidadID == "" {
			return models.Usuario{}, ErrEntidadRequired
		}
		var rolID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM rol WHERE nombre_rol = 'Administrador de Entidad'`).Scan(&rolID); err != nil {
			return models.Usuario{}, fmt.Errorf("lookup rol admin_entidad: %w", err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO administrador_entidad (usuario_id, entidad_id, rol_id, activo, fecha_asignacion)
			 VALUES ($1, $2, $3, true, NOW())`,
			user.ID, req.EntidadID, rolID,
		)
		if err != nil {
			return models.Usuario{}, fmt.Errorf("create administrador_entidad: %w", err)
		}

	case "admin_plataforma":
		var rolID, plataformaID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM rol WHERE nombre_rol = 'Administrador de Plataforma'`).Scan(&rolID); err != nil {
			return models.Usuario{}, fmt.Errorf("lookup rol admin_plataforma: %w", err)
		}
		if err := tx.QueryRow(ctx, `SELECT id FROM plataforma LIMIT 1`).Scan(&plataformaID); err != nil {
			return models.Usuario{}, fmt.Errorf("lookup plataforma: %w", err)
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO administrador_plataforma (usuario_id, plataforma_id, rol_id, activo, fecha_asignacion)
			 VALUES ($1, $2, $3, true, NOW())`,
			user.ID, plataformaID, rolID,
		)
		if err != nil {
			return models.Usuario{}, fmt.Errorf("create administrador_plataforma: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Usuario{}, fmt.Errorf("commit create usuario: %w", err)
	}

	return user, nil
}

func (r *UsuarioRepository) Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) error {
	var builder db.QueryBuilder
	builder.AddIfNotNil("nombre_usuario = $%d", req.NombreUsuario)
	builder.AddIfNotNil("apellidos = $%d", req.Apellidos)
	builder.AddIfNotNil("email = $%d", req.Email)
	builder.AddIfNotNil("tipo_usuario = $%d", req.TipoUsuario)
	builder.AddIfNotNil("estado = $%d", req.Estado)

	if len(builder.Args()) == 0 {
		return ErrNoUpdateFields
	}

	setClause := builder.Build(", ")
	query := `UPDATE usuario SET ` + setClause + ", fecha_actualizacion = NOW() WHERE id = $" + strconv.Itoa(len(builder.Args())+1) + " AND estado = true"
	args := append(builder.Args(), id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("update usuario: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUsuarioNotFound
	}

	return nil
}

func (r *UsuarioRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE usuario SET estado = false, fecha_actualizacion = NOW()
		 WHERE id = $1 AND estado = true`,
		id,
	)
	if err != nil {
		return fmt.Errorf("deactivate usuario: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUsuarioNotFound
	}

	return nil
}

func (r *UsuarioRepository) AssignRole(ctx context.Context, id uuid.UUID, role string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE usuario SET tipo_usuario = $1, fecha_actualizacion = NOW()
		 WHERE id = $2 AND estado = true`,
		role, id,
	)
	if err != nil {
		return fmt.Errorf("assign role usuario: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrUsuarioNotFound
	}

	return nil
}

func (r *UsuarioRepository) List(ctx context.Context, filters UsuarioFilters) ([]models.Usuario, int, error) {
	query := `SELECT id, nombre_usuario, apellidos, email, tipo_usuario, estado, fecha_creacion, fecha_actualizacion FROM usuario WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM usuario WHERE 1=1`
	args := make([]interface{}, 0, 8)
	countArgs := make([]interface{}, 0, 8)
	argIndex := 1

	if filters.Search != "" {
		clause := fmt.Sprintf(` AND (nombre_usuario ILIKE $%d OR apellidos ILIKE $%d OR email ILIKE $%d)`, argIndex, argIndex+1, argIndex+2)
		query += clause
		countQuery += clause
		searchPattern := "%" + filters.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
		argIndex += 3
	}

	if filters.Rol != "" {
		query += fmt.Sprintf(` AND tipo_usuario = $%d`, argIndex)
		countQuery += fmt.Sprintf(` AND tipo_usuario = $%d`, argIndex)
		args = append(args, filters.Rol)
		countArgs = append(countArgs, filters.Rol)
		argIndex++
	}

	if filters.Estado != nil {
		query += fmt.Sprintf(` AND estado = $%d`, argIndex)
		countQuery += fmt.Sprintf(` AND estado = $%d`, argIndex)
		args = append(args, *filters.Estado)
		countArgs = append(countArgs, *filters.Estado)
		argIndex++
	}

	query += ` ORDER BY fecha_creacion DESC`
	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)
	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list usuarios: %w", err)
	}
	defer rows.Close()

	usuarios := make([]models.Usuario, 0)
	for rows.Next() {
		var u models.Usuario
		if err := rows.Scan(&u.ID, &u.NombreUsuario, &u.Apellidos, &u.Email, &u.TipoUsuario, &u.Estado, &u.FechaCreacion, &u.FechaActualizacion); err != nil {
			return nil, 0, fmt.Errorf("scan usuario: %w", err)
		}
		usuarios = append(usuarios, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate usuarios: %w", err)
	}

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count usuarios: %w", err)
	}

	return usuarios, total, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}