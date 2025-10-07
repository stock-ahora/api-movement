package consumer

import (
	"encoding/json"
	"log"

	"api-movement/models"
	"api-movement/services"

	"github.com/streadway/amqp"
)

// ConsumeMovements conecta al topic "movement.generated" y procesa mensajes
func ConsumeMovements(ch *amqp.Channel, queueName string, svc *services.MovimientoService) error {
	msgs, err := ch.Consume(
		queueName,
		"",
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	for msg := range msgs {
		var event models.MovementsEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("❌ Error parseando mensaje: %v", err)
			continue
		}

		if err := svc.ProcessMovement(event); err != nil {
			log.Printf("❌ Error procesando movimiento: %v", err)
		}
	}

	return nil
}
