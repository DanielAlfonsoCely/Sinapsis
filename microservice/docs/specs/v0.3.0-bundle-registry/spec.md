# Especificación de Funcionalidad: Registry de Bundles MONAI (v0.3.0)

**Creado**: 2026-07-05

## Escenarios de Usuario y Pruebas *(obligatorio)*

> El "usuario" de esta funcionalidad es el propio sistema: la capa Application
> (`AnalysisService`, v0.4.0) necesitará resolver un `AnalysisType` a un bundle MONAI
> listo para ejecutar. Esta versión entrega el componente de infraestructura que lo hace
> posible, sin ejecutar inferencia real.

### Historia de Usuario 1 - Resolver un tipo de análisis a un bundle listo para usar (Prioridad: P1)

Como capa de aplicación del microservicio, cuando recibo una solicitud con un
`AnalysisType` soportado, necesito obtener un `ModelRef` que identifique el bundle MONAI
correspondiente (nombre, versión y ruta local en caché), descargándolo si aún no está
disponible localmente, para poder ejecutar la inferencia en la versión siguiente.

**Por qué esta prioridad**: Es el núcleo de la versión — sin resolución de bundles no hay
inferencia posible. Aporta valor por sí solo: convierte un tipo de análisis abstracto en
un modelo concreto y localizable.

**Prueba Independiente**: Puede probarse completamente invocando el registry con un
`AnalysisType` conocido y verificando (con la descarga de MONAI mockeada) que se solicita
la descarga al directorio de caché configurado y que se devuelve un `ModelRef` con nombre,
versión y ruta local correctos.

**Escenarios de Aceptación**:

1. **Escenario**: Tipo conocido se resuelve a un ModelRef
   - **Dado** un registry configurado con `BUNDLE_CACHE_DIR` y `BUNDLE_SOURCE`, y un
     `AnalysisType.CT_SPLEEN_SEGMENTATION` mapeado al bundle `spleen_ct_segmentation`
   - **Cuando** se solicita resolver ese tipo de análisis
   - **Entonces** se descarga el bundle (vía `monai.bundle`, mockeado en unit tests) y se
     devuelve un `ModelRef` con `name="spleen_ct_segmentation"`, la versión del bundle y
     la ruta local dentro del directorio de caché

2. **Escenario**: Fallo de descarga se traduce a error del dominio
   - **Dado** un registry cuyo mecanismo de descarga falla (p. ej. error de red o bundle
     inexistente en la fuente)
   - **Cuando** se solicita resolver un tipo conocido no cacheado
   - **Entonces** se lanza `BundleResolutionError` (con la causa original encadenada) y
     no se devuelve ningún `ModelRef`

---

### Historia de Usuario 2 - Rechazar tipos de análisis desconocidos (Prioridad: P2)

Como sistema fail-safe, cuando se solicita resolver un tipo de análisis que no tiene
mapeo registrado a un bundle, necesito que el registry lo rechace explícitamente con un
error del dominio, para que la capa superior lo envíe a dead-letter en lugar de ejecutar
un modelo "a ciegas".

**Por qué esta prioridad**: Es la garantía de seguridad (fail-safe defaults de
DESIGN.md §3): un tipo no soportado nunca debe llegar a inferencia. Es independiente de
la descarga/caché.

**Prueba Independiente**: Puede probarse invocando el registry con un valor sin mapeo
registrado y verificando que lanza `UnknownAnalysisTypeError` sin intentar ninguna
descarga.

**Escenarios de Aceptación**:

1. **Escenario**: Tipo sin mapeo lanza UnknownAnalysisTypeError
   - **Dado** un registry con un catálogo de mapeos `AnalysisType → bundle` definido
   - **Cuando** se solicita resolver un tipo de análisis que no está en el catálogo
   - **Entonces** se lanza `UnknownAnalysisTypeError` y no se realiza ningún intento de
     descarga

---

### Historia de Usuario 3 - Reutilizar bundles cacheados (Prioridad: P3)

Como operador del servicio, necesito que un bundle ya descargado se reutilice desde
`BUNDLE_CACHE_DIR` en resoluciones posteriores, para no re-descargar pesos de cientos de
MB en cada mensaje y mantener latencias y costos de red bajos.

**Por qué esta prioridad**: Optimización esencial para operación (los pesos son pesados),
pero el sistema es funcionalmente correcto sin ella — de ahí P3.

**Prueba Independiente**: Puede probarse resolviendo dos veces el mismo tipo con la
descarga mockeada y verificando que el mecanismo de descarga se invoca exactamente una
vez (o cero veces si el bundle ya existe en el directorio de caché).

**Escenarios de Aceptación**:

1. **Escenario**: Segunda resolución no vuelve a descargar
   - **Dado** un registry que ya resolvió (y descargó) el bundle de un tipo de análisis
   - **Cuando** se solicita resolver el mismo tipo nuevamente
   - **Entonces** se devuelve un `ModelRef` equivalente sin invocar la descarga de nuevo

2. **Escenario**: Bundle presente en caché de un proceso anterior
   - **Dado** un directorio de caché que ya contiene el bundle completo (p. ej. de una
     ejecución previa del worker)
   - **Cuando** un registry recién construido resuelve ese tipo de análisis
   - **Entonces** detecta el bundle en caché y devuelve el `ModelRef` sin descargar

---

### Casos Límite

