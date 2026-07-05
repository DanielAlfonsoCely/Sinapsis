package handlers
//agregar imports necesarios
import (
    "context"
    "fmt"
    "log"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "sinapsis-backend/models" // Asegúrate de reemplazar con la ruta correcta a tu paquete de modelos
)

func CrearUsuario(c *gin.Context) {
    // 1. Verificar autenticación
    userIDRaw, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "No autenticado"})
        return
    }
    userID, err := uuid.Parse(userIDRaw.(string))
    if err != nil {
        c.JSON(401, gin.H{"error": "Token inválido"})
        return
    }
    
    // 2. Verificar que sea admin_plataforma
    userType, exists := c.Get("tipo_usuario")
    if !exists || userType != "admin_plataforma" {
        c.JSON(403, gin.H{"error": "Solo el administrador de plataforma puede crear usuarios"})
        return
    }
    
    // 3. Parsear el JSON
    var req models.RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 4. Hash de la contraseña
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(500, gin.H{"error": "Error al procesar la contraseña"})
        return
    }
    
    // 5. Insertar en la BD
    ctx := context.Background()
    var id uuid.UUID
    var fechaCreacion time.Time
    
    err = h.pool.QueryRow(ctx,
        `INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario, fecha_creacion, fecha_actualizacion, estado)
         VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), 'activo')
         RETURNING id, fecha_creacion`,
        req.NombreUsuario, req.Apellidos, req.Email, string(hashedPassword), req.TipoUsuario,
    ).Scan(&id, &fechaCreacion)
    
    if err != nil {
        // Verificar si es error de email duplicado
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(409, gin.H{"error": "Ya existe un usuario con ese email"})
            return
        }
        log.Printf("Error al crear usuario: %v", err)
        c.JSON(500, gin.H{"error": "Error al crear el usuario"})
        return
    }
    
    // 6. Devolver respuesta (sin la contraseña)
    c.JSON(201, gin.H{
        "id":             id,
        "nombre_usuario": req.NombreUsuario,
        "apellidos":      req.Apellidos,
        "email":          req.Email,
        "tipo_usuario":   req.TipoUsuario,
        "fecha_creacion": fechaCreacion,
        "estado":         "activo",
    })
}

func EditarUsuario(c *gin.Context) {
    // 1. Verificar autenticación
    userIDRaw, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "No autenticado"})
        return
    }
    _, err := uuid.Parse(userIDRaw.(string))
    if err != nil {
        c.JSON(401, gin.H{"error": "Token inválido"})
        return
    }

    // 2. Verificar que sea admin_plataforma
    userType, exists := c.Get("tipo_usuario")
    if !exists || userType != "admin_plataforma" {
        c.JSON(403, gin.H{"error": "Solo el administrador de plataforma puede editar usuarios"})
        return
    }

    // 3. Obtener ID del usuario a editar
    targetID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(400, gin.H{"error": "ID de usuario inválido"})
        return
    }

    // 4. Parsear body
    var req models.UpdateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 5. Construir query dinámica (solo actualizar campos que vienen en el request)
    ctx := context.Background()
    query := `UPDATE usuario SET `
    args := []interface{}{}
    argIndex := 1
    setParts := []string{}

    if req.NombreUsuario != nil {
        setParts = append(setParts, fmt.Sprintf("nombre_usuario = $%d", argIndex))
        args = append(args, *req.NombreUsuario)
        argIndex++
    }
    if req.Apellidos != nil {
        setParts = append(setParts, fmt.Sprintf("apellidos = $%d", argIndex))
        args = append(args, *req.Apellidos)
        argIndex++
    }
    if req.Email != nil {
        setParts = append(setParts, fmt.Sprintf("email = $%d", argIndex))
        args = append(args, *req.Email)
        argIndex++
    }
    if req.TipoUsuario != nil {
        setParts = append(setParts, fmt.Sprintf("tipo_usuario = $%d", argIndex))
        args = append(args, *req.TipoUsuario)
        argIndex++
    }
    if req.Estado != nil {
        setParts = append(setParts, fmt.Sprintf("estado = $%d", argIndex))
        args = append(args, *req.Estado)
        argIndex++
    }

    // Siempre actualizar fecha_actualizacion
    setParts = append(setParts, fmt.Sprintf("fecha_actualizacion = NOW()"))

    if len(setParts) == 1 {
        c.JSON(400, gin.H{"error": "No se enviaron campos para actualizar"})
        return
    }

    query += strings.Join(setParts, ", ")
    query += fmt.Sprintf(" WHERE id = $%d AND estado = 'activo'", argIndex)
    args = append(args, targetID)

    // 6. Ejecutar
    tag, err := h.pool.Exec(ctx, query, args...)
    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            c.JSON(409, gin.H{"error": "Ya existe un usuario con ese email"})
            return
        }
        log.Printf("Error al editar usuario: %v", err)
        c.JSON(500, gin.H{"error": "Error al editar el usuario"})
        return
    }

    if tag.RowsAffected() == 0 {
        c.JSON(404, gin.H{"error": "Usuario no encontrado"})
        return
    }

    // 7. Si cambió el rol, invalidar sesiones activas (opcional)
    // ... (podrías agregar lógica para invalidar tokens)

    c.JSON(200, gin.H{"mensaje": "Usuario actualizado correctamente"})
}

