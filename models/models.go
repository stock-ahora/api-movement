package models

import (
	"time"
	"github.com/google/uuid"
)

// Product modelo compatible con tu API Stock
type Product struct {
	ID            uuid.UUID `json:"id"`
	ReferencialID uuid.UUID `json:"referencial_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Stock         int64     `json:"stock"`
	Status        string    `json:"status"`
	ClientAccount uuid.UUID `json:"client_account_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Sku           []Sku     `json:"sku,omitempty"`
}

// Sku modelo compatible con tu API Stock
type Sku struct {
	ID        uuid.UUID `json:"id"`
	NameSku   string    `json:"name_sku"`
	Status    bool      `json:"status"`
	ProductID uuid.UUID `json:"product_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Product   Product   `json:"product,omitempty"`
}

// RequestStatus estados de request
type RequestStatus string

const (
	RequestCreated         RequestStatus = "created"
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusApproved  RequestStatus = "approved"
	RequestStatusRejected  RequestStatus = "rejected"
	RequestStatusCancelled RequestStatus = "cancelled"
)

// Request modelo compatible con tu API Stock
type Request struct {
	ID              uuid.UUID     `json:"id"`
	ClientAccountID uuid.UUID     `json:"client_account_id"`
	Status          RequestStatus `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Documents       []Documents   `json:"documents,omitempty"`
}

// Documents modelo compatible con tu API Stock
type Documents struct {
	ID         uuid.UUID `json:"id"`
	S3Path     string    `json:"s3_path"`
	RequestID  uuid.UUID `json:"request_id"`
	TextractId string    `json:"textract_id"`
	BedrockId  string    `json:"bedrock_id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Movimiento modelo principal para trazabilidad
type Movimiento struct {
	ID               int         `json:"id" db:"id"`
	ProductID        uuid.UUID   `json:"product_id" db:"product_id"`
	SkuID            *uuid.UUID  `json:"sku_id,omitempty" db:"sku_id"`
	RequestID        *uuid.UUID  `json:"request_id,omitempty" db:"request_id"`
	TipoMovimiento   string      `json:"tipo_movimiento" db:"tipo_movimiento"`
	Cantidad         int64       `json:"cantidad" db:"cantidad"`
	CantidadAnterior int64       `json:"cantidad_anterior" db:"cantidad_anterior"`
	CantidadNueva    int64       `json:"cantidad_nueva" db:"cantidad_nueva"`
	FechaMovimiento  time.Time   `json:"fecha_movimiento" db:"fecha_movimiento"`
	UsuarioID        *string     `json:"usuario_id,omitempty" db:"usuario_id"`
	Motivo           *string     `json:"motivo,omitempty" db:"motivo"`
	ClientAccountID  uuid.UUID   `json:"client_account_id" db:"client_account_id"`
	DocumentID       *uuid.UUID  `json:"document_id,omitempty" db:"document_id"`
	Origen           string      `json:"origen" db:"origen"` // ocr, manual, api, etc
	DatosEncriptados string      `json:"-" db:"datos_encriptados"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}

// MovimientoMessage estructura del mensaje de RabbitMQ
type MovimientoMessage struct {
	ProductID       uuid.UUID  `json:"product_id"`
	SkuID           *uuid.UUID `json:"sku_id,omitempty"`
	RequestID       *uuid.UUID `json:"request_id,omitempty"`
	DocumentID      *uuid.UUID `json:"document_id,omitempty"`
	TipoMovimiento  string     `json:"tipo_movimiento"` // entrada, salida, ajuste
	Cantidad        int64      `json:"cantidad"`
	UsuarioID       *string    `json:"usuario_id,omitempty"`
	Motivo          *string    `json:"motivo,omitempty"`
	ClientAccountID uuid.UUID  `json:"client_account_id"`
	Origen          string     `json:"origen"` // ocr, manual, api, etc
	Timestamp       time.Time  `json:"timestamp"`
	// Metadatos adicionales
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// StockUpdateRequest para actualizar stock en API Stock
type StockUpdateRequest struct {
	Stock int64 `json:"stock"`
}

// TrazabilidadResponse respuesta de trazabilidad
type TrazabilidadResponse struct {
	ProductID            uuid.UUID             `json:"product_id"`
	ProductName          string                `json:"product_name"`
	TotalMovimientos     int                   `json:"total_movimientos"`
	StockActual          int64                 `json:"stock_actual"`
	Historial            []HistorialMovimiento `json:"historial"`
	RequestsRelacionados []RequestSummary      `json:"requests_relacionados,omitempty"`
	SkusAfectados        []SkuSummary          `json:"skus_afectados,omitempty"`
	Resumen              ResumenMovimientos    `json:"resumen"`
}

// HistorialMovimiento detalle de cada movimiento
type HistorialMovimiento struct {
	Fecha         time.Time  `json:"fecha"`
	Tipo          string     `json:"tipo"`
	Cantidad      int64      `json:"cantidad"`
	StockAnterior int64      `json:"stock_anterior"`
	StockNuevo    int64      `json:"stock_nuevo"`
	Motivo        *string    `json:"motivo,omitempty"`
	Usuario       *string    `json:"usuario,omitempty"`
	RequestID     *uuid.UUID `json:"request_id,omitempty"`
	DocumentID    *uuid.UUID `json:"document_id,omitempty"`
	SkuID         *uuid.UUID `json:"sku_id,omitempty"`
	SkuName       *string    `json:"sku_name,omitempty"`
	Origen        string     `json:"origen"`
}

// RequestSummary resumen de request relacionado
type RequestSummary struct {
	RequestID   uuid.UUID     `json:"request_id"`
	Status      RequestStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	Documentos  int           `json:"total_documentos"`
	Movimientos int           `json:"total_movimientos"`
}

// SkuSummary resumen de SKU afectado
type SkuSummary struct {
	SkuID       uuid.UUID `json:"sku_id"`
	SkuName     string    `json:"sku_name"`
	Status      bool      `json:"status"`
	Movimientos int       `json:"total_movimientos"`
}

// ResumenMovimientos resumen estadístico
type ResumenMovimientos struct {
	TotalEntradas int64 `json:"total_entradas"`
	TotalSalidas  int64 `json:"total_salidas"`
	TotalAjustes  int64 `json:"total_ajustes"`
	PorOrigen     map[string]int `json:"por_origen"`
}

// NotificationMessage mensaje de notificación
type NotificationMessage struct {
	Type        string                 `json:"type"`
	ProductID   uuid.UUID              `json:"product_id"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    string                 `json:"severity"` // info, warning, error
}

// MetricsData métricas del sistema
type MetricsData struct {
	Service              string                 `json:"service"`
	Version              string                 `json:"version"`
	TotalMovimientos     int                    `json:"total_movimientos"`
	MovimientosHoy       int                    `json:"movimientos_hoy"`
	MovimientosPorTipo   map[string]int         `json:"movimientos_por_tipo"`
	MovimientosPorOrigen map[string]int         `json:"movimientos_por_origen"`
	UltimosMovimientos   []HistorialMovimiento  `json:"ultimos_movimientos"`
	Timestamp            time.Time              `json:"timestamp"`
	Uptime               string                 `json:"uptime"`
}