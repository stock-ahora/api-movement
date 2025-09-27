package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Connect conecta a la base de datos PostgreSQL
func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error abriendo conexión: %w", err)
	}

	// Verificar conexión
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error verificando conexión: %w", err)
	}

	// Configurar pool de conexiones
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("✅ Conectado a PostgreSQL")
	return db, nil
}

// InitTables inicializa las tablas necesarias
func InitTables(db *sql.DB) error {
	createTableQuery := `
	-- Extensión para UUID si no existe
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	-- Tabla principal de movimientos
	CREATE TABLE IF NOT EXISTS movimientos (
		id SERIAL PRIMARY KEY,
		product_id UUID NOT NULL,
		sku_id UUID,
		request_id UUID,
		tipo_movimiento VARCHAR(50) NOT NULL CHECK (tipo_movimiento IN ('entrada', 'salida', 'ajuste')),
		cantidad BIGINT NOT NULL CHECK (cantidad > 0),
		cantidad_anterior BIGINT NOT NULL CHECK (cantidad_anterior >= 0),
		cantidad_nueva BIGINT NOT NULL CHECK (cantidad_nueva >= 0),
		fecha_movimiento TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		usuario_id VARCHAR(255),
		motivo TEXT,
		client_account_id UUID NOT NULL,
		document_id UUID,
		origen VARCHAR(50) DEFAULT 'api' CHECK (origen IN ('api', 'ocr', 'manual', 'sistema')),
		datos_encriptados TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Índices para optimizar consultas
	CREATE INDEX IF NOT EXISTS idx_movimientos_product_id ON movimientos(product_id);
	CREATE INDEX IF NOT EXISTS idx_movimientos_sku_id ON movimientos(sku_id);
	CREATE INDEX IF NOT EXISTS idx_movimientos_request_id ON movimientos(request_id);
	CREATE INDEX IF NOT EXISTS idx_movimientos_client_account ON movimientos(client_account_id);
	CREATE INDEX IF NOT EXISTS idx_movimientos_fecha ON movimientos(fecha_movimiento);
	CREATE INDEX IF NOT EXISTS idx_movimientos_tipo ON movimientos(tipo_movimiento);
	CREATE INDEX IF NOT EXISTS idx_movimientos_origen ON movimientos(origen);
	CREATE INDEX IF NOT EXISTS idx_movimientos_document_id ON movimientos(document_id);

	-- Índices compuestos para consultas frecuentes
	CREATE INDEX IF NOT EXISTS idx_movimientos_product_fecha ON movimientos(product_id, fecha_movimiento DESC);
	CREATE INDEX IF NOT EXISTS idx_movimientos_client_fecha ON movimientos(client_account_id, fecha_movimiento DESC);

	-- Función para actualizar updated_at automáticamente
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	-- Trigger para actualizar updated_at
	DROP TRIGGER IF EXISTS update_movimientos_updated_at ON movimientos;
	CREATE TRIGGER update_movimientos_updated_at
		BEFORE UPDATE ON movimientos
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();

	-- Vista para consultas frecuentes con información enriquecida
	CREATE OR REPLACE VIEW movimientos_enriquecidos AS
	SELECT
		m.*,
		p.name as product_name,
		p.description as product_description,
		s.name_sku as sku_name,
		s.status as sku_status
	FROM movimientos m
	LEFT JOIN product p ON m.product_id = p.id
	LEFT JOIN sku s ON m.sku_id = s.id;

	-- Función para obtener stock actual por producto
	CREATE OR REPLACE FUNCTION get_stock_actual(p_product_id UUID)
	RETURNS BIGINT AS $$
	DECLARE
		stock_actual BIGINT;
	BEGIN
		SELECT cantidad_nueva INTO stock_actual
		FROM movimientos
		WHERE product_id = p_product_id
		ORDER BY fecha_movimiento DESC, id DESC
		LIMIT 1;

		RETURN COALESCE(stock_actual, 0);
	END;
	$$ LANGUAGE plpgsql;

	-- Función para obtener resumen de movimientos por producto
	CREATE OR REPLACE FUNCTION get_resumen_movimientos(p_product_id UUID)
	RETURNS TABLE(
		total_entradas BIGINT,
		total_salidas BIGINT,
		total_ajustes BIGINT,
		total_movimientos BIGINT
	) AS $$
	BEGIN
		RETURN QUERY
		SELECT
			COALESCE(SUM(CASE WHEN tipo_movimiento = 'entrada' THEN cantidad END), 0) as total_entradas,
			COALESCE(SUM(CASE WHEN tipo_movimiento = 'salida' THEN cantidad END), 0) as total_salidas,
			COALESCE(SUM(CASE WHEN tipo_movimiento = 'ajuste' THEN cantidad END), 0) as total_ajustes,
			COUNT(*)::BIGINT as total_movimientos
		FROM movimientos
		WHERE product_id = p_product_id;
	END;
	$$ LANGUAGE plpgsql;
	`

	_, err := db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("error creando tablas: %w", err)
	}

	log.Println("✅ Tablas de base de datos inicializadas")
	return nil
}

// GetMovimientosByFilters obtiene movimientos con filtros
func GetMovimientosByFilters(db *sql.DB, filters map[string]interface{}, limit int) (*sql.Rows, error) {
	query := `
		SELECT m.id, m.product_id, m.sku_id, m.request_id, m.tipo_movimiento,
		       m.cantidad, m.cantidad_anterior, m.cantidad_nueva, m.fecha_movimiento,
		       m.usuario_id, m.motivo, m.client_account_id, m.document_id, m.origen,
		       m.created_at, m.updated_at,
		       p.name as product_name, s.name_sku as sku_name
		FROM movimientos m
		LEFT JOIN product p ON m.product_id = p.id
		LEFT JOIN sku s ON m.sku_id = s.id
		WHERE 1=1`

	var args []interface{}
	argIndex := 1

	// Aplicar filtros dinámicamente
	if productID, ok := filters["product_id"]; ok {
		query += fmt.Sprintf(" AND m.product_id = $%d", argIndex)
		args = append(args, productID)
		argIndex++
	}

	if skuID, ok := filters["sku_id"]; ok {
		query += fmt.Sprintf(" AND m.sku_id = $%d", argIndex)
		args = append(args, skuID)
		argIndex++
	}

	if requestID, ok := filters["request_id"]; ok {
		query += fmt.Sprintf(" AND m.request_id = $%d", argIndex)
		args = append(args, requestID)
		argIndex++
	}

	if clientAccount, ok := filters["client_account_id"]; ok {
		query += fmt.Sprintf(" AND m.client_account_id = $%d", argIndex)
		args = append(args, clientAccount)
		argIndex++
	}

	if tipoMovimiento, ok := filters["tipo_movimiento"]; ok {
		query += fmt.Sprintf(" AND m.tipo_movimiento = $%d", argIndex)
		args = append(args, tipoMovimiento)
		argIndex++
	}

	if origen, ok := filters["origen"]; ok {
		query += fmt.Sprintf(" AND m.origen = $%d", argIndex)
		args = append(args, origen)
		argIndex++
	}

	// Ordenar y limitar
	query += fmt.Sprintf(" ORDER BY m.fecha_movimiento DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	return db.Query(query, args...)
}