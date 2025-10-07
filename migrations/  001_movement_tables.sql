-- Tabla principal de movimientos
CREATE TABLE IF NOT EXISTS movement (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Tabla de productos por movimiento
CREATE TABLE IF NOT EXISTS request_per_product (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    movement_id UUID NOT NULL REFERENCES movement(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    count INT NOT NULL,
    movement_type INT NOT NULL,
    date_limit TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índices para optimizar consultas frecuentes
CREATE INDEX IF NOT EXISTS idx_movement_request_id ON movement(request_id);
CREATE INDEX IF NOT EXISTS idx_request_per_product_movement_id ON request_per_product(movement_id);
CREATE INDEX IF NOT EXISTS idx_request_per_product_product_id ON request_per_product(product_id);

-- Función para generar UUIDs si no existe
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
