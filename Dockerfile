# Multi-stage build para API Movimiento
FROM golang:1.21-alpine AS builder

# Instalar dependencias del sistema
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY api-movimiento .

# Compilar la aplicación
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

# Etapa final - imagen mínima de producción
FROM alpine:latest

# Instalar certificados CA y timezone data
RUN apk --no-cache add ca-certificates tzdata curl

# Crear usuario no-root
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

WORKDIR /app

# Copiar el binario compilado
COPY --from=builder /app/main .

# Cambiar ownership
RUN chown -R appuser:appgroup /app

# Cambiar a usuario no-root
USER appuser

# Exponer puerto
EXPOSE 8000

# Variables de entorno por defecto
ENV GIN_MODE=release
ENV PORT=8000
ENV TZ=America/Santiago

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/api/v1/health || exit 1

# Comando para ejecutar la aplicación
CMD ["./main"]