func EliminarUsuario(c *gin.Context) {
    // 1. Verificar autenticación
    userIDRaw, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "No autenticado"})
        return
    }
    adminID, err := uuid.Parse(userIDRaw.(string))
    if err != nil {
        c.JSON(401, gin.H{"error": "Token inválido"})
        return
    }

    // 2. Verificar que sea admin_plataforma
    userType, exists := c.Get("tipo_usuario")
    if !exists || userType != "admin_plataforma" {
        c.JSON(403, gin.H{"error": "Solo el administrador de plataforma puede eliminar usuarios"})
        return
    }

    // 3. Obtener ID del usuario a eliminar
    targetID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(400, gin.H{"error": "ID de usuario inválido"})
        return
    }

    // 4. No permitir que el admin se elimine a sí mismo
    if targetID == adminID {
        c.JSON(400, gin.H{"error": "No puedes eliminarte a ti mismo"})
        return
    }

    // 5. Desactivar usuario (no eliminar físicamente)
    ctx := context.Background()
    tag, err := h.pool.Exec(ctx,
        `UPDATE usuario SET estado = 'inactivo', fecha_actualizacion = NOW() 
         WHERE id = $1 AND estado = 'activo'`,
        targetID,
    )
    if err != nil {
        log.Printf("Error al eliminar usuario: %v", err)
        c.JSON(500, gin.H{"error": "Error al eliminar el usuario"})
        return
    }

    if tag.RowsAffected() == 0 {
        c.JSON(404, gin.H{"error": "Usuario no encontrado o ya inactivo"})
        return
    }

    // 6. Opcional: invalidar sesiones activas del usuario eliminado

    c.JSON(200, gin.H{"mensaje": "Usuario desactivado correctamente"})
}

