// consumer/rabbitmq_consumer.go
package consumer

import (
	"encoding/json"
	"log"

	"api-movement/models"
	"api-movement/services"

	"github.com/streadway/amqp"
)

// Start consume mensajes de RabbitMQ de forma robusta.
func Start(ch *amqp.Channel, queueName string, svc *services.MovimientoService) error {
	log.Printf("✅ Esperando mensajes en queue '%s'...", queueName)

	// Consume con Acknowledge manual
	msgs, err := ch.Consume(
		queueName,
		"",
		false, // auto-ack = false
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("📦 Mensaje recibido: %s", string(msg.Body))

			var event models.MovementsEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("🚨 Error parseando JSON del mensaje: %v. Mensaje descartado.", err)
				msg.Nack(false, false) // Descartar mensaje (no re-encolar)
				continue
			}

			if err := svc.ProcessMovement(event); err != nil {
				log.Printf("🚨 Error procesando movimiento: %v. Re-encolando mensaje.", err)
				msg.Nack(false, true) // Re-encolar mensaje para reintentar
			} else {
				log.Printf("✅ Mensaje para movimiento %s procesado y confirmado.", event.Id)
				msg.Ack(false) // Confirmar que el mensaje fue procesado
			}
		}
	}()

	<-forever
	return nil
}