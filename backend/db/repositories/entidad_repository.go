package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

var ErrEntidadDuplicada = errors.New("ya existe una entidad con ese NIT")

type EntidadRepository struct {
	pool *pgxpool.Pool
}

func NewEntidadRepository(pool *pgxpool.Pool) *EntidadRepository {
	return &EntidadRepository{pool: pool}
}

func (r *EntidadRepository) Create(ctx context.Context, req models.CreateEntidadRequest) (models.Entidad, error) {
	var entidad models.Entidad
	err := r.pool.QueryRow(ctx,
		`INSERT INTO entidad (nombre_entidad, tipo_entidad, nit, ciudad, direccion, telefono, estado, fecha_creacion)
		 VALUES ($1, $2, $3, $4, $5, $6, true, NOW())
		 RETURNING id, nombre_entidad, tipo_entidad, nit, direccion, telefono, ciudad, estado, fecha_creacion`,
		req.NombreEntidad, req.TipoEntidad, req.NIT, req.Ciudad, req.Direccion, req.Telefono,
	).Scan(
		&entidad.ID, &entidad.NombreEntidad, &entidad.TipoEntidad, &entidad.NIT,
		&entidad.Direccion, &entidad.Telefono, &entidad.Ciudad, &entidad.Estado, &entidad.FechaCreacion,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return entidad, ErrEntidadDuplicada
		}
		return entidad, err
	}
	return entidad, nil
}

func (r *EntidadRepository) List(ctx context.Context) ([]models.Entidad, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, nombre_entidad, tipo_entidad, nit, direccion, telefono, ciudad, estado, fecha_creacion
		 FROM entidad
		 ORDER BY nombre_entidad`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entidades := make([]models.Entidad, 0)
	for rows.Next() {
		var e models.Entidad
		if err := rows.Scan(
			&e.ID, &e.NombreEntidad, &e.TipoEntidad, &e.NIT,
			&e.Direccion, &e.Telefono, &e.Ciudad, &e.Estado, &e.FechaCreacion,
		); err != nil {
			return nil, err
		}
		entidades = append(entidades, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entidades, nil
}

func (r *EntidadRepository) ListAdmin(ctx context.Context, q string) ([]models.AdminEntidadListItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, nombre_entidad, tipo_entidad, nit, ciudad, estado, fecha_creacion
		 FROM entidad
		 WHERE (
		     $1 = ''
		     OR nombre_entidad ILIKE '%' || $1 || '%'
		     OR nit            ILIKE '%' || $1 || '%'
		     OR ciudad         ILIKE '%' || $1 || '%'
		 )
		 ORDER BY nombre_entidad`,
		q,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entidades := make([]models.AdminEntidadListItem, 0)
	for rows.Next() {
		var e models.AdminEntidadListItem
		if err := rows.Scan(
			&e.ID, &e.NombreEntidad, &e.TipoEntidad, &e.NIT,
			&e.Ciudad, &e.Estado, &e.FechaCreacion,
		); err != nil {
			return nil, err
		}
		entidades = append(entidades, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entidades, nil
}

// GetByIDAdmin arma el detalle completo de una entidad en 4 queries,
// igual que hacía el handler original.
func (r *EntidadRepository) GetByIDAdmin(ctx context.Context, id uuid.UUID) (models.EntidadDetalle, error) {
	var detalle models.EntidadDetalle

	err := r.pool.QueryRow(ctx,
		`SELECT id, nombre_entidad, tipo_entidad, nit, ciudad, direccion, telefono, estado, fecha_creacion
		 FROM entidad WHERE id = $1`,
		id,
	).Scan(
		&detalle.ID, &detalle.NombreEntidad, &detalle.TipoEntidad, &detalle.NIT,
		&detalle.Ciudad, &detalle.Direccion, &detalle.Telefono, &detalle.Estado, &detalle.FechaCreacion,
	)
	if err != nil {
		return detalle, err // el handler distingue pgx.ErrNoRows
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM historia_clinica WHERE entidad_id = $1`,
		id,
	).Scan(&detalle.ConveniosActivos); err != nil {
		return detalle, err
	}

	periodoRows, err := r.pool.Query(ctx,
		`SELECT EXTRACT(YEAR FROM fecha_creacion)::int AS anio,
		        COUNT(*)::int                          AS cantidad_historias
		 FROM historia_clinica
		 WHERE entidad_id = $1
		 GROUP BY anio
		 ORDER BY anio DESC`,
		id,
	)
	if err != nil {
		return detalle, err
	}
	defer periodoRows.Close()

	detalle.PeriodosHistoricos = make([]models.PeriodoHistorico, 0)
	for periodoRows.Next() {
		var p models.PeriodoHistorico
		if err := periodoRows.Scan(&p.Anio, &p.CantidadHistorias); err != nil {
			return detalle, err
		}
		detalle.PeriodosHistoricos = append(detalle.PeriodosHistoricos, p)
	}
	if err := periodoRows.Err(); err != nil {
		return detalle, err
	}

	usuarioRows, err := r.pool.Query(ctx,
		`SELECT u.id, u.nombre_usuario, u.apellidos, u.tipo_usuario
		 FROM usuario u
		 WHERE u.id IN (
		     SELECT usuario_id FROM medico WHERE entidad_id = $1
		     UNION
		     SELECT usuario_id FROM administrador_entidad WHERE entidad_id = $1
		 )
		 ORDER BY u.apellidos, u.nombre_usuario`,
		id,
	)
	if err != nil {
		return detalle, err
	}
	defer usuarioRows.Close()

	detalle.UsuariosAsociados = make([]models.UsuarioAsociado, 0)
	for usuarioRows.Next() {
		var u models.UsuarioAsociado
		if err := usuarioRows.Scan(&u.ID, &u.NombreUsuario, &u.Apellidos, &u.TipoUsuario); err != nil {
			return detalle, err
		}
		detalle.UsuariosAsociados = append(detalle.UsuariosAsociados, u)
	}
	if err := usuarioRows.Err(); err != nil {
		return detalle, err
	}

	return detalle, nil
}

type PlatformStats struct {
	TotalConsultas        int
	TotalPacientesActivos int
	TotalUsuariosActivos  int
}

func (r *EntidadRepository) Stats(ctx context.Context) (PlatformStats, error) {
	var s PlatformStats
	err := r.pool.QueryRow(ctx,
		`SELECT
		    (SELECT COUNT(*) FROM consulta)                          AS total_consultas,
		    (SELECT COUNT(*) FROM paciente WHERE estado = true)      AS total_pacientes_activos,
		    (SELECT COUNT(*) FROM usuario  WHERE estado = true)      AS total_usuarios_activos`,
	).Scan(&s.TotalConsultas, &s.TotalPacientesActivos, &s.TotalUsuariosActivos)
	return s, err
}
