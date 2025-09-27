# API Movimiento - Stock Ahora

**Sistema de trazabilidad de movimientos de inventario** que procesa mensajes desde RabbitMQ y proporciona APIs de consulta para análisis y reportes.

## Características principales

- **Consumer RabbitMQ**: Procesa movimientos automáticamente desde colas
- **API de Consultas**: Endpoints REST para obtener trazabilidad y métricas
- **Seguridad**: Encriptación AES-256 de datos sensibles
- **Integración**: Compatible con API Stock existente (UUID, PostgreSQL)
- **Métricas**: Sistema completo de métricas y health checks
- **Contenedores**: Desarrollo y producción con Docker

## Estructura del proyecto

```
api-movimiento/
├── main.go                     # Punto de entrada principal
├── go.mod                      # Dependencias Go
├── Dockerfile                  # Imagen Docker optimizada
├── docker-compose.yml          # Stack completo con profiles
├── Makefile                    # Comandos automatizados
├── .env                        # Variables de entorno
├── config/
│   └── config.go              # Configuración AWS y RabbitMQ TLS
├── models/
│   └── models.go              # Modelos de datos compatible con API Stock
├── database/
│   └── database.go            # Conexión PostgreSQL y migraciones
├── services/
│   ├── movimiento_service.go  # Consumer RabbitMQ y lógica principal
│   ├── query_service.go       # Servicios de consulta
│   └── utils.go              # Utilidades y health checks
├── handlers/
│   ├── movimiento_handler.go  # Endpoints HTTP de consulta
│   └── health_handler.go      # Health check y métricas
└── scripts/
    └── test_publisher.go      # Script para enviar mensajes de prueba
```

##  Configuración inicial

### 1. Clonar y configurar proyecto

```bash
git clone <tu-repo>
cd api-movimiento
make setup
```

### 2. Configurar variables de entorno

```bash
# Copiar archivo de ejemplo
cp .env.example .env

# Editar con tus credenciales
nano .env
```

### 3. Generar clave de encriptación

```bash
make generate-key
# Copiar la clave generada al AWS Secrets Manager
```

### 4. Configurar AWS Secret Manager

```json
{
  "database_url": "postgresql://user:pass@host:5432/movimientos_db",
  "encryption_key": "tu_clave_base64_32_bytes",
  "stock_api_url": "http://tu-api-stock:8001",
  "stock_api_key": "tu_api_key",
  "redis_url": "redis://redis:6379/0",
  "rabbitmq_config": {
    "user": "tu_usuario",
    "password": "tu_password", 
    "host": "tu_host_rabbitmq",
    "port": 5671,
    "vhost": "/"
  }
}
```

## Ejecución

### Desarrollo local con RabbitMQ

```bash
# Levantar stack de desarrollo
make up

# Ver logs en tiempo real
make logs-api

# Probar con mensaje de prueba
make test-consumer
```

### Solo producción

```bash
# Stack mínimo de producción
make up-prod
```

### Con monitoreo completo

```bash
# Incluye Grafana y Prometheus
make up-monitoring
```

## Estructura del mensaje RabbitMQ

El API consume mensajes de la cola `movimientos_queue` con esta estructura:

```json
{
  "product_id": "123e4567-e89b-12d3-a456-426614174000",
  "sku_id": "123e4567-e89b-12d3-a456-426614174001",
  "request_id": "123e4567-e89b-12d3-a456-426614174002", 
  "document_id": "123e4567-e89b-12d3-a456-426614174003",
  "tipo_movimiento": "entrada|salida|ajuste",
  "cantidad": 50,
  "usuario_id": "user123",
  "motivo": "Descripción del movimiento",
  "client_account_id": "123e4567-e89b-12d3-a456-426614174004",
  "origen": "ocr|manual|api|sistema",
  "timestamp": "2024-01-15T10:30:00Z",
  "metadata": {
    "documento_origen": "factura_001.pdf",
    "proveedor": "Proveedor XYZ"
  }
}
```

## Endpoints disponibles

### Consultas de movimientos

```http
GET /api/v1/movimientos
    ?product_id={uuid}
    &sku_id={uuid}
    &request_id={uuid}
    &client_account_id={uuid}
    &tipo_movimiento=entrada|salida|ajuste
    &origen=ocr|manual|api
    &limit=100

GET /api/v1/movimientos/producto/{product_id}/trazabilidad
    ?include_requests=true

GET /api/v1/movimientos/sku/{sku_id}
    ?limit=50

GET /api/v1/movimientos/request/{request_id}
    ?limit=100
```

### Sistema

```http
GET /api/v1/health         # Estado del sistema
GET /api/v1/metrics        # Métricas detalladas
```

## Comandos útiles

```bash
# === DESARROLLO ===
make build              # Compilar aplicación
make run               # Ejecutar localmente  
make run-dev           # Auto-reload (requiere air)
make test              # Ejecutar tests
make lint              # Linter
make format            # Formatear código

# === DOCKER ===
make docker-build      # Construir imagen
make up                # Levantar desarrollo
make up-prod           # Levantar producción
make down              # Bajar servicios
make logs              # Ver logs
make restart           # Reiniciar

# === UTILIDADES ===
make health            # Verificar salud
make metrics           # Ver métricas actuales
make test-endpoints    # Probar todos los endpoints
make clean             # Limpiar archivos
make reset             # Reset completo

# === BASE DE DATOS ===
make db-backup         # Backup
make db-restore        # Restaurar
make clean-volumes     # ⚠️ Borrar todos los datos
```

## Perfiles de Docker Compose

- **default**: Solo API y PostgreSQL
- **dev**: + RabbitMQ Management para desarrollo
- **monitoring**: + Prometheus y Grafana
- **production**: + Nginx y SSL

```bash
# Ejemplos de uso
docker-compose --profile dev up -d
docker-compose --profile monitoring up -d
```

## Flujo de procesamiento

1. **Mensaje llega** a `movimientos_queue`
2. **Consumer valida** estructura y datos
3. **Consulta stock actual** del API Stock
4. **Calcula nuevo stock** según tipo de movimiento
5. **Actualiza stock** en API Stock vía HTTP
6. **Guarda movimiento** en PostgreSQL con datos encriptados
7. **Publica notificación** a cola de notificaciones
8. **Confirma mensaje** procesado o envía a cola de error

## Integración con tu API Stock

El servicio se integra automáticamente con tu API Stock:

```go
// Obtener producto
GET {STOCK_API_URL}/products/{product_id}
Authorization: Bearer {STOCK_API_KEY}

// Actualizar stock  
PUT {STOCK_API_URL}/products/{product_id}
Content-Type: application/json
{
  "stock": 150
}
```

## Seguridad

- **Encriptación AES-256** de datos sensibles
- **TLS/SSL** para conexiones RabbitMQ
- **JWT/Bearer tokens** para API Stock
- **Validación UUID** en todos los endpoints
- **Rate limiting** y timeouts
- **Health checks** para monitoreo

## Monitoreo

### Métricas disponibles

- Total de movimientos procesados
- Movimientos por tipo (entrada/salida/ajuste)
- Movimientos por origen (OCR/manual/API)
- Últimos movimientos
- Estado de conexiones (DB/RabbitMQ)
- Tiempo de respuesta de endpoints

### Dashboards Grafana

- Overview del sistema
- Trazabilidad por producto
- Performance