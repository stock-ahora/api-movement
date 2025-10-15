// consumer/rabbitmq_consumer.go
package consumer

import (
	"encoding/json"
	"fmt"
	"log"

	"api-movement/models"
	"api-movement/services"

	"github.com/streadway/amqp"
)

// Start consume mensajes de RabbitMQ de forma robusta.
func Start(ch *amqp.Channel, queueName string, svc *services.MovimientoService) error {
	log.Printf("✅ Esperando mensajes en queue '%s'...", queueName)

	err := ch.ExchangeDeclare(
		"events.topic", // nombre (debe ser el mismo que el productor)
		"topic",        // tipo
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)

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

	err = SetupListener(ch, queueName, []string{"movement.generated"}, 5)
	if err != nil {
		log.Printf("❌ Error configurando listener: %v", err)
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
				msg.Nack(false, false) // Re-encolar mensaje para reintentar
			} else {
				log.Printf("✅ Mensaje para movimiento %s procesado y confirmado.", event.Id)
				msg.Ack(false) // Confirmar que el mensaje fue procesado
			}
		}
	}()

	<-forever
	return nil
}

func SetupListener(channel *amqp.Channel, queueName string, routingKeys []string, workerCount int) error {
	q, err := channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error creando queue: %w", err)
	}

	for _, rk := range routingKeys {
		if err := channel.QueueBind(
			q.Name,
			rk,
			"events.topic",
			false,
			nil,
		); err != nil {
			return fmt.Errorf("error en binding de routing key %s: %w", rk, err)
		}
	}

	// QoS → mensajes por worker
	if err := channel.Qos(workerCount, 0, false); err != nil {
		return fmt.Errorf("error configurando QoS: %w", err)
	}

	return nil
}
