-- ==============================================================================
-- 0. INSERCION DE DATOS INICIALES (SEEDING)
-- Se ejecuta despues de schema.sql
-- ==============================================================================

DO $$
DECLARE
    password_hash CONSTANT text := '$2a$12$BOTeYeKgI9CxSh0vSP/dI.5icXBEQUyfU/w/5qneKG3zw9SlU9SJe';

    plataforma_id uuid;
    rol_admin_entidad_id uuid;
    rol_admin_plataforma_id uuid;
    admin_entidad_usuario_id uuid;
    admin_plataforma_usuario_id uuid;
    entidad_id uuid;
    medico_usuario_id uuid;
    medico_id uuid;
    paciente_usuario_id uuid;
    paciente_id uuid;
    historia_clinica_id uuid;

    entidad_ids uuid[] := ARRAY[]::uuid[];
    paciente_ids uuid[];

    entidad_nombres text[] := ARRAY[
        'Clinica Las Nieves',
        'Hospital San Rafael',
        'IPS Salud Norte',
        'Clinica Santa Lucia',
        'Consultorio Medico Andino'
    ];
    entidad_tipos tipo_entidad_enum[] := ARRAY[
        'clinica'::tipo_entidad_enum,
        'hospital'::tipo_entidad_enum,
        'IPS'::tipo_entidad_enum,
        'clinica'::tipo_entidad_enum,
        'consultorio'::tipo_entidad_enum
    ];
    entidad_nits text[] := ARRAY[
        '900123456-7',
        '901234567-8',
        '902345678-9',
        '903456789-0',
        '904567890-1'
    ];
    entidad_direcciones text[] := ARRAY[
        'Calle Falsa 123',
        'Carrera 45 # 18-20',
        'Avenida Norte # 70-15',
        'Calle 82 # 11-35',
        'Carrera 7 # 120-40'
    ];
    entidad_telefonos text[] := ARRAY[
        '555-0100',
        '555-0200',
        '555-0300',
        '555-0400',
        '555-0500'
    ];
    entidad_ciudades text[] := ARRAY[
        'Bogota D.C.',
        'Medellin',
        'Cali',
        'Barranquilla',
        'Bucaramanga'
    ];
    medicos_por_entidad int[] := ARRAY[3, 4, 2, 5, 3];

    medico_nombres text[] := ARRAY[
        'Carlos', 'Laura', 'Andres', 'Paula', 'Jorge',
        'Camila', 'Miguel', 'Valentina', 'Fernando', 'Diana',
        'Ricardo', 'Natalia', 'Santiago', 'Marcela', 'Felipe',
        'Juliana', 'Oscar'
    ];
    medico_apellidos text[] := ARRAY[
        'Mendoza', 'Rojas', 'Perez', 'Vargas', 'Morales',
        'Castro', 'Herrera', 'Silva', 'Ortega', 'Navarro',
        'Suarez', 'Cortes', 'Mejia', 'Torres', 'Ramirez',
        'Pardo', 'Acosta'
    ];
    especialidades text[] := ARRAY[
        'Medicina General',
        'Pediatria',
        'Medicina Interna',
        'Ginecologia',
        'Cardiologia',
        'Dermatologia',
        'Ortopedia',
        'Neurologia',
        'Psiquiatria',
        'Oftalmologia'
    ];

    paciente_nombres text[] := ARRAY[
        'Ana', 'Luis', 'Sofia', 'Maria', 'Jose', 'Daniela',
        'Juan', 'Carolina', 'Pedro', 'Elena', 'Diego', 'Andrea',
        'Mateo', 'Isabella', 'Sebastian', 'Lucia', 'Nicolas', 'Valeria'
    ];
    paciente_apellidos text[] := ARRAY[
        'Garcia', 'Martinez', 'Rodriguez', 'Lopez', 'Gomez', 'Diaz',
        'Sanchez', 'Romero', 'Alvarez', 'Ruiz', 'Moreno', 'Jimenez',
        'Munoz', 'Castillo', 'Vega', 'Rincon', 'Arias', 'Campos'
    ];
    motivos text[] := ARRAY[
        'Control general',
        'Dolor de cabeza',
        'Seguimiento de tratamiento',
        'Lectura de examenes',
        'Consulta prioritaria',
        'Control de signos vitales'
    ];
    diagnosticos text[] := ARRAY[
        'Rinofaringitis aguda',
        'Lumbago no especificado',
        'Hipertension esencial primaria',
        'Gastritis no especificada',
        'Migraña sin aura',
        'Dermatitis alergica de contacto',
        'Diabetes mellitus tipo 2 sin complicaciones',
        'Infeccion de vias urinarias'
    ];
    diagnosticos_cie10 text[] := ARRAY[
        'J00',
        'M545',
        'I10',
        'K297',
        'G430',
        'L239',
        'E119',
        'N390'
    ];

    entidad_index int;
    medico_index int;
    doctor_counter int := 0;
    paciente_index int;
    patient_counter int := 0;
    consulta_index int;
    day_offset int;
    cita_index int;
    paciente_nombre text;
    paciente_apellido text;
    paciente_email text;
    paciente_documento text;
    paciente_tipo_documento tipo_documento_enum;
    paciente_sexo sexo_enum;
