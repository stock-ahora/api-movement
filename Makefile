# Makefile para API Movimiento - Stock Ahora

BINARY_NAME=api-movimiento
DOCKER_IMAGE=stock-ahora/api-movimiento
VERSION=1.0.0

.PHONY: help build run test clean docker-build docker-run setup

help: ## Mostrar ayuda
	@echo "🚀 API Movimiento - Stock Ahora"
	@echo "Comandos disponibles:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# === SETUP Y CONFIGURACIÓN ===
setup: ## Configurar proyecto inicial
	@echo "⚙️ Configurando proyecto..."
	@go mod tidy
	@cp .env.example .env || echo "⚠️ Crear archivo .env manualmente"
	@echo "✅ Proyecto configurado. Edita el archivo .env con tus credenciales"

init-db: ## Inicializar base de datos local
	@echo "📊 Inicializando base de datos..."
	@docker-compose up -d postgres
	@sleep 10
	@echo "✅ Base de datos lista"

# === DESARROLLO ===
build: ## Compilar la aplicación
	@echo "🔨 Compilando aplicación..."
	@CGO_ENABLED=1 GOOS=linux go build -o $(BINARY_NAME) .
	@echo "✅ Compilación completada: $(BINARY_NAME)"

run: ## Ejecutar la aplicación localmente
	@echo "🚀 Ejecutando aplicación..."
	@go run .

run-dev: ## Ejecutar con recarga automática (requiere air)
	@echo "🔄 Ejecutando con auto-reload..."
	@air

test: ## Ejecutar tests
	@echo "🧪 Ejecutando tests..."
	@go test -v ./...

test-coverage: ## Generar reporte de cobertura
	@echo "📊 Generando reporte de cobertura..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Reporte generado: coverage.html"

lint: ## Ejecutar linter
	@echo "🔍 Ejecutando linter..."
	@golangci-lint run
	@echo "✅ Linting completado"

format: ## Formatear código
	@echo "🎨 Formateando código..."
	@go fmt ./...
	@goimports -w .
	@echo "✅ Código formateado"

# === DOCKER ===
docker-build: ## Construir imagen Docker
	@echo "🐳 Construyendo imagen Docker..."
	@docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .
	@echo "✅ Imagen Docker creada: $(DOCKER_IMAGE):$(VERSION)"

docker-run: docker-build ## Ejecutar contenedor Docker
	@echo "🐳 Ejecutando contenedor..."
	@docker run -p 8000:8000 --env-file .env $(DOCKER_IMAGE):$(VERSION)

# === DOCKER COMPOSE ===
up: ## Levantar servicios de desarrollo
	@echo "🐳 Levantando servicios de desarrollo..."
	@docker-compose --profile dev up -d
	@echo "✅ Servicios levantados"
	@echo "📊 API: http://localhost:8000"
	@echo "🐰 RabbitMQ Management: http://localhost:15672"
	@echo "💾 PostgreSQL: localhost:5432"

up-prod: ## Levantar servicios de producción
	@echo "🏭 Levantando servicios de producción..."
	@docker-compose up -d
	@echo "✅ Servicios de producción levantados"

up-monitoring: ## Levantar con monitoreo completo
	@echo "📊 Levantando servicios con monitoreo..."
	@docker-compose --profile monitoring up -d
	@echo "✅ Servicios con monitoreo levantados"
	@echo "📊 Grafana: http://localhost:3000"
	@echo "📈 Prometheus: http://localhost:9090"

down: ## Bajar todos los servicios
	@echo "🛑 Bajando servicios..."
	@docker-compose --profile dev --profile monitoring down
	@echo "✅ Servicios detenidos"

logs: ## Ver logs de todos los servicios
	@docker-compose logs -f

logs-api: ## Ver logs solo del API
	@docker-compose logs -f api-movimiento

restart: ## Reiniciar servicios
	@echo "🔄 Reiniciando servicios..."
	@docker-compose restart
	@echo "✅ Servicios reiniciados"

# === UTILIDADES ===
clean: ## Limpiar archivos generados
	@echo "🧹 Limpiando archivos..."
	@go clean
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@docker system prune -f
	@echo "✅ Archivos limpiados"

clean-volumes: ## Limpiar volúmenes de Docker (CUIDADO: borra datos)
	@echo "⚠️ Esto borrará TODOS los datos de la base de datos"
	@read -p "¿Estás seguro? [y/N]: " confirm && [ "$$confirm" = "y" ]
	@docker-compose down -v
	@docker volume prune -f
	@echo "🗑️ Volúmenes eliminados"

reset: clean clean-volumes up ## Reset completo del entorno