- ¿Qué ocurre cuando `BUNDLE_CACHE_DIR` no existe? → El registry lo crea (junto con
  directorios intermedios) antes de descargar; no falla por directorio ausente.
- ¿Qué ocurre si el bundle está en caché pero su `metadata.json` no existe o no declara
  versión? → Se devuelve `ModelRef` con `version=""` (la versión es informativa; la ruta
  local es lo operativo).
- ¿Qué ocurre si la descarga deja un bundle parcial/corrupto (interrupción)? → La
  detección de "ya cacheado" debe basarse en un criterio verificable (p. ej. presencia de
  la estructura mínima del bundle); si el criterio no se cumple, se re-descarga.
- ¿Qué ocurre si un miembro nuevo de `AnalysisType` existe en el enum pero nadie registró
  su mapeo a bundle? → `UnknownAnalysisTypeError` (mismo tratamiento que HU2), nunca un
  `KeyError` crudo.
- ¿Qué ocurre con la concurrencia? → No aplica dentro de un proceso (`prefetch=1`, flujo
  síncrono); no se requiere locking del caché en esta versión.

## Requisitos *(obligatorio)*

### Requisitos Funcionales

- **RF-001**: El sistema DEBE mantener en `infrastructure/bundle_registry.py` un catálogo
  único que mapee cada `AnalysisType` soportado al nombre de su bundle MONAI
  (`CT_SPLEEN_SEGMENTATION → "spleen_ct_segmentation"`).
- **RF-002**: El registry DEBE exponer una operación de resolución que reciba un
  `AnalysisType` y devuelva un `ModelRef` del dominio con nombre del bundle, versión y
  ruta local del bundle en caché.
- **RF-003**: El registry DEBE descargar el bundle mediante la API de `monai.bundle`
  usando `BUNDLE_CACHE_DIR` como destino y `BUNDLE_SOURCE` como fuente de descarga
  (valores provistos vía `Settings`, nunca hardcodeados).
- **RF-004**: La entidad `ModelRef` (`domain/models.py`) DEBE extenderse con la ruta
  local del bundle (según DESIGN.md §5: "bundle resuelto: name, version, ruta local en
  caché") sin romper el contrato de mensajes: `presentation/schemas.py` DEBE seguir
  serializando únicamente `name` y `version`.
- **RF-005**: Ante un `AnalysisType` sin mapeo en el catálogo, el registry DEBE lanzar
  `UnknownAnalysisTypeError` sin intentar descargas.
- **RF-006**: Ante un fallo al descargar o cargar el bundle, el registry DEBE lanzar
  `BundleResolutionError` encadenando la excepción original (`raise ... from ...`).
- **RF-007**: Si el bundle ya está disponible en `BUNDLE_CACHE_DIR`, el registry NO DEBE
  volver a descargarlo; DEBE reutilizarlo y devolver el `ModelRef` correspondiente.
- **RF-008**: El registry DEBE crear `BUNDLE_CACHE_DIR` si no existe.
- **RF-009**: El registry DEBE obtener la versión del bundle desde los metadatos del
  bundle cacheado (`metadata.json`); si no está disponible, DEBE usar cadena vacía.
- **RF-010**: El registry NO DEBE importar módulos de las capas `application/` ni
  `presentation/` (regla de dependencias: Infrastructure → Domain únicamente), y `monai`
  DEBE quedar confinado a la capa de infraestructura.
- **RF-011**: Los unit tests DEBEN mockear la descarga de MONAI (sin red, sin descargar
  pesos reales) y cubrir los cuatro escenarios del CHANGELOG v0.3.0.
- **RF-012**: El logging del registry DEBE limitarse a metadatos operativos (tipo de
  análisis, nombre/versión del bundle, cache hit/miss); nunca PII ni datos de paciente.

### Entidades Clave

- **`ModelRef`** *(existente, se extiende)*: referencia al bundle resuelto. Pasa de
  (`name`, `version`) a (`name`, `version`, ruta local en caché). La ruta local es de uso
  interno (Application/Infrastructure) y no viaja en el contrato de mensajes.
- **Catálogo de bundles** *(nuevo, interno al registry)*: mapeo estático
  `AnalysisType → nombre de bundle MONAI`. Fuente única de verdad para la extensión
  descrita en DESIGN.md §11 (agregar un tipo = agregar enum + entrada al catálogo).

## Criterios de Éxito *(obligatorio)*

### Resultados Medibles

- **CE-001**: Los cuatro escenarios de test del CHANGELOG v0.3.0 pasan en
  `tests/unit/infrastructure/test_bundle_registry.py`:
  `test_resolve_known_type_returns_modelref`, `test_resolve_unknown_type_raises`,
  `test_bundle_cached_not_redownloaded`,
  `test_download_failure_raises_bundle_resolution_error`.
- **CE-002**: La secuencia completa de verificación pasa sin errores:
  `uv run ruff format --check .` → `uv run ruff check .` → `uv run mypy src` →
  `uv run pytest`.
- **CE-003**: La suite unitaria completa se ejecuta sin acceso a red y sin descargar
  pesos de modelos (descarga de MONAI 100% mockeada).
- **CE-004**: Cobertura de `src/sinapsis_ai/infrastructure/bundle_registry.py` ≥ 85%
  (umbral del proyecto, DESIGN.md §6).
- **CE-005**: Los tests existentes de v0.1.0 y v0.2.0 siguen pasando sin modificaciones
  de contrato (la extensión de `ModelRef` es retrocompatible).
