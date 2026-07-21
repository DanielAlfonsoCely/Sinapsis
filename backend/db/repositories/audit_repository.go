package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sinapsis-backend/models"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

// Record construye un AuditLogEntry a partir de los parámetros sueltos y
// delega en Insert, para no duplicar la sentencia SQL.
// FIX: antes hacía panic("unimplemented"); si algún llamador invocaba
// AuditService.Record, tumbaba el proceso en producción.
func (r *AuditRepository) Record(ctx context.Context, usuarioID uuid.UUID, operacion models.AuditOperation, tabla string, registroID *uuid.UUID, ipOrigen *string, detalles *string) error {
	return r.Insert(ctx, models.AuditLogEntry{
		ID:             uuid.New(),
		UsuarioID:      usuarioID,
		TipoOperacion:  operacion,
		TablaAfectada:  tabla,
		RegistroID:     registroID,
		IPOrigen:       ipOrigen,
		Detalles:       detalles,
		FechaOperacion: time.Now(),
		Gravedad:       models.Critical, // default value; could be parameterized if needed
	})
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Insert(ctx context.Context, entry models.AuditLogEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO bitacora_auditoria
   (id, usuario_id, tipo_operacion, tabla_afectada, registro_id, valores_anteriores, valores_nuevos, ip_origen, detalles, fecha_operacion, gravedad)
 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		entry.ID, entry.UsuarioID, entry.TipoOperacion, entry.TablaAfectada,
		entry.RegistroID, entry.ValoresAnteriores, entry.ValoresNuevos, entry.IPOrigen, entry.Detalles, entry.FechaOperacion, entry.Gravedad,
	)
	return err
}

// ListRecent devuelve la página solicitada junto con el total real de registros
// en bitacora_auditoria (no el tamaño de la página).
// FIX: antes el handler usaba len(entries) como "total", que solo reflejaba
// el tamaño de la página actual (acotado por limit).
// FIX: corregido nombre de tabla audit_log → bitacora_auditoria.
func (r *AuditRepository) ListRecent(ctx context.Context, limit, offset int) ([]models.AuditLogEntry, int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT al.id, al.usuario_id, u.nombre_usuario, u.email, al.tipo_operacion,
		        al.tabla_afectada, al.registro_id, al.ip_origen, al.detalles, al.fecha_operacion, al.gravedad
		 FROM bitacora_auditoria al
		 JOIN usuario u ON u.id = al.usuario_id
		 ORDER BY al.fecha_operacion DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]models.AuditLogEntry, 0)
	for rows.Next() {
		var e models.AuditLogEntry
		if err := rows.Scan(
			&e.ID, &e.UsuarioID, &e.UsuarioNombre, &e.UsuarioEmail, &e.TipoOperacion,
			&e.TablaAfectada, &e.RegistroID, &e.IPOrigen, &e.Detalles, &e.FechaOperacion, &e.Gravedad,
		); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM bitacora_auditoria`).Scan(&total); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}

func (r *AuditRepository) LookCritical(ctx context.Context, limit, offset int) ([]models.AuditLogEntry, int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT al.id, al.usuario_id, u.nombre_usuario, u.email, al.tipo_operacion,
		        al.tabla_afectada, al.registro_id, al.ip_origen, al.detalles, al.fecha_operacion, al.gravedad
		 FROM bitacora_auditoria al
		 JOIN usuario u ON u.id = al.usuario_id
		 WHERE al.gravedad IN ('CRITICAL', 'HIGH')
		 ORDER BY al.fecha_operacion DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]models.AuditLogEntry, 0)
	for rows.Next() {
		var e models.AuditLogEntry
		if err := rows.Scan(
			&e.ID, &e.UsuarioID, &e.UsuarioNombre, &e.UsuarioEmail, &e.TipoOperacion,
			&e.TablaAfectada, &e.RegistroID, &e.IPOrigen, &e.Detalles, &e.FechaOperacion, &e.Gravedad,
		); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM bitacora_auditoria WHERE gravedad IN ('CRITICAL', 'HIGH')`).Scan(&total); err != nil {
		return nil, 0, err
	}

	return entries, total, nil
}