# === TESTING Y DESARROLLO ===
test-consumer: ## Probar consumer con mensaje de prueba
	@echo "🧪 Enviando mensaje de prueba..."
	@go run scripts/test_publisher.go

test-endpoints: ## Probar todos los endpoints
	@echo "🧪 Probando endpoints..."
	@curl -s http://localhost:8000/api/v1/health | jq .
	@curl -s http://localhost:8000/api/v1/metrics | jq .
	@echo "✅ Endpoints funcionando"

benchmark: ## Ejecutar benchmarks
	@echo "⚡ Ejecutando benchmarks..."
	@go test -bench=. -benchmem ./...

# === SEGURIDAD ===
generate-key: ## Generar clave de encriptación
	@echo "🔐 Generando clave de encriptación..."
	@echo "Clave AES-256 (base64):"
	@openssl rand -base64 32
	@echo "✅ Copiar esta clave al AWS Secrets Manager"

check-security: ## Verificar seguridad del código
	@echo "🔒 Verificando seguridad..."
	@gosec ./...
	@echo "✅ Verificación de seguridad completada"

# === DEPLOYMENT ===
deploy-staging: ## Desplegar a staging
	@echo "🚀 Desplegando a staging..."
	@docker-compose -f docker-compose.staging.yml up -d
	@echo "✅ Desplegado a staging"

deploy-prod: ## Desplegar a producción
	@echo "🏭 Desplegando a producción..."
	@docker-compose -f docker-compose.prod.yml up -d
	@echo "✅ Desplegado a producción"

# === MONITOREO ===
health: ## Verificar salud de todos los servicios
	@echo "🏥 Verificando salud de servicios..."
	@curl -s http://localhost:8000/api/v1/health | jq .
	@echo "✅ Verificación completada"

metrics: ## Mostrar métricas actuales
	@echo "📊 Obteniendo métricas..."
	@curl -s http://localhost:8000/api/v1/metrics | jq .

# === BASE DE DATOS ===
db-migrate: ## Ejecutar migraciones
	@echo "📊 Ejecutando migraciones..."
	@echo "⚠️ Migraciones automáticas al iniciar el servicio"

db-backup: ## Backup de base de datos
	@echo "💾 Creando backup..."
	@docker-compose exec postgres pg_dump -U postgres movimientos_db > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup creado"

db-restore: ## Restaurar base de datos (requiere archivo backup.sql)
	@echo "📥 Restaurando base de datos..."
	@docker-compose exec -T postgres psql -U postgres movimientos_db < backup.sql
	@echo "✅ Base de datos restaurada"

# === INFORMACIÓN ===
status: ## Mostrar estado de servicios
	@echo "📋 Estado de servicios:"
	@docker-compose ps

info: ## Mostrar información del proyecto
	@echo "📋 Información del proyecto:"
	@echo "Nombre: $(BINARY_NAME)"
	@echo "Versión: $(VERSION)"
	@echo "Imagen Docker: $(DOCKER_IMAGE)"
	@go version

# === DESARROLLO AVANZADO ===
dev-setup: ## Configuración inicial para desarrollo
	@echo "⚙️ Configurando entorno de desarrollo..."
	@cp .env.example .env || echo "Crear archivo .env manualmente"
	@make setup
	@make docker-compose-up
	@echo "✅ Entorno de desarrollo listo"

dev-reset: ## Resetear entorno de desarrollo
	@echo "🔄 Reseteando entorno de desarrollo..."
	@make docker-compose-down
	@docker volume prune -f
	@make docker-compose-up
	@echo "✅ Entorno reseteado"

# === COMANDOS DE MONITOREO ===
logs: ## Ver logs de la aplicación
	@echo "📋 Mostrando logs..."
	@tail -f logs/api-movimiento.log || echo "No hay archivo de logs disponible"

# === COMANDOS DE TESTING ===
test-unit: ## Ejecutar tests unitarios
	@echo "🧪 Ejecutando tests unitarios..."
	@go test -v -short ./...

test-integration: ## Ejecutar tests de integración
	@echo "🧪 Ejecutando tests de integración..."
	@go test -v -run Integration ./...

# === COMANDOS DE DOCUMENTACIÓN ===
docs: ## Generar documentación
	@echo "📚 Generando documentación..."
	@godoc -http=:6060 &
	@echo "✅ Documentación disponible en http://localhost:6060"

# === VALIDACIÓN ===
validate: lint test ## Validar código completo
	@echo "✅ Validación completada"

# === BUILD PARA PRODUCCIÓN ===
build-prod: ## Build optimizado para producción
	@echo "🏭 Compilando para producción..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-a -installsuffix cgo \
		-o $(BINARY_NAME) .
	@echo "✅ Build de producción completado"