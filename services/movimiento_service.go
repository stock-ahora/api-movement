// services/movimiento_service.go
package services

import (
	"api-movement/models"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// MovimientoService representa el servicio de movimientos
type MovimientoService struct {
	db *sql.DB
}

// NewMovimientoService ahora recibe la conexión a la DB en lugar de crear una nueva.
func NewMovimientoService(db *sql.DB) *MovimientoService {
	log.Println("✅ Servicio de movimientos listo")
	return &MovimientoService{db: db}
}

// ProcessMovement inserta un MovementsEvent en la DB
func (s *MovimientoService) ProcessMovement(event models.MovementsEvent) error {
	var err error
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			log.Printf("🚨 Revirtiendo transacción debido a error: %v", err)
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for _, movementProduct := range event.ProductPerMovement {

		movementQuery := `INSERT INTO movement (id, count, product_id, request_id, movement_type_id, create_at) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`
		_, err = tx.Exec(movementQuery, movementProduct.Id, movementProduct.Count, movementProduct.ProductID, event.RequestId, movementProduct.MovementTypeId, movementProduct.CreatedAt)
		if err != nil {
			return nil
			log.Printf("error insertando en movement")
		}

		productQuery := `INSERT INTO request_per_product (id, product_id, movement_id, request_id) VALUES ($1, $2, $3, $4)`
		_, err = tx.Exec(productQuery, uuid.New(), movementProduct.ProductID, movementProduct.Id, event.RequestId)
		if err != nil {
			return fmt.Errorf("error insertando en request_per_product: %w", err)
		}
	}

	log.Printf("✅ Movimiento %s procesado correctamente con %d productos\n", event.Id, len(event.ProductPerMovement))
	return nil
}

// --- Nuevas funciones para la API ---

// MovementResponse es el struct que se devolverá en la API
type MovementResponse struct {
	ID        uuid.UUID       `json:"id"`
	RequestID uuid.UUID       `json:"request_id"`
	CreatedAt time.Time       `json:"created_at"`
	Products  []ProductDetail `json:"products,omitempty"` // omitempty para no mostrarlo si está vacío
}

type ProductDetail struct {
	ID           uuid.UUID `json:"id"` // ID de la tabla request_per_product
	ProductID    uuid.UUID `json:"product_id"`
	Count        int       `json:"count"`
	MovementType int       `json:"movement_type"`
	DateLimit    time.Time `json:"date_limit"`
}

// FindAllMovements consulta una lista de movimientos sin detalle de productos
func (s *MovimientoService) FindAllMovements() ([]MovementResponse, error) {
	rows, err := s.db.Query("SELECT id, request_id, created_at FROM movement ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movements []MovementResponse
	for rows.Next() {
		var m MovementResponse
		if err := rows.Scan(&m.ID, &m.RequestID, &m.CreatedAt); err != nil {
			return nil, err
		}
		movements = append(movements, m)
	}
	return movements, nil
}

// FindMovementByID encuentra un movimiento y todos sus productos asociados
func (s *MovimientoService) FindMovementByID(id uuid.UUID) (*MovementResponse, error) {
	query := `
		SELECT
			m.id, m.request_id, m.created_at,
			rpp.id, rpp.product_id, rpp.count, rpp.movement_type, rpp.date_limit
		FROM movement m
		LEFT JOIN request_per_product rpp ON m.id = rpp.movement_id
		WHERE m.id = $1
	`
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movement MovementResponse
	var found bool

	for rows.Next() {
		found = true
		var p ProductDetail
		var pID, pProductID sql.NullString
		var pCount, pMovementType sql.NullInt64
		var pDateLimit sql.NullTime

		if err := rows.Scan(
			&movement.ID, &movement.RequestID, &movement.CreatedAt,
			&pID, &pProductID, &pCount, &pMovementType, &pDateLimit,
		); err != nil {
			return nil, err
		}

		if pID.Valid {
			p.ID, _ = uuid.Parse(pID.String)
			p.ProductID, _ = uuid.Parse(pProductID.String)
			p.Count = int(pCount.Int64)
			p.MovementType = int(pMovementType.Int64)
			p.DateLimit = pDateLimit.Time
			movement.Products = append(movement.Products, p)
		}
	}

	if !found {
		return nil, fmt.Errorf("movimiento con ID %s no encontrado", id)
	}

	return &movement, nil
}
