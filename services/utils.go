package services

import (
	"context"
	"fmt"
	"time"
)

// CheckDBConnection verifica la conexión a la base de datos
func (s *MovimientoService) CheckDBConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.db.PingContext(ctx)
}

// CheckRabbitConnection verifica la conexión a RabbitMQ
func (s *MovimientoService) CheckRabbitConnection() error {
	if s.rabbitCh == nil {
		return fmt.Errorf("canal RabbitMQ no inicializado")
	}

	return nil
}
