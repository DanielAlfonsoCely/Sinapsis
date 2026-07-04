 -- ==============================================================================
-- 0. EXTENSIONES
-- ==============================================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ==============================================================================
-- 1. ENUMS
-- ==============================================================================
CREATE TYPE tipo_documento_enum AS ENUM ('CC', 'TI', 'CE', 'PA', 'RC');
CREATE TYPE sexo_enum AS ENUM ('M', 'F', 'O');
CREATE TYPE estado_consulta_enum AS ENUM ('programada', 'en_curso', 'completada', 'cancelada', 'no_asistio');
CREATE TYPE estado_formula_enum AS ENUM ('vigente', 'anulada');
CREATE TYPE estado_consentimiento_enum AS ENUM ('pendiente', 'aprobado', 'denegado');
CREATE TYPE tipo_operacion_enum AS ENUM ('crear', 'actualizar', 'eliminar', 'consultar', 'exportar', 'cambiar_permisos', 'usar_ia');
CREATE TYPE tipo_entidad_enum AS ENUM ('IPS', 'EPS', 'clinica', 'hospital', 'consultorio');
CREATE TYPE estado_revision_enum AS ENUM ('pendiente', 'revisada', 'rechazada');
CREATE TYPE tipo_usuario_enum AS ENUM ('medico', 'paciente', 'admin_entidad', 'admin_plataforma');

-- ==============================================================================
-- 2. TABLAS
-- ==============================================================================

