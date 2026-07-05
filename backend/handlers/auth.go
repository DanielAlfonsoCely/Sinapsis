package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"sinapsis-backend/config"
	"sinapsis-backend/models"
)

type AuthHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewAuthHandler(pool *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{pool: pool, cfg: cfg}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Contrasena), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process password"})
		return
	}

	var user models.Usuario
	var id uuid.UUID
	err = h.pool.QueryRow(
		context.Background(),
		`INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario, fecha_creacion, fecha_actualizacion)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		 RETURNING id, nombre_usuario, apellidos, email, tipo_usuario, fecha_creacion, fecha_actualizacion`,
		req.NombreUsuario, req.Apellidos, req.Email, string(hashedPassword), req.TipoUsuario,
	).Scan(&id, &user.NombreUsuario, &user.Apellidos, &user.Email, &user.TipoUsuario, &user.FechaCreacion, &user.FechaActualizacion)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
			return
		}
		log.Printf("register error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	user.ID = id

	c.JSON(http.StatusCreated, gin.H{
		"id":                  user.ID,
		"nombre_usuario":      user.NombreUsuario,
		"apellidos":           user.Apellidos,
		"email":               user.Email,
		"tipo_usuario":        user.TipoUsuario,
		"fecha_creacion":      user.FechaCreacion,
		"fecha_actualizacion": user.FechaActualizacion,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.Usuario
	var id uuid.UUID
	err := h.pool.QueryRow(
		context.Background(),
		`SELECT id, nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario, fecha_creacion, fecha_actualizacion
		 FROM usuario WHERE email = $1`,
		req.Email,
	).Scan(&id, &user.NombreUsuario, &user.Apellidos, &user.Email, &user.Contrasena, &user.TipoUsuario, &user.FechaCreacion, &user.FechaActualizacion)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	user.ID = id

	if err := bcrypt.CompareHashAndPassword([]byte(user.Contrasena), []byte(req.Contrasena)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	// Si es médico, buscar especialidad y entidad para el frontend.
	var especialidad *string
	var entidadNombre *string
	if user.TipoUsuario == "medico" {
		var esp, ent string
		err := h.pool.QueryRow(
			context.Background(),
			`SELECT m.especialidad, e.nombre_entidad
			 FROM medico m
			 JOIN entidad e ON e.id = m.entidad_id
			 WHERE m.usuario_id = $1`,
			user.ID,
		).Scan(&esp, &ent)
		if err == nil {
			especialidad = &esp
			entidadNombre = &ent
		}
	}

	claims := jwt.MapClaims{
		"user_id":      user.ID,
		"email":        user.Email,
		"tipo_usuario": user.TipoUsuario,
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
		"iat":          time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: tokenString,
		Usuario: gin.H{
			"id":             user.ID,
			"nombre_usuario": user.NombreUsuario,
			"apellidos":      user.Apellidos,
			"email":          user.Email,
			"tipo_usuario":   user.TipoUsuario,
			"especialidad":   especialidad,
			"entidad":        entidadNombre,
		},
	})
}
