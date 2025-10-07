package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"api-movement/models"
	"api-movement/database"

)

// MovimientoService representa el servicio de movimientos
type MovimientoService struct {
	db *sql.DB
}

// NewMovimientoService crea un nuevo servicio de movimientos y conecta a la DB
func NewMovimientoService(user, pass, host, port, dbname string) (*MovimientoService, error) {
	db, err := database.Connect(user, pass, host, port, dbname)
	if err != nil {
		return nil, fmt.Errorf("error conectando a la DB: %w", err)
	}

	log.Println("✅ Servicio de movimientos listo")
	return &MovimientoService{db: db}, nil
}

// ProcessMovement inserta un MovementsEvent en la DB
func (s *MovimientoService) ProcessMovement(event models.MovementsEvent) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("error iniciando transacción: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Insertar movimiento general
	movementQuery := `
		INSERT INTO movement (id, request_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO NOTHING
	`
	_, err = tx.Exec(movementQuery, event.Id, event.RequestId, time.Now())
	if err != nil {
		return fmt.Errorf("error insertando en movement: %w", err)
	}

	// Insertar cada producto relacionado
	productQuery := `
		INSERT INTO request_per_product
			(id, movement_id, product_id, count, movement_type, date_limit, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`

	for _, p := range event.ProductPerMovement {
		_, err := tx.Exec(
			productQuery,
			p.MovementId,
			event.Id,
			p.ProductID,
			p.Count,
			p.MovementTypeId,
			p.DateLimit,
			time.Now(),
		)
		if err != nil {
			return fmt.Errorf("error insertando en request_per_product: %w", err)
		}
	}

	log.Printf("✅ Movimiento %s procesado correctamente con %d productos\n", event.Id, len(event.ProductPerMovement))
	return nil
}
