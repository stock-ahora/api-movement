package handlers

import (
	"api-movimiento/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MovimientoHandler maneja los endpoints HTTP para movimientos
type MovimientoHandler struct {
	service *services.MovimientoService
}

// NewMovimientoHandler crea una nueva instancia del handler
func NewMovimientoHandler(service *services.MovimientoService) *MovimientoHandler {
	return &MovimientoHandler{
		service: service,
	}
}

// ListarMovimientos obtiene movimientos con filtros
// GET /api/v1/movimientos?product_id=...&client_account_id=...&limit=100
func (h *MovimientoHandler) ListarMovimientos(c *gin.Context) {
	// Obtener parámetros de query
	filters := make(map[string]interface{})

	if productID := c.Query("product_id"); productID != "" {
		productUUID, err := uuid.Parse(productID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
			return
		}
		filters["product_id"] = productUUID
	}

	if skuID := c.Query("sku_id"); skuID != "" {
		skuUUID, err := uuid.Parse(skuID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID de SKU inválido"})
			return
		}
		filters["sku_id"] = skuUUID
	}

	if requestID := c.Query("request_id"); requestID != "" {
		requestUUID, err := uuid.Parse(requestID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID de request inválido"})
			return
		}
		filters["request_id"] = requestUUID
	}

	if clientAccount := c.Query("client_account_id"); clientAccount != "" {
		clientUUID, err := uuid.Parse(clientAccount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID de cliente inválido"})
			return
		}
		filters["client_account_id"] = clientUUID
	}

	if tipoMovimiento := c.Query("tipo_movimiento"); tipoMovimiento != "" {
		if tipoMovimiento != "entrada" && tipoMovimiento != "salida" && tipoMovimiento != "ajuste" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo de movimiento inválido"})
			return
		}
		filters["tipo_movimiento"] = tipoMovimiento
	}

	if origen := c.Query("origen"); origen != "" {
		filters["origen"] = origen
	}

	// Límite de resultados
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	// Obtener movimientos
	movimientos, err := h.service.GetMovimientos(filters, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo movimientos: " + err.Error()})
		return
	}

	// Respuesta
	c.JSON(http.StatusOK, gin.H{
		"data":    movimientos,
		"count":   len(movimientos),
		"limit":   limit,
		"filters": filters,
	})
}

// ObtenerTrazabilidad obtiene la trazabilidad completa de un producto
// GET /api/v1/movimientos/producto/{product_id}/trazabilidad?include_requests=true
func (h *MovimientoHandler) ObtenerTrazabilidad(c *gin.Context) {
	productIDStr := c.Param("product_id")
	includeRequests := c.DefaultQuery("include_requests", "false") == "true"

	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de producto inválido"})
		return
	}

	trazabilidad, err := h.service.GetTrazabilidadProducto(productID, includeRequests)
	if err != nil {
		if err.Error() == "no se encontraron movimientos para el producto" {
			c.JSON(http.StatusNotFound, gin.H{"error": "No se encontraron movimientos para este producto"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo trazabilidad: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, trazabilidad)
}

// ObtenerMovimientosSku obtiene movimientos por SKU específico
// GET /api/v1/movimientos/sku/{sku_id}?limit=50
func (h *MovimientoHandler) ObtenerMovimientosSku(c *gin.Context) {
	skuIDStr := c.Param("sku_id")
	limitStr := c.DefaultQuery("limit", "50")

	skuID, err := uuid.Parse(skuIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de SKU inválido"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 500 {
		limit = 50
	}

	movimientos, err := h.service.GetMovimientosBySku(skuID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo movimientos por SKU: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sku_id":            skuID,
		"total_movimientos": len(movimientos),
		"limit":             limit,
		"movimientos":       movimientos,
	})
}

// ObtenerMovimientosRequest obtiene movimientos por Request ID
// GET /api/v1/movimientos/request/{request_id}?limit=100
func (h *MovimientoHandler) ObtenerMovimientosRequest(c *gin.Context) {
	requestIDStr := c.Param("request_id")
	limitStr := c.DefaultQuery("limit", "100")

	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID de request inválido"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	movimientos, err := h.service.GetMovimientosByRequest(requestID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo movimientos por request: " + err.Error()})
		return
	}

	// Agrupar por producto para estadísticas
	productStats := make(map[uuid.UUID]map[string]interface{})
	for _, mov := range movimientos {
		if _, exists := productStats[mov.ProductID]; !exists {
			productStats[mov.ProductID] = map[string]interface{}{
				"product_id":        mov.ProductID,
				"total_movimientos": 0,
				"total_cantidad":    int64(0),
				"tipos":             make(map[string]int),
			}
		}

		stats := productStats[mov.ProductID]
		stats["total_movimientos"] = stats["total_movimientos"].(int) + 1
		stats["total_cantidad"] = stats["total_cantidad"].(int64) + mov.Cantidad

		tipos := stats["tipos"].(map[string]int)
		tipos[mov.TipoMovimiento]++
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":             requestID,
		"total_movimientos":      len(movimientos),
		"productos_afectados":    len(productStats),
		"limit":                  limit,
		"movimientos":            movimientos,
		"estadisticas_productos": productStats,
	})
}