func AsignarRol(c *gin.Context) {
    // 1. Verificar autenticación
    userIDRaw, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "No autenticado"})
        return
    }
    _, err := uuid.Parse(userIDRaw.(string))
    if err != nil {
        c.JSON(401, gin.H{"error": "Token inválido"})
        return
    }

    // 2. Verificar que sea admin_plataforma
    userType, exists := c.Get("tipo_usuario")
    if !exists || userType != "admin_plataforma" {
        c.JSON(403, gin.H{"error": "Solo el administrador de plataforma puede asignar roles"})
        return
    }

    // 3. Obtener ID del usuario
    targetID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(400, gin.H{"error": "ID de usuario inválido"})
        return
    }

    // 4. Parsear body
    var req models.RoleRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 5. Validar que el rol sea válido
    validRoles := []string{"medico", "paciente", "admin_entidad", "admin_plataforma"}
    if !contains(validRoles, req.TipoUsuario) {
        c.JSON(400, gin.H{"error": "Rol inválido. Roles permitidos: medico, paciente, admin_entidad, admin_plataforma"})
        return
    }

    // 6. Actualizar el rol
    ctx := context.Background()
    tag, err := h.pool.Exec(ctx,
        `UPDATE usuario SET tipo_usuario = $1, fecha_actualizacion = NOW() 
         WHERE id = $2 AND estado = 'activo'`,
        req.TipoUsuario, targetID,
    )
    if err != nil {
        log.Printf("Error al asignar rol: %v", err)
        c.JSON(500, gin.H{"error": "Error al asignar el rol"})
        return
    }

    if tag.RowsAffected() == 0 {
        c.JSON(404, gin.H{"error": "Usuario no encontrado o inactivo"})
        return
    }

    // 7. Invalidar sesiones activas (para que el cambio surta efecto inmediato)
    // Esto se hace típicamente con una blacklist de tokens o cambiando el JWT secret

    c.JSON(200, gin.H{"mensaje": "Rol asignado correctamente"})
}


func ObtenerUsuarios(c *gin.Context) {
    // 1. Verificar autenticación
    _, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "No autenticado"})
        return
    }

    // 2. Verificar que sea admin_plataforma
    userType, exists := c.Get("tipo_usuario")
    if !exists || userType != "admin_plataforma" {
        c.JSON(403, gin.H{"error": "Solo el administrador de plataforma puede ver usuarios"})
        return
    }

    // 3. Obtener filtros de query params
    search := c.Query("search")
    rol := c.Query("rol")
    estado := c.Query("estado")
    limit := c.DefaultQuery("limit", "20")
    offset := c.DefaultQuery("offset", "0")

    // 4. Construir query con filtros
    ctx := context.Background()
    query := `SELECT id, nombre_usuario, apellidos, email, tipo_usuario, estado, fecha_creacion, fecha_actualizacion
              FROM usuario WHERE 1=1`
    args := []interface{}{}
    argIndex := 1

    if search != "" {
        query += fmt.Sprintf(` AND (nombre_usuario ILIKE $%d OR apellidos ILIKE $%d OR email ILIKE $%d)`, argIndex, argIndex+1, argIndex+2)
        searchPattern := "%" + search + "%"
        args = append(args, searchPattern, searchPattern, searchPattern)
        argIndex += 3
    }
    if rol != "" {
        query += fmt.Sprintf(` AND tipo_usuario = $%d`, argIndex)
        args = append(args, rol)
        argIndex++
    }
    if estado != "" {
        query += fmt.Sprintf(` AND estado = $%d`, argIndex)
        args = append(args, estado)
        argIndex++
    }

    query += ` ORDER BY fecha_creacion DESC`
    query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIndex, argIndex+1)
    args = append(args, limit, offset)

    rows, err := h.pool.Query(ctx, query, args...)
    if err != nil {
        log.Printf("Error al obtener usuarios: %v", err)
        c.JSON(500, gin.H{"error": "Error al obtener usuarios"})
        return
    }
    defer rows.Close()

    // 5. Procesar resultados
    usuarios := []models.Usuario{}
    for rows.Next() {
        var u models.Usuario
        err := rows.Scan(&u.ID, &u.NombreUsuario, &u.Apellidos, &u.Email,
            &u.TipoUsuario, &u.Estado, &u.FechaCreacion, &u.FechaActualizacion)
        if err != nil {
            log.Printf("Error al escanear usuario: %v", err)
            c.JSON(500, gin.H{"error": "Error al leer usuarios"})
            return
        }
        usuarios = append(usuarios, u)
    }

    // 6. Obtener total (para paginación)
    var total int
    countQuery := `SELECT COUNT(*) FROM usuario WHERE 1=1`
    // ... (repetir filtros sin LIMIT/OFFSET)

    c.JSON(200, gin.H{
        "usuarios": usuarios,
        "total":    total,
        "limit":    limit,
        "offset":   offset,
    })
}