CREATE TABLE rol (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre_rol VARCHAR(100) NOT NULL UNIQUE,
    descripcion TEXT,
    permisos JSONB,
    fecha_creacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE usuario (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre_usuario VARCHAR(100) NOT NULL,
    apellidos VARCHAR(100) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    contrasena_hash VARCHAR(255) NOT NULL,
    estado BOOLEAN DEFAULT true NOT NULL,
    fecha_creacion TIMESTAMP DEFAULT NOW(),
    fecha_actualizacion TIMESTAMP DEFAULT NOW(),
    tipo_usuario tipo_usuario_enum NOT NULL
);

CREATE TABLE entidad (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre_entidad VARCHAR(150) NOT NULL,
    tipo_entidad tipo_entidad_enum NOT NULL,
    nit VARCHAR(50) UNIQUE NOT NULL,
    direccion VARCHAR(255),
    telefono VARCHAR(50),
    ciudad VARCHAR(100),
    estado BOOLEAN DEFAULT true NOT NULL,
    fecha_creacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE plataforma (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre_plataforma VARCHAR(150) NOT NULL,
    descripcion TEXT,
    fecha_creacion TIMESTAMP DEFAULT NOW(),
    estado BOOLEAN DEFAULT true NOT NULL
);

CREATE TABLE paciente (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL,
    numero_documento VARCHAR(50) UNIQUE NOT NULL,
    tipo_documento tipo_documento_enum NOT NULL,
    nombre_paciente VARCHAR(100) NOT NULL,
    apellidos_paciente VARCHAR(100) NOT NULL,
    fecha_nacimiento DATE NOT NULL,
    sexo sexo_enum,
    tipo_sangre VARCHAR(5),
    alergias TEXT,
    direccion VARCHAR(255),
    telefono VARCHAR(50),
    email VARCHAR(150),
    contacto_emergencia VARCHAR(100),
    telefono_emergencia VARCHAR(50),
    antecedentes_medicos TEXT,
    medicamentos_actuales TEXT,
    estado_civil VARCHAR(50),
    ocupacion VARCHAR(100),
    aseguradora VARCHAR(100),
    numero_afiliacion VARCHAR(100),
    fecha_registro TIMESTAMP DEFAULT NOW(),
    estado BOOLEAN DEFAULT true NOT NULL
);

CREATE TABLE medico (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL,
    numero_documento VARCHAR(50) UNIQUE NOT NULL,
    especialidad VARCHAR(100) NOT NULL,
    numero_colegiado VARCHAR(50) UNIQUE NOT NULL,
    experiencia_anios INT,
    entidad_id UUID NOT NULL,
    estado BOOLEAN DEFAULT true NOT NULL,
    fecha_registro TIMESTAMP DEFAULT NOW()
);



CREATE TABLE administrador_entidad (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL,
    entidad_id UUID NOT NULL,
    rol_id UUID NOT NULL,
    activo BOOLEAN DEFAULT true NOT NULL,
    fecha_asignacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE administrador_plataforma (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL,
    plataforma_id UUID NOT NULL,
    rol_id UUID NOT NULL,
    permisos_adicionales JSONB,
    activo BOOLEAN DEFAULT true NOT NULL,
    fecha_asignacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE historia_clinica (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paciente_id UUID NOT NULL,
    entidad_id UUID NOT NULL,
    medico_tratante_id UUID,  -- médico responsable actual del paciente (reasignable por remisión)
    fecha_creacion TIMESTAMP DEFAULT NOW(),
    fecha_actualizacion TIMESTAMP DEFAULT NOW(),
    CONSTRAINT uq_paciente_historia UNIQUE (paciente_id)  -- RN-003: un paciente, una historia
);

-- Agenda mínima: cita = turno que el paciente agenda con un médico.
-- Se considera "activa en el horario" cuando estado='programada' y es para hoy.
CREATE TABLE cita (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paciente_id UUID NOT NULL,
    medico_id UUID NOT NULL,
    fecha_hora TIMESTAMP NOT NULL,
    motivo VARCHAR(255),
    estado estado_consulta_enum NOT NULL DEFAULT 'programada',
    fecha_creacion TIMESTAMP DEFAULT NOW()
);

-- Remisión: autorización de un médico general para que su paciente consulte una
-- especialidad. NO cambia el médico tratante; el especialista atiende temporalmente.
CREATE TABLE remision (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paciente_id UUID NOT NULL,
    medico_remitente_id UUID NOT NULL,   -- médico general (tratante) que autoriza
    especialidad VARCHAR(100) NOT NULL,  -- especialidad autorizada
    motivo TEXT,
    estado VARCHAR(20) NOT NULL DEFAULT 'autorizada', -- autorizada / cancelada
    fecha_creacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE consulta (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    historia_clinica_id UUID NOT NULL,
    paciente_id UUID NOT NULL,
    medico_id UUID NOT NULL,
    tipo_consulta VARCHAR(100),
    motivo_consulta TEXT NOT NULL,
    diagnostico_principal VARCHAR(255),
    diagnostico_cie10 VARCHAR(10),         -- código CIE-10 del diagnóstico (estándar Colombia)
    hallazgos_clinicos TEXT,
    anamnesis TEXT,                        -- enfermedad actual
    revision_sistemas TEXT,                -- revisión por sistemas
    examen_fisico TEXT,
    -- Signos vitales (Res. 1995 de 1999)
    presion_arterial VARCHAR(20),          -- ej: 120/80
    frecuencia_cardiaca INT,               -- lpm
    frecuencia_respiratoria INT,           -- rpm
    temperatura DECIMAL(4,1),              -- °C
    saturacion_oxigeno INT,                -- %
    peso_kg DECIMAL(5,2),
    talla_cm DECIMAL(5,2),
    observaciones_medico TEXT,
    plan_manejo TEXT,                      -- plan de tratamiento/manejo
    medicamentos_prescritos TEXT,
    procedimientos_indicados TEXT,
    proxima_cita DATE,
    fecha_consulta TIMESTAMP DEFAULT NOW() NOT NULL,
    duracion_minutos INT,
    estado_consulta estado_consulta_enum NOT NULL DEFAULT 'programada',
    pre_diagnostico TEXT,  -- RN-007: requerido antes de usar IA
    fecha_creacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE formula_medica (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    historia_clinica_id UUID NOT NULL,
    paciente_id UUID NOT NULL,
    medico_id UUID NOT NULL,
    consulta_id UUID,  -- consulta donde se recetó (para enlazar con la HC)
    medicamentos JSONB NOT NULL,
    fecha_prescripcion TIMESTAMP DEFAULT NOW() NOT NULL,
    fecha_vencimiento DATE,
    indicaciones TEXT,
    contraindicaciones TEXT,
    estado_formula estado_formula_enum NOT NULL DEFAULT 'vigente',
    numero_renovaciones_permitidas INT DEFAULT 0,
    numero_renovaciones_realizadas INT DEFAULT 0,
    fecha_creacion TIMESTAMP DEFAULT NOW()
);

CREATE TABLE examinagen (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    historia_clinica_id UUID NOT NULL,
    paciente_id UUID NOT NULL,
    consulta_id UUID,                     -- consulta a la que se adjuntó el anexo (HU-07)
    tipo_examen VARCHAR(100) NOT NULL,    -- categoría: imagen / documento
    descripcion TEXT,                     -- nombre del anexo (ej: "RX tórax")
    url_imagen VARCHAR(255),              -- nombre del archivo almacenado en el volumen
    fecha_examen TIMESTAMP DEFAULT NOW() NOT NULL,
    medico_solicitante_id UUID NOT NULL,
    estado_examen VARCHAR(50),
    observaciones TEXT,
    fecha_carga TIMESTAMP DEFAULT NOW()
);

CREATE TABLE sugerencia_ia (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    examinagen_id UUID NOT NULL,
    historia_clinica_id UUID NOT NULL,
    modelo_ia_utilizado VARCHAR(100) NOT NULL,
    confianza_prediccion DECIMAL(5,2),
    descripcion_hallazgo TEXT,
    diagnostico_sugerido VARCHAR(255),
    fecha_analisis TIMESTAMP DEFAULT NOW(),
    estado_revision estado_revision_enum NOT NULL DEFAULT 'pendiente',
    observaciones_medico TEXT,
    fecha_revision TIMESTAMP,
    medico_revisor_id UUID
);

CREATE TABLE consentimiento_informado (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paciente_id UUID NOT NULL,
    historia_clinica_id UUID NOT NULL,
    tipo_consentimiento VARCHAR(100) NOT NULL,
    descripcion TEXT,
    fecha_generacion DATE NOT NULL,
    fecha_vencimiento DATE,
    estado_consentimiento estado_consentimiento_enum NOT NULL DEFAULT 'pendiente',
    fecha_aceptacion TIMESTAMP,
    observaciones TEXT,
    firma_paciente BOOLEAN DEFAULT false NOT NULL,
    fecha_firma TIMESTAMP
);

CREATE TABLE conversacion_asistente (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    paciente_id UUID NOT NULL,
    pregunta_usuario TEXT NOT NULL,
    respuesta_asistente TEXT NOT NULL,
    contexto_clinico JSONB,
    fecha_mensaje TIMESTAMP DEFAULT NOW() NOT NULL,
    tokens_utilizados INT,
    confianza_respuesta DECIMAL(5,2),
    feedback_util BOOLEAN
);

CREATE TABLE bitacora_auditoria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL,
    tipo_operacion tipo_operacion_enum NOT NULL,
    tabla_afectada VARCHAR(100) NOT NULL,
    registro_id UUID,
    valores_anteriores JSONB,
    valores_nuevos JSONB,
    ip_origen VARCHAR(50),
    fecha_operacion TIMESTAMP DEFAULT NOW() NOT NULL,
    detalles TEXT
);

-- ==============================================================================
-- 3. FOREIGN KEYS
-- ==============================================================================
ALTER TABLE paciente ADD CONSTRAINT fk_paciente_usuario FOREIGN KEY (usuario_id) REFERENCES usuario(id) ON DELETE RESTRICT;

ALTER TABLE medico ADD CONSTRAINT fk_medico_usuario FOREIGN KEY (usuario_id) REFERENCES usuario(id) ON DELETE RESTRICT;
ALTER TABLE medico ADD CONSTRAINT fk_medico_entidad FOREIGN KEY (entidad_id) REFERENCES entidad(id) ON DELETE RESTRICT;

ALTER TABLE ips ADD CONSTRAINT fk_ips_entidad FOREIGN KEY (entidad_id) REFERENCES entidad(id) ON DELETE RESTRICT;

ALTER TABLE administrador_entidad ADD CONSTRAINT fk_adminent_usuario FOREIGN KEY (usuario_id) REFERENCES usuario(id) ON DELETE RESTRICT;
ALTER TABLE administrador_entidad ADD CONSTRAINT fk_adminent_entidad FOREIGN KEY (entidad_id) REFERENCES entidad(id) ON DELETE RESTRICT;
ALTER TABLE administrador_entidad ADD CONSTRAINT fk_adminent_rol FOREIGN KEY (rol_id) REFERENCES rol(id) ON DELETE RESTRICT;

ALTER TABLE administrador_plataforma ADD CONSTRAINT fk_adminplat_usuario FOREIGN KEY (usuario_id) REFERENCES usuario(id) ON DELETE RESTRICT;
ALTER TABLE administrador_plataforma ADD CONSTRAINT fk_adminplat_plataforma FOREIGN KEY (plataforma_id) REFERENCES plataforma(id) ON DELETE RESTRICT;
ALTER TABLE administrador_plataforma ADD CONSTRAINT fk_adminplat_rol FOREIGN KEY (rol_id) REFERENCES rol(id) ON DELETE RESTRICT;

ALTER TABLE historia_clinica ADD CONSTRAINT fk_hc_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE historia_clinica ADD CONSTRAINT fk_hc_entidad FOREIGN KEY (entidad_id) REFERENCES entidad(id) ON DELETE RESTRICT;
ALTER TABLE historia_clinica ADD CONSTRAINT fk_hc_medico_tratante FOREIGN KEY (medico_tratante_id) REFERENCES medico(id) ON DELETE RESTRICT;

ALTER TABLE cita ADD CONSTRAINT fk_cita_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE cita ADD CONSTRAINT fk_cita_medico FOREIGN KEY (medico_id) REFERENCES medico(id) ON DELETE RESTRICT;

ALTER TABLE remision ADD CONSTRAINT fk_remision_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE remision ADD CONSTRAINT fk_remision_medico FOREIGN KEY (medico_remitente_id) REFERENCES medico(id) ON DELETE RESTRICT;

ALTER TABLE consulta ADD CONSTRAINT fk_consulta_hc FOREIGN KEY (historia_clinica_id) REFERENCES historia_clinica(id) ON DELETE RESTRICT;
ALTER TABLE consulta ADD CONSTRAINT fk_consulta_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE consulta ADD CONSTRAINT fk_consulta_medico FOREIGN KEY (medico_id) REFERENCES medico(id) ON DELETE RESTRICT;

ALTER TABLE formula_medica ADD CONSTRAINT fk_formula_hc FOREIGN KEY (historia_clinica_id) REFERENCES historia_clinica(id) ON DELETE RESTRICT;
ALTER TABLE formula_medica ADD CONSTRAINT fk_formula_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE formula_medica ADD CONSTRAINT fk_formula_medico FOREIGN KEY (medico_id) REFERENCES medico(id) ON DELETE RESTRICT;
ALTER TABLE formula_medica ADD CONSTRAINT fk_formula_consulta FOREIGN KEY (consulta_id) REFERENCES consulta(id) ON DELETE RESTRICT;

ALTER TABLE examinagen ADD CONSTRAINT fk_examen_hc FOREIGN KEY (historia_clinica_id) REFERENCES historia_clinica(id) ON DELETE RESTRICT;
ALTER TABLE examinagen ADD CONSTRAINT fk_examen_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE examinagen ADD CONSTRAINT fk_examen_medico FOREIGN KEY (medico_solicitante_id) REFERENCES medico(id) ON DELETE RESTRICT;
ALTER TABLE examinagen ADD CONSTRAINT fk_examen_consulta FOREIGN KEY (consulta_id) REFERENCES consulta(id) ON DELETE RESTRICT;

ALTER TABLE sugerencia_ia ADD CONSTRAINT fk_sia_examen FOREIGN KEY (examinagen_id) REFERENCES examinagen(id) ON DELETE RESTRICT;
ALTER TABLE sugerencia_ia ADD CONSTRAINT fk_sia_hc FOREIGN KEY (historia_clinica_id) REFERENCES historia_clinica(id) ON DELETE RESTRICT;
ALTER TABLE sugerencia_ia ADD CONSTRAINT fk_sia_medico_revisor FOREIGN KEY (medico_revisor_id) REFERENCES medico(id) ON DELETE RESTRICT;

ALTER TABLE consentimiento_informado ADD CONSTRAINT fk_consentimiento_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;
ALTER TABLE consentimiento_informado ADD CONSTRAINT fk_consentimiento_hc FOREIGN KEY (historia_clinica_id) REFERENCES historia_clinica(id) ON DELETE RESTRICT;

ALTER TABLE conversacion_asistente ADD CONSTRAINT fk_conv_paciente FOREIGN KEY (paciente_id) REFERENCES paciente(id) ON DELETE RESTRICT;

ALTER TABLE bitacora_auditoria ADD CONSTRAINT fk_auditoria_usuario FOREIGN KEY (usuario_id) REFERENCES usuario(id) ON DELETE RESTRICT;

-- ==============================================================================
-- 4. ÍNDICES
-- ==============================================================================
CREATE INDEX idx_paciente_usuario ON paciente(usuario_id);
CREATE INDEX idx_paciente_documento ON paciente(numero_documento);
CREATE INDEX idx_medico_usuario ON medico(usuario_id);
CREATE INDEX idx_medico_entidad ON medico(entidad_id);
CREATE INDEX idx_hc_paciente ON historia_clinica(paciente_id);
CREATE INDEX idx_hc_medico_tratante ON historia_clinica(medico_tratante_id);
CREATE INDEX idx_cita_paciente ON cita(paciente_id);
CREATE INDEX idx_cita_medico_fecha ON cita(medico_id, fecha_hora);
CREATE INDEX idx_remision_paciente ON remision(paciente_id);
CREATE INDEX idx_consulta_hc ON consulta(historia_clinica_id);
CREATE INDEX idx_consulta_medico ON consulta(medico_id);
CREATE INDEX idx_consulta_paciente ON consulta(paciente_id);
CREATE INDEX idx_formula_hc ON formula_medica(historia_clinica_id);
CREATE INDEX idx_formula_consulta ON formula_medica(consulta_id);
CREATE INDEX idx_examen_hc ON examinagen(historia_clinica_id);
CREATE INDEX idx_examen_consulta ON examinagen(consulta_id);
CREATE INDEX idx_bitacora_usuario ON bitacora_auditoria(usuario_id);
CREATE INDEX idx_bitacora_fecha ON bitacora_auditoria(fecha_operacion);
CREATE INDEX idx_usuario_email ON usuario(email);
CREATE INDEX idx_consentimiento_paciente ON consentimiento_informado(paciente_id);
CREATE INDEX idx_conversacion_paciente ON conversacion_asistente(paciente_id);

-- ==============================================================================
-- 5. BITÁCORA INMUTABLE (RN-010)
-- ==============================================================================
CREATE OR REPLACE FUNCTION fn_bitacora_inmutable()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'La bitácora de auditoría es inmutable. No se permite % en esta tabla.', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_bitacora_inmutable
BEFORE UPDATE OR DELETE ON bitacora_auditoria
FOR EACH ROW EXECUTE FUNCTION fn_bitacora_inmutable();
