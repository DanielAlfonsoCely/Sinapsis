# Plan de Implementación: [FUNCIONALIDAD]

**Fecha**: [FECHA]  
**Especificación**: [enlace]

## Resumen

[Extraer de la especificación de funcionalidad: requisito principal + enfoque técnico de la investigación]

## Contexto Técnico

<!--
  ACCIÓN REQUERIDA: Reemplazar el contenido de esta sección con los detalles
  técnicos del proyecto. La estructura presentada aquí tiene carácter orientativo
  para guiar el proceso de iteración.
-->

**Lenguaje/Versión**: [ej., Python 3.11, Swift 5.9, Rust 1.75 o REQUIERE ACLARACIÓN]  
**Dependencias Principales**: [ej., FastAPI, UIKit, LLVM o REQUIERE ACLARACIÓN]  
**Almacenamiento**: [si aplica, ej., PostgreSQL, CoreData, archivos o N/A]  
**Testing**: [ej., pytest, XCTest, cargo test o REQUIERE ACLARACIÓN]  
**Plataforma Objetivo**: [ej., servidor Linux, iOS 15+, WASM o REQUIERE ACLARACIÓN]  
**Tipo de Proyecto**: [individual/web/móvil - determina la estructura del código fuente]  
**Objetivos de Rendimiento**: [específicos del dominio, ej., 1000 req/s, 10k líneas/seg, 60 fps o REQUIERE ACLARACIÓN]  
**Restricciones**: [específicas del dominio, ej., <200ms p95, <100MB memoria, funcionalidad offline o REQUIERE ACLARACIÓN]  
**Escala/Alcance**: [específicos del dominio, ej., 10k usuarios, 1M LOC, 50 pantallas o REQUIERE ACLARACIÓN]

## Estructura del Proyecto

### Documentación (esta funcionalidad)

```text
docs/specs/[funcionalidad]
├── plan.md  # Este archivo
└── spec.md
```

### Código Fuente (raíz del repositorio)
<!--
  ACCIÓN REQUERIDA: Reemplazar el árbol de marcadores de posición a continuación
  con la estructura concreta para esta funcionalidad. Eliminar las opciones no
  utilizadas y ampliar la estructura elegida con rutas reales (ej., apps/admin,
  packages/algo). El plan entregado no debe incluir etiquetas de Opción.
-->

```text
# [ELIMINAR SI NO SE USA] Opción 1: Proyecto único (POR DEFECTO)
src/
├── models/
├── services/
├── cli/
└── lib/

tests/
├── contract/
├── integration/
└── unit/

# [ELIMINAR SI NO SE USA] Opción 2: Aplicación web (cuando se detecta "frontend" + "backend")
backend/
├── src/
│   ├── models/
│   ├── services/
│   └── api/
└── tests/

frontend/
├── src/
│   ├── components/
│   ├── pages/
│   └── services/
└── tests/
```

**Decisión de Estructura**: [Documentar la estructura seleccionada y referenciar los directorios reales capturados arriba]


<!-- 
  ============================================================================
  IMPORTANTE: Las tareas a continuación son TAREAS DE EJEMPLO con fines
  ilustrativos únicamente.
  
  DEBEN reemplazarse con tareas reales basadas en:
  - Historias de usuario del spec.md
  - Requisitos de funcionalidad de este archivo
  - Entidades requeridas para el caso de uso
  - Endpoints requeridos
  
  NO conservar estas tareas de ejemplo.
  ============================================================================
-->

## Fase 1: Configuración (Infraestructura Compartida)

**Propósito**: Inicialización del proyecto y estructura básica

- [ ] T001 Crear estructura del proyecto según el plan de implementación
- [ ] T002 Inicializar proyecto [lenguaje] con dependencias de [framework]
- [ ] T003 Configurar herramientas de linting y formateo

---

## Fase 2: Fundacional (Prerequisitos Bloqueantes)

**Propósito**: Infraestructura central que DEBE estar completa antes de poder implementar CUALQUIER historia de usuario

**⚠️ CRÍTICO**: Ningún trabajo de historia de usuario puede comenzar hasta que esta fase esté completa

Ejemplos de tareas fundacionales (ajustar según el proyecto):

- [ ] T004 Configurar esquema de base de datos y framework de migraciones
- [ ] T005 Implementar framework de autenticación/autorización
- [ ] T006 Configurar enrutamiento de API y estructura de middleware
- [ ] T007 Crear modelos/entidades base de los que dependen todas las historias
- [ ] T008 Configurar manejo de errores e infraestructura de logging
- [ ] T009 Configurar gestión de configuración del entorno

**Punto de Control**: Fundación lista - la implementación de historias de usuario puede comenzar en paralelo

---

## Fase 3: Historia de Usuario 1 - [Título] (Prioridad: P1)

**Objetivo**: [Breve descripción de lo que entrega esta historia]

**Prueba Independiente**: [Cómo verificar que esta historia funciona por sí sola]

### Pruebas para Historia de Usuario 1

- [ ] T010 [P] [HU1] Prueba de contrato para [endpoint] en tests/contract/test_[nombre].py
- [ ] T011 [P] [HU1] Prueba de integración para [recorrido de usuario] en tests/integration/test_[nombre].py

