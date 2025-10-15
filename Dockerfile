# Etapa 1: build
FROM golang:1.25-alpine AS builder

# Crear y establecer directorio de trabajo
WORKDIR /app

# Copiar go.mod y go.sum primero para aprovechar cache
COPY go.mod go.sum ./
RUN go mod download

# Copiar el resto del código
COPY . .

# Compilar binario
RUN go build -o server .

# Etapa 2: imagen final
FROM alpine:latest

# Crear directorio de trabajo
WORKDIR /app

# Copiar binario desde la etapa anterior
COPY --from=builder /app/server .
COPY .env .env

# Exponer el puerto 8081
EXPOSE 8081

# Comando para ejecutar
CMD ["./server"]