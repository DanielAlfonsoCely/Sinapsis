package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/db"
	"sinapsis-backend/models"
)

var (
	ErrDuplicateEmail  = errors.New("usuario duplicate email")
	ErrNoUpdateFields  = errors.New("no update fields provided")
	ErrUsuarioNotFound = errors.New("usuario not found")
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

func (r *UsuarioRepository) Create(ctx context.Context, req models.RegisterRequest, hashedPassword string) (models.Usuario, error) {
	var user models.Usuario
	err := r.pool.QueryRow(
		ctx,
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