### Implementación para Historia de Usuario 1

- [ ] T012 [P] [HU1] Crear modelo [Entidad1] en src/models/[entidad1].py
- [ ] T013 [P] [HU1] Crear modelo [Entidad2] en src/models/[entidad2].py
- [ ] T014 [HU1] Implementar [Servicio] en src/services/[servicio].py (depende de T012, T013)
- [ ] T015 [HU1] Implementar [endpoint/funcionalidad] en src/[ubicación]/[archivo].py
- [ ] T016 [HU1] Agregar validación y manejo de errores
- [ ] T017 [HU1] Agregar logging para operaciones de historia de usuario 1

**Punto de Control**: En este punto, la Historia de Usuario 1 debe ser completamente funcional y testeable de forma independiente

---

## Fase 4: Historia de Usuario 2 - [Título] (Prioridad: P2)

**Objetivo**: [Breve descripción de lo que entrega esta historia]

**Prueba Independiente**: [Cómo verificar que esta historia funciona por sí sola]

### Pruebas para Historia de Usuario 2

- [ ] T018 [P] [HU2] Prueba de contrato para [endpoint] en tests/contract/test_[nombre].py
- [ ] T019 [P] [HU2] Prueba de integración para [recorrido de usuario] en tests/integration/test_[nombre].py

### Implementación para Historia de Usuario 2

- [ ] T020 [P] [HU2] Crear modelo [Entidad] en src/models/[entidad].py
- [ ] T021 [HU2] Implementar [Servicio] en src/services/[servicio].py
- [ ] T022 [HU2] Implementar [endpoint/funcionalidad] en src/[ubicación]/[archivo].py
- [ ] T023 [HU2] Integrar con componentes de Historia de Usuario 1 (si es necesario)

**Punto de Control**: En este punto, las Historias de Usuario 1 Y 2 deben funcionar de forma independiente

---

## Fase 5: Historia de Usuario 3 - [Título] (Prioridad: P3)

**Objetivo**: [Breve descripción de lo que entrega esta historia]

**Prueba Independiente**: [Cómo verificar que esta historia funciona por sí sola]

### Pruebas para Historia de Usuario 3

- [ ] T024 [P] [HU3] Prueba de contrato para [endpoint] en tests/contract/test_[nombre].py
- [ ] T025 [P] [HU3] Prueba de integración para [recorrido de usuario] en tests/integration/test_[nombre].py

### Implementación para Historia de Usuario 3

- [ ] T026 [P] [HU3] Crear modelo [Entidad] en src/models/[entidad].py
- [ ] T027 [HU3] Implementar [Servicio] en src/services/[servicio].py
- [ ] T028 [HU3] Implementar [endpoint/funcionalidad] en src/[ubicación]/[archivo].py

**Punto de Control**: Todas las historias de usuario deben ser ahora funcionalmente independientes

---

[Agregar más fases de historias de usuario según sea necesario, siguiendo el mismo patrón]

---

## Fase N: Acabado y Preocupaciones Transversales

**Propósito**: Mejoras que afectan a múltiples historias de usuario

- [ ] TXXX Actualizaciones de documentación en docs/
- [ ] TXXX Limpieza y refactorización de código
- [ ] TXXX Optimización de rendimiento en todas las historias
- [ ] TXXX Pruebas unitarias adicionales en tests/unit/
- [ ] TXXX Refuerzo de seguridad

---

## Dependencias y Orden de Ejecución

### Dependencias entre Fases

- **Configuración (Fase 1)**: Sin dependencias - puede comenzar de inmediato
- **Fundacional (Fase 2)**: Depende de la finalización de la Configuración - BLOQUEA todas las historias de usuario
- **Historias de Usuario (Fase 3+)**: Todas dependen de la finalización de la fase Fundacional
  - Las historias de usuario pueden proceder en paralelo (si hay personal suficiente)
  - O secuencialmente en orden de prioridad (P1 → P2 → P3)
- **Acabado (Fase Final)**: Depende de que todas las historias de usuario deseadas estén completas

### Dependencias entre Historias de Usuario

- **Historia de Usuario 1 (P1)**: Puede comenzar después de lo Fundacional (Fase 2) - Sin dependencias en otras historias
- **Historia de Usuario 2 (P2)**: Puede comenzar después de lo Fundacional (Fase 2) - Puede integrarse con HU1 pero debe ser testeable de forma independiente
- **Historia de Usuario 3 (P3)**: Puede comenzar después de lo Fundacional (Fase 2) - Puede integrarse con HU1/HU2 pero debe ser testeable de forma independiente

### Dentro de Cada Historia de Usuario

- Modelos antes que servicios
- Servicios antes que endpoints
- Implementación central antes que integración
- Historia completa antes de pasar a la siguiente prioridad
- Pruebas después de la implementación

## Notas

- La etiqueta [Historia] mapea la tarea a una historia de usuario específica para trazabilidad
- Cada historia de usuario debe ser completable y testeable de forma independiente
- Verificar que las pruebas pasen
- Hacer commit después de cada tarea o grupo lógico
- Detenerse en cualquier punto de control para validar la historia de forma independiente
- Evitar: tareas vagas, conflictos en el mismo archivo, dependencias entre historias que rompan la independencia
