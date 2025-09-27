package services

import (
	"api-movimiento/database"
	"api-movimiento/models"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// GetMovimientos obtiene movimientos con filtros
func (s *MovimientoService) GetMovimientos(filters map[string]interface{}, limit int) ([]models.Movimiento, error) {
	rows, err := database.GetMovimientosByFilters(s.db, filters, limit)
	if err != nil {
		return nil, fmt.Errorf("error ejecutando query: %w", err)
	}
	defer rows.Close()

	var movimientos []models.Movimiento
	for rows.Next() {
		var m models.Movimiento
		var productName, skuName sql.NullString

		err := rows.Scan(
			&m.ID, &m.ProductID, &m.SkuID, &m.RequestID, &m.TipoMovimiento,
			&m.Cantidad, &m.CantidadAnterior, &m.CantidadNueva, &m.FechaMovimiento,
			&m.UsuarioID, &m.Motivo, &m.ClientAccountID, &m.DocumentID, &m.Origen,
			&m.CreatedAt, &m.UpdatedAt, &productName, &skuName,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando movimiento: %w", err)
		}

		movimientos = append(movimientos, m)
	}

	return movimientos, nil
}

// GetTrazabilidadProducto obtiene la trazabilidad completa de un producto
func (s *MovimientoService) GetTrazabilidadProducto(productID uuid.UUID, includeRequests bool) (*models.TrazabilidadResponse, error) {
	// Query principal para historial
	query := `
		SELECT m.id, m.sku_id, m.request_id, m.tipo_movimiento, m.cantidad,
		       m.cantidad_anterior, m.cantidad_nueva, m.fecha_movimiento,
		       m.usuario_id, m.motivo, m.document_id, m.origen,
		       p.name as product_name, s.name_sku as sku_name
		FROM movimientos m
		LEFT JOIN product p ON m.product_id = p.id
		LEFT JOIN sku s ON m.sku_id = s.id
		WHERE m.product_id = $1
		ORDER BY m.fecha_movimiento ASC`

	rows, err := s.db.Query(query, productID)
	if err != nil {
		return nil, fmt.Errorf("error consultando trazabilidad: %w", err)
	}
	defer rows.Close()

	var historial []models.HistorialMovimiento
	var stockActual int64
	var productName string
	requestIDsMap := make(map[uuid.UUID]bool)
	skuIDsMap := make(map[uuid.UUID]string) // sku_id -> sku_name

	// Contadores para resumen
	var totalEntradas, totalSalidas, totalAjustes int64
	origenCount := make(map[string]int)

	for rows.Next() {
		var m models.Movimiento
		var pName, skuName sql.NullString

		err := rows.Scan(
			&m.ID, &m.SkuID, &m.RequestID, &m.TipoMovimiento, &m.Cantidad,
			&m.CantidadAnterior, &m.CantidadNueva, &m.FechaMovimiento,
			&m.UsuarioID, &m.Motivo, &m.DocumentID, &m.Origen,
			&pName, &skuName,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando trazabilidad: %w", err)
		}

		if pName.Valid {
			productName = pName.String
		}

		// Agregar RequestID al mapa si existe
		if m.RequestID != nil {
			requestIDsMap[*m.RequestID] = true
		}

		// Agregar SKU al mapa si existe
		if m.SkuID != nil && skuName.Valid {
			skuIDsMap[*m.SkuID] = skuName.String
		}

		// Crear entrada del historial
		var skuNamePtr *string
		if skuName.Valid {
			skuNamePtr = &skuName.String
		}

		historial = append(historial, models.HistorialMovimiento{
			Fecha:         m.FechaMovimiento,
			Tipo:          m.TipoMovimiento,
			Cantidad:      m.Cantidad,
			StockAnterior: m.CantidadAnterior,
			StockNuevo:    m.CantidadNueva,
			Motivo:        m.Motivo,
			Usuario:       m.UsuarioID,
			RequestID:     m.RequestID,
			DocumentID:    m.DocumentID,
			SkuID:         m.SkuID,
			SkuName:       skuNamePtr,
			Origen:        m.Origen,
		})

		stockActual = m.CantidadNueva

		// Actualizar contadores
		switch m.TipoMovimiento {
		case "entrada":
			totalEntradas += m.Cantidad
		case "salida":
			totalSalidas += m.Cantidad
		case "ajuste":
			totalAjustes += m.Cantidad
		}
		origenCount[m.Origen]++
	}

	if len(historial) == 0 {
		return nil, fmt.Errorf("no se encontraron movimientos para el producto")
	}

	// Crear respuesta base
	response := &models.TrazabilidadResponse{
		ProductID:        productID,
		ProductName:      productName,
		TotalMovimientos: len(historial),
		StockActual:      stockActual,
		Historial:        historial,
		Resumen: models.ResumenMovimientos{
			TotalEntradas: totalEntradas,
			TotalSalidas:  totalSalidas,
			TotalAjustes:  totalAjustes,
			PorOrigen:     origenCount,
		},
	}

	// Agregar información de requests si se solicita
	if includeRequests && len(requestIDsMap) > 0 {
		requestsSummary, err := s.getRequestsSummary(requestIDsMap)
		if err == nil {
			response.RequestsRelacionados = requestsSummary
		}
	}

	// Agregar información de SKUs
	if len(skuIDsMap) > 0 {
		skusSummary, err := s.getSkusSummary(productID, skuIDsMap)
		if err == nil {
			response.SkusAfectados = skusSummary
		}
	}

	return response, nil
}

// GetMovimientosBySku obtiene movimientos por SKU específico
func (s *MovimientoService) GetMovimientosBySku(skuID uuid.UUID, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT m.id, m.product_id, m.sku_id, m.request_id, m.tipo_movimiento,
		       m.cantidad, m.cantidad_anterior, m.cantidad_nueva, m.fecha_movimiento,
		       m.usuario_id, m.motivo, m.client_account_id, m.document_id, m.origen,
		       s.name_sku, p.name as product_name
		FROM movimientos m
		JOIN sku s ON m.sku_id = s.id
		JOIN product p ON m.product_id = p.id
		WHERE m.sku_id = $1
		ORDER BY m.fecha_movimiento DESC
		LIMIT $2`

	rows, err := s.db.Query(query, skuID, limit)
	if err != nil {
		return nil, fmt.Errorf("error consultando movimientos por SKU: %w", err)
	}
	defer rows.Close()

	var movimientos []map[string]interface{}
	for rows.Next() {
		var m models.Movimiento
		var skuName, productName string

		err := rows.Scan(
			&m.ID, &m.ProductID, &m.SkuID, &m.RequestID, &m.TipoMovimiento,
			&m.Cantidad, &m.CantidadAnterior, &m.CantidadNueva, &m.FechaMovimiento,
			&m.UsuarioID, &m.Motivo, &m.ClientAccountID, &m.DocumentID, &m.Origen,
			&skuName, &productName,
		)
		if err != nil {
			return nil, fmt.Errorf("error escaneando movimiento: %w", err)
		}

		movimiento := map[string]interface{}{
			"id":                m.ID,
			"product_id":        m.ProductID,
			"product_name":      productName,
			"sku_id":           m.SkuID,
			"sku_name":         skuName,
			"request_id":        m.RequestID,
			"tipo_movimiento":   m.TipoMovimiento,
			"cantidad":          m.Cantidad,
			"cantidad_anterior": m.CantidadAnterior,
			"cantidad_nueva":    m.CantidadNueva,
			"fecha_movimiento":  m.FechaMovimiento,
			"usuario_id":        m.UsuarioID,
			"motivo":           m.Motivo,
			"client_account_id": m.ClientAccountID,
			"document_id":       m.DocumentID,
			"origen":           m.Origen,
		}

		movimientos = append(movimientos, movimiento)
	}

	return movimientos, nil
}

// GetMovimientosByRequest obtiene movimientos por Request ID
func (s *MovimientoService) GetMovimientosByRequest(requestID uuid.UUID, limit int) ([]models.Movimiento, error) {
	filters := map[string]interface{}{
		"request_id": requestID,
	}

	return s.GetMovimientos(filters, limit)
}

// GetMetrics obtiene métricas del sistema
func (s *MovimientoService) GetMetrics() (*models.MetricsData, error) {
	metrics := &models.MetricsData{
		Service:              "API Movimientos",
		Version:              "1.0.0",
		MovimientosPorTipo:   make(map[string]int),
		MovimientosPorOrigen: make(map[string]int),
		Timestamp:            time.Now(),
	}

	// Total movimientos
	err := s.db.QueryRow("SELECT COUNT(*) FROM movimientos").Scan(&metrics.TotalMovimientos)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo total movimientos: %w", err)
	}

	// Movimientos hoy
	err = s.db.QueryRow("SELECT COUNT(*) FROM movimientos WHERE DATE(created_at) = CURRENT_DATE").Scan(&metrics.MovimientosHoy)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo movimientos de hoy: %w", err)
	}

	// Movimientos por tipo
	rows, err := s.db.Query("SELECT tipo_movimiento, COUNT(*) FROM movimientos GROUP BY tipo_movimiento")
	if err != nil {
		return nil, fmt.Errorf("error obteniendo movimientos por tipo: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tipo string
		var count int
		if err := rows.Scan(&tipo, &count); err != nil {
			continue
		}
		metrics.MovimientosPorTipo[tipo] = count
	}

	// Movimientos por origen
	rows, err = s.db.Query("SELECT origen, COUNT(*) FROM movimientos GROUP BY origen")
	if err != nil {
		return nil, fmt.Errorf("error obteniendo movimientos por origen: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var origen string
		var count int
		if err := rows.Scan(&origen, &count); err != nil {
			continue
		}
		metrics.MovimientosPorOrigen[origen] = count
	}

	// Últimos movimientos
	rows, err = s.db.Query(`
		SELECT m.fecha_movimiento, m.tipo_movimiento, m.cantidad, m.cantidad_anterior,
		       m.cantidad_nueva, m.motivo, m.usuario_id, m.request_id, m.document_id,
		       m.sku_id, m.origen, p.name as product_name
		FROM movimientos m
		LEFT JOIN product p ON m.product_id = p.id
		ORDER BY m.fecha_movimiento DESC
		LIMIT 5`)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo últimos movimientos: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h models.HistorialMovimiento
		var productName sql.NullString

		err := rows.Scan(
			&h.Fecha, &h.Tipo, &h.Cantidad, &h.StockAnterior,
			&h.StockNuevo, &h.Motivo, &h.Usuario, &h.RequestID, &h.DocumentID,
			&h.SkuID, &h.Origen, &productName,
		)
		if err != nil {
			continue
		}

		metrics.UltimosMovimientos = append(metrics.UltimosMovimientos, h)
	}

	return metrics, nil
}

// getRequestsSummary obtiene resumen de requests relacionados
func (s *MovimientoService) getRequestsSummary(requestIDsMap map[uuid.UUID]bool) ([]models.RequestSummary, error) {
	if len(requestIDsMap) == 0 {
		return nil, nil
	}

	requestIDs := make([]uuid.UUID, 0, len(requestIDsMap))
	for requestID := range requestIDsMap {
		requestIDs = append(requestIDs, requestID)
	}

	query := `
		SELECT r.id, r.status, r.create_at,
		       COUNT(d.id) as total_documentos,
		       COUNT(m.id) as total_movimientos
		FROM request r
		LEFT JOIN documents d ON r.id = d.request_id
		LEFT JOIN movimientos m ON r.id = m.request_id
		WHERE r.id = ANY($1)
		GROUP BY r.id, r.status, r.create_at
		ORDER BY r.create_at DESC`

	rows, err := s.db.Query(query, pq.Array(requestIDs))
	if err != nil {
		return nil, fmt.Errorf("error obteniendo resumen de requests: %w", err)
	}
	defer rows.Close()

	var requestsSummary []models.RequestSummary
	for rows.Next() {
		var rs models.RequestSummary
		err := rows.Scan(&rs.RequestID, &rs.Status, &rs.CreatedAt, &rs.Documentos, &rs.Movimientos)
		if err != nil {
			continue
		}
		requestsSummary = append(requestsSummary, rs)
	}

	return requestsSummary, nil
}

// getSkusSummary obtiene resumen de SKUs afectados
func (s *MovimientoService) getSkusSummary(productID uuid.UUID, skuIDsMap map[uuid.UUID]string) ([]models.SkuSummary, error) {
	if len(skuIDsMap) == 0 {
		return nil, nil
	}

	skuIDs := make([]uuid.UUID, 0, len(skuIDsMap))
	for skuID := range skuIDsMap {
		skuIDs = append(skuIDs, skuID)
	}

	query := `
		SELECT s.id, s.name_sku, s.status,
		       COUNT(m.id) as total_movimientos
		FROM sku s
		LEFT JOIN movimientos m ON s.id = m.sku_id AND m.product_id = $1
		WHERE s.id = ANY($2)
		GROUP BY s.id, s.name_sku, s.status
		ORDER BY s.name_sku`

	rows, err := s.db.Query(query, productID, pq.Array(skuIDs))
	if err != nil {
		return nil, fmt.Errorf("error obteniendo resumen de SKUs: %w", err)
	}
	defer rows.Close()

	var skusSummary []models.SkuSummary
	for rows.Next() {
		var ss models.SkuSummary
		err := rows.Scan(&ss.SkuID, &ss.SkuName, &ss.Status, &ss.Movimientos)
		if err != nil {
			continue
		}
		skusSummary = append(skusSummary, ss)
	}

	return skusSummary, nil
}