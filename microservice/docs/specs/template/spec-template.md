# Especificación de Funcionalidad: [NOMBRE DE LA FUNCIONALIDAD]

**Creado**: [FECHA]  

## Escenarios de Usuario y Pruebas *(obligatorio)*

<!--
  IMPORTANTE: Las historias de usuario deben ser PRIORIZADAS como recorridos de
  usuario ordenados por importancia. Cada historia/recorrido de usuario debe ser
  TESTEABLE DE FORMA INDEPENDIENTE - lo que significa que si implementas solo UNA
  de ellas, deberías tener aún un MVP (Producto Mínimo Viable) que aporte valor.
  
  Asignar prioridades (P1, P2, P3, etc.) a cada historia, donde P1 es la más crítica.
  Pensar en cada historia como un segmento autónomo de funcionalidad que puede:
  - Desarrollarse de forma independiente
  - Testearse de forma independiente
  - Desplegarse de forma independiente
  - Demostrarse a usuarios de forma independiente
-->

### Historia de Usuario 1 - [Título Breve] (Prioridad: P1)

[Describir este recorrido de usuario en lenguaje simple]

**Por qué esta prioridad**: [Explicar el valor y por qué tiene este nivel de prioridad]

**Prueba Independiente**: [Describir cómo puede testearse de forma independiente - ej., "Puede probarse completamente mediante [acción específica] y aporta [valor específico]"]

**Escenarios de Aceptación**:

1. **Escenario**: [Nombre descriptivo del escenario]
   - **Dado** [estado inicial]
   - **Cuando** [acción]
   - **Entonces** [resultado esperado]

2. **Escenario**: [Nombre descriptivo del escenario]
   - **Dado** [estado inicial]
   - **Cuando** [acción]
   - **Entonces** [resultado esperado]

---

### Historia de Usuario 2 - [Título Breve] (Prioridad: P2)

[Describir este recorrido de usuario en lenguaje simple]

**Por qué esta prioridad**: [Explicar el valor y por qué tiene este nivel de prioridad]

**Prueba Independiente**: [Describir cómo puede testearse de forma independiente]

**Escenarios de Aceptación**:

1. **Escenario**: [Nombre descriptivo del escenario]
   - **Dado** [estado inicial]
   - **Cuando** [acción]
   - **Entonces** [resultado esperado]

---

[Agregar más historias de usuario según sea necesario, cada una con una prioridad asignada]

### Casos Límite

<!--
  ACCIÓN REQUERIDA: El contenido de esta sección representa marcadores de posición.
  Completarlos con los casos límite correctos.
-->

- ¿Qué ocurre cuando [condición de borde]?
- ¿Cómo maneja el sistema [escenario de error]?

## Requisitos *(obligatorio)*

<!--
  ACCIÓN REQUERIDA: El contenido de esta sección representa marcadores de posición.
  Completarlos con los requisitos funcionales correctos.
-->

### Requisitos Funcionales

- **RF-001**: El sistema DEBE [capacidad específica, ej., "permitir a los usuarios crear cuentas"]
- **RF-002**: El sistema DEBE [capacidad específica, ej., "validar direcciones de correo electrónico"]
- **RF-003**: Los usuarios DEBEN poder [interacción clave, ej., "restablecer su contraseña"]
- **RF-004**: El sistema DEBE [requisito de datos, ej., "persistir las preferencias del usuario"]
- **RF-005**: El sistema DEBE [comportamiento, ej., "registrar todos los eventos de seguridad"]

*Ejemplo de cómo marcar requisitos poco claros:*

- **RF-006**: El sistema DEBE autenticar usuarios mediante [REQUIERE ACLARACIÓN: método de autenticación no especificado - correo/contraseña, SSO, OAuth?]
- **RF-007**: El sistema DEBE conservar los datos del usuario durante [REQUIERE ACLARACIÓN: período de retención no especificado]

### Entidades Clave *(incluir si la funcionalidad involucra datos)*

- **[Entidad 1]**: [Qué representa, atributos clave sin implementación]
- **[Entidad 2]**: [Qué representa, relaciones con otras entidades]

## Criterios de Éxito *(obligatorio)*

<!--
  ACCIÓN REQUERIDA: Definir criterios de éxito medibles.
  Deben ser agnósticos a la tecnología y medibles.
-->

### Resultados Medibles

- **CE-001**: [Métrica medible, ej., "Los usuarios pueden completar la creación de cuenta en menos de 2 minutos"]
- **CE-002**: [Métrica medible, ej., "El sistema maneja 1000 usuarios simultáneos sin degradación"]
- **CE-003**: [Métrica de satisfacción del usuario, ej., "El 90% de los usuarios completan la tarea principal en el primer intento"]
- **CE-004**: [Métrica de negocio, ej., "Reducir los tickets de soporte relacionados con [X] en un 50%"]
