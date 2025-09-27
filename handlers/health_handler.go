package handlers

import (
	"api-movimiento/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler maneja endpoints de salud y métricas
type HealthHandler struct {
	service *services.MovimientoService
}

// NewHealthHandler crea nueva instancia del health handler
func NewHealthHandler(service *services.MovimientoService) *HealthHandler {
	return &HealthHandler{
		service: service,
	}
}

// HealthCheck verifica el estado del servicio
// GET /api/v1/health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	// Verificar conexión a base de datos
	dbStatus := "healthy"
	if err := h.service.CheckDBConnection(); err != nil {
		dbStatus = "unhealthy: " + err.Error()
	}

	// Verificar conexión a RabbitMQ
	rabbitStatus := "healthy"
	if err := h.service.CheckRabbitConnection(); err != nil {
		rabbitStatus = "unhealthy: " + err.Error()
	}

	// Estado general
	overallStatus := "healthy"
	httpStatus := http.StatusOK

	if dbStatus != "healthy" || rabbitStatus != "healthy" {
		overallStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	response := gin.H{
		"status":    overallStatus,
		"timestamp": time.Now(),
		"service":   "API Movimientos",
		"version":   "1.0.0",
		"checks": gin.H{
			"database": dbStatus,
			"rabbitmq": rabbitStatus,
		},
		"uptime": time.Since(startTime).String(),
	}

	c.JSON(httpStatus, response)
}

// GetMetrics obtiene métricas del sistema
// GET /api/v1/metrics
func (h *HealthHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.service.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error obteniendo métricas: " + err.Error(),
		})
		return
	}

	// Agregar métricas adicionales del sistema
	metrics.Uptime = time.Since(startTime).String()

	c.JSON(http.StatusOK, metrics)
}

// Variable global para tiempo de inicio
var startTime = time.Now()