BEGIN
    -- Roles base
    INSERT INTO rol (nombre_rol, descripcion, permisos)
    VALUES (
        'Administrador de Entidad',
        'Rol con permisos completos para la administracion de una entidad.',
        '{"can_manage_users": true, "can_manage_doctors": true, "can_manage_patients": true}'
    )
    RETURNING id INTO rol_admin_entidad_id;

    INSERT INTO rol (nombre_rol, descripcion, permisos)
    VALUES (
        'Administrador de Plataforma',
        'Rol con permisos globales sobre la plataforma.',
        '{"can_manage_entities": true, "can_view_audits": true, "can_manage_platform_admins": true}'
    )
    RETURNING id INTO rol_admin_plataforma_id;

    -- Plataforma
    INSERT INTO plataforma (nombre_plataforma, descripcion)
    VALUES (
        'SINAPSIS Ecosistema de Salud',
        'Plataforma central para la gestion de historias clinicas electronicas.'
    )
    RETURNING id INTO plataforma_id;

    -- Entidades. Se conserva la entidad original como primer registro.
    FOR entidad_index IN 1..array_length(entidad_nombres, 1) LOOP
        INSERT INTO entidad (nombre_entidad, tipo_entidad, nit, direccion, telefono, ciudad)
        VALUES (
            entidad_nombres[entidad_index],
            entidad_tipos[entidad_index],
            entidad_nits[entidad_index],
            entidad_direcciones[entidad_index],
            entidad_telefonos[entidad_index],
            entidad_ciudades[entidad_index]
        )
        RETURNING id INTO entidad_id;

        entidad_ids := array_append(entidad_ids, entidad_id);
    END LOOP;

    -- Administrador de plataforma
    INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario)
    VALUES (
        'Super',
        'Admin',
        'admin.plataforma@sinapsis.com',
        password_hash,
        'admin_plataforma'
    )
    RETURNING id INTO admin_plataforma_usuario_id;

    INSERT INTO administrador_plataforma (usuario_id, plataforma_id, rol_id)
    VALUES (admin_plataforma_usuario_id, plataforma_id, rol_admin_plataforma_id);

    -- Un administrador por entidad, todos con la misma contrasena.
    FOR entidad_index IN 1..array_length(entidad_ids, 1) LOOP
        INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario)
        VALUES (
            'Admin',
            'Entidad ' || entidad_index,
            CASE
                WHEN entidad_index = 1 THEN 'admin.entidad@sinapsis.com'
                ELSE 'admin.entidad' || lpad(entidad_index::text, 2, '0') || '@sinapsis.com'
            END,
            password_hash,
            'admin_entidad'
        )
        RETURNING id INTO admin_entidad_usuario_id;

        INSERT INTO administrador_entidad (usuario_id, entidad_id, rol_id)
        VALUES (admin_entidad_usuario_id, entidad_ids[entidad_index], rol_admin_entidad_id);
    END LOOP;

    -- Medicos, pacientes, historias clinicas y citas.
    FOR entidad_index IN 1..array_length(entidad_ids, 1) LOOP
        FOR medico_index IN 1..medicos_por_entidad[entidad_index] LOOP
            doctor_counter := doctor_counter + 1;

            INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario)
            VALUES (
                medico_nombres[doctor_counter],
                medico_apellidos[doctor_counter],
                CASE
                    WHEN doctor_counter = 1 THEN 'carlos.mendoza@nieves.com'
                    ELSE 'medico' || lpad(doctor_counter::text, 2, '0') || '@sinapsis.com'
                END,
                password_hash,
                'medico'
            )
            RETURNING id INTO medico_usuario_id;

            INSERT INTO medico (
                usuario_id,
                numero_documento,
                especialidad,
                numero_colegiado,
                experiencia_anios,
                entidad_id
            )
            VALUES (
                medico_usuario_id,
                CASE
                    WHEN doctor_counter = 1 THEN '79123456'
                    ELSE '79' || lpad(doctor_counter::text, 6, '0')
                END,
                especialidades[((doctor_counter - 1) % array_length(especialidades, 1)) + 1],
                CASE
                    WHEN doctor_counter = 1 THEN 'MED-12345'
                    ELSE 'MED-SEED-' || lpad(doctor_counter::text, 4, '0')
                END,
                3 + (doctor_counter % 12),
                entidad_ids[entidad_index]
            )
            RETURNING id INTO medico_id;

            paciente_ids := ARRAY[]::uuid[];

            -- Seis pacientes por medico para que cada agenda tenga variedad.
            FOR paciente_index IN 1..6 LOOP
                patient_counter := patient_counter + 1;
                paciente_nombre := paciente_nombres[((patient_counter - 1) % array_length(paciente_nombres, 1)) + 1];
                paciente_apellido := paciente_apellidos[((patient_counter - 1) % array_length(paciente_apellidos, 1)) + 1];
                paciente_tipo_documento := 'CC'::tipo_documento_enum;
                paciente_sexo := CASE WHEN patient_counter % 2 = 0 THEN 'M'::sexo_enum ELSE 'F'::sexo_enum END;

                IF patient_counter = 1 THEN
                    paciente_email := 'ana.garcia@email.com';
                    paciente_documento := '1001234567';
                    paciente_tipo_documento := 'CC'::tipo_documento_enum;
                    paciente_sexo := 'F'::sexo_enum;
                ELSIF patient_counter = 2 THEN
                    paciente_email := 'luis.martinez@email.com';
                    paciente_documento := '1001234568';
                    paciente_tipo_documento := 'CC'::tipo_documento_enum;
                    paciente_sexo := 'M'::sexo_enum;
                ELSIF patient_counter = 3 THEN
                    paciente_email := 'sofia.rodriguez@email.com';
                    paciente_documento := '1001234569';
                    paciente_tipo_documento := 'TI'::tipo_documento_enum;
                    paciente_sexo := 'F'::sexo_enum;
                ELSE
                    paciente_email := 'paciente' || lpad(patient_counter::text, 3, '0') || '@sinapsis.com';
                    paciente_documento := '1002' || lpad(patient_counter::text, 6, '0');
                END IF;

                INSERT INTO usuario (nombre_usuario, apellidos, email, contrasena_hash, tipo_usuario)
                VALUES (
                    paciente_nombre,
                    paciente_apellido,
                    paciente_email,
                    password_hash,
                    'paciente'
                )
                RETURNING id INTO paciente_usuario_id;

                INSERT INTO paciente (
                    usuario_id,
                    numero_documento,
                    tipo_documento,
                    nombre_paciente,
                    apellidos_paciente,
                    fecha_nacimiento,
                    sexo,
                    email,
                    telefono,
                    direccion,
                    aseguradora,
                    numero_afiliacion
                )
                VALUES (
                    paciente_usuario_id,
                    paciente_documento,
                    paciente_tipo_documento,
                    paciente_nombre,
                    paciente_apellido,
                    CASE
                        WHEN patient_counter = 1 THEN DATE '1990-05-15'
                        WHEN patient_counter = 2 THEN DATE '1985-11-20'
                        WHEN patient_counter = 3 THEN DATE '2005-02-10'
                        ELSE (DATE '1975-01-01' + (patient_counter * INTERVAL '97 days'))::date
                    END,
                    paciente_sexo,
                    paciente_email,
                    '300' || lpad(patient_counter::text, 7, '0'),
                    'Direccion paciente ' || patient_counter,
                    entidad_nombres[entidad_index],
                    'AF-' || lpad(patient_counter::text, 6, '0')
                )
                RETURNING id INTO paciente_id;

                paciente_ids := array_append(paciente_ids, paciente_id);

                INSERT INTO historia_clinica (paciente_id, entidad_id, medico_tratante_id)
                VALUES (paciente_id, entidad_ids[entidad_index], medico_id)
                RETURNING id INTO historia_clinica_id;

                -- Tres consultas previas completadas por paciente, en anos historicos.
                FOR consulta_index IN 1..3 LOOP
                    INSERT INTO consulta (
                        historia_clinica_id,
                        paciente_id,
                        medico_id,
                        tipo_consulta,
                        motivo_consulta,
                        diagnostico_principal,
                        diagnostico_cie10,
                        hallazgos_clinicos,
                        anamnesis,
                        revision_sistemas,
                        examen_fisico,
                        presion_arterial,
                        frecuencia_cardiaca,
                        frecuencia_respiratoria,
                        temperatura,
                        saturacion_oxigeno,
                        peso_kg,
                        talla_cm,
                        observaciones_medico,
                        plan_manejo,
                        medicamentos_prescritos,
                        procedimientos_indicados,
                        proxima_cita,
                        fecha_consulta,
                        duracion_minutos,
                        estado_consulta,
                        pre_diagnostico
                    )
                    VALUES (
                        historia_clinica_id,
                        paciente_id,
                        medico_id,
                        CASE consulta_index
                            WHEN 1 THEN 'Consulta de control'
                            WHEN 2 THEN 'Consulta de seguimiento'
                            ELSE 'Consulta prioritaria'
                        END,
                        motivos[((patient_counter + consulta_index - 2) % array_length(motivos, 1)) + 1],
                        diagnosticos[((patient_counter + consulta_index - 2) % array_length(diagnosticos, 1)) + 1],
                        diagnosticos_cie10[((patient_counter + consulta_index - 2) % array_length(diagnosticos_cie10, 1)) + 1],
                        'Paciente estable, sin signos de alarma durante la valoracion.',
                        'Refiere evolucion favorable desde el ultimo control.',
                        'Niega sintomas respiratorios severos, dolor toracico o perdida de conciencia.',
                        'Examen fisico general dentro de parametros esperados.',
                        (110 + ((patient_counter + consulta_index) % 20))::text || '/' || (70 + ((patient_counter + consulta_index) % 12))::text,
                        68 + ((patient_counter + consulta_index) % 24),
                        14 + ((patient_counter + consulta_index) % 6),
                        36.1 + (((patient_counter + consulta_index) % 8)::numeric / 10),
                        94 + ((patient_counter + consulta_index) % 6),
                        55 + ((patient_counter + consulta_index) % 38),
                        150 + ((patient_counter + consulta_index) % 35),
                        'Se explican signos de alarma y recomendaciones generales.',
                        'Continuar manejo ambulatorio, control de habitos y seguimiento periodico.',
                        CASE
                            WHEN consulta_index = 1 THEN 'Acetaminofen 500 mg cada 8 horas si dolor o fiebre.'
                            WHEN consulta_index = 2 THEN 'Loratadina 10 mg cada 24 horas por 5 dias si sintomas alergicos.'
                            ELSE 'No se formulan medicamentos nuevos.'
                        END,
                        CASE
                            WHEN consulta_index = 3 THEN 'Solicitar laboratorios de control segun evolucion.'
                            ELSE 'No requiere procedimientos adicionales.'
                        END,
                        make_date(2023 + consulta_index, ((patient_counter + consulta_index) % 12) + 1, 20),
                        make_timestamp(
                            2022 + consulta_index,
                            ((patient_counter + consulta_index) % 12) + 1,
                            ((patient_counter + consulta_index) % 24) + 1,
                            8 + ((patient_counter + consulta_index) % 8),
                            CASE WHEN consulta_index = 1 THEN 0 WHEN consulta_index = 2 THEN 30 ELSE 15 END,
                            0
                        ),
                        25 + (consulta_index * 5),
                        'completada',
                        'Valoracion inicial sugiere condicion controlada sin criterios de urgencia.'
                    );
                END LOOP;
            END LOOP;

            -- Cuatro citas diarias por medico durante tres dias.
            FOR day_offset IN 0..2 LOOP
                FOR cita_index IN 1..4 LOOP
                    INSERT INTO cita (paciente_id, medico_id, fecha_hora, motivo, estado)
                    VALUES (
                        paciente_ids[((cita_index + day_offset - 1) % array_length(paciente_ids, 1)) + 1],
                        medico_id,
                        CURRENT_DATE
                            + (day_offset * INTERVAL '1 day')
                            + INTERVAL '8 hours'
                            + ((cita_index - 1) * INTERVAL '45 minutes'),
                        motivos[((cita_index + doctor_counter - 2) % array_length(motivos, 1)) + 1],
                        'programada'
                    );
                END LOOP;
            END LOOP;
        END LOOP;
    END LOOP;
END $$;
