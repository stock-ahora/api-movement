package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

// MovimientoMessage estructura del mensaje de prueba
type MovimientoMessage struct {
	ProductID       uuid.UUID              `json:"product_id"`
	SkuID           *uuid.UUID             `json:"sku_id,omitempty"`
	RequestID       *uuid.UUID             `json:"request_id,omitempty"`
	DocumentID      *uuid.UUID             `json:"document_id,omitempty"`
	TipoMovimiento  string                 `json:"tipo_movimiento"`
	Cantidad        int64                  `json:"cantidad"`
	UsuarioID       *string                `json:"usuario_id,omitempty"`
	Motivo          *string                `json:"motivo,omitempty"`
	ClientAccountID uuid.UUID              `json:"client_account_id"`
	Origen          string                 `json:"origen"`
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func main() {
	// Conectar a RabbitMQ local (development)
	conn, err := amqp.Dial("amqp://admin:admin123@localhost:5672/")
	if err != nil {
		log.Fatalf("Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Error abriendo canal: %v", err)
	}
	defer ch.Close()

	// Crear mensaje de prueba
	productID := uuid.New()
	skuID := uuid.New()
	requestID := uuid.New()
	documentID := uuid.New()
	clientAccountID := uuid.New()
	usuarioID := "test-user-123"
	motivo := "Movimiento de prueba desde script"

	mensaje := MovimientoMessage{
		ProductID:       productID,
		SkuID:           &skuID,
		RequestID:       &requestID,
		DocumentID:      &documentID,
		TipoMovimiento:  "entrada",
		Cantidad:        25,
		UsuarioID:       &usuarioID,
		Motivo:          &motivo,
		ClientAccountID: clientAccountID,
		Origen:          "script_test",
		Timestamp:       time.Now(),
		Metadata: map[string]interface{}{
			"test":           true,
			"script_version": "1.0",
			"ambiente":       "development",
		},
	}

	// Serializar mensaje
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Fatalf("Error serializando mensaje: %v", err)
	}

	// Publicar mensaje
	err = ch.Publish(
		"",                   // exchange
		"movimientos_queue",  // routing key
		false,                // mandatory
		false,                // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		log.Fatalf("Error publicando mensaje: %v", err)
	}

	log.Println("✅ Mensaje de prueba enviado exitosamente")
	log.Printf("📦 ProductID: %s", productID)
	log.Printf("🏷️ SkuID: %s", skuID)
	log.Printf("📋 RequestID: %s", requestID)
	log.Printf("📄 DocumentID: %s", documentID)
	log.Printf("👤 ClientAccountID: %s", clientAccountID)
	log.Printf("📊 Tipo: %s, Cantidad: %d", mensaje.TipoMovimiento, mensaje.Cantidad)

	// Crear algunos mensajes adicionales para pruebas
	testCases := []struct {
		tipo     string
		cantidad int64
		origen   string
		motivo   string
	}{
		{"salida", 10, "ocr", "Salida detectada por OCR"},
		{"ajuste", 100, "manual", "Ajuste de inventario manual"},
		{"entrada", 50, "api", "Entrada via API externa"},
		{"salida", 5, "ocr", "Venta procesada por OCR"},
		{"entrada", 75, "manual", "Recepción de mercancía"},
	}

	for i, tc := range testCases {
		time.Sleep(1 * time.Second) // Esperar entre mensajes

		newProductID := uuid.New()
		newClientID := uuid.New()
		newSkuID := uuid.New()
		newRequestID := uuid.New()
		motivo := tc.motivo

		msg := MovimientoMessage{
			ProductID:       newProductID,
			SkuID:           &newSkuID,
			RequestID:       &newRequestID,
			TipoMovimiento:  tc.tipo,
			Cantidad:        tc.cantidad,
			UsuarioID:       &usuarioID,
			Motivo:          &motivo,
			ClientAccountID: newClientID,
			Origen:          tc.origen,
			Timestamp:       time.Now(),
			Metadata: map[string]interface{}{
				"test_case": i + 1,
				"batch":     "test_batch_001",
				"automated": true,
			},
		}

		body, _ := json.Marshal(msg)
		err = ch.Publish("", "movimientos_queue", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

		if err != nil {
			log.Printf("❌ Error enviando mensaje %d: %v", i+1, err)
		} else {
			log.Printf("✅ Mensaje %d enviado: %s - %d unidades (Producto: %s)",
				i+1, tc.tipo, tc.cantidad, newProductID.String()[:8])
		}
	}

	// Mensaje de prueba con error para testing
	log.Println("📨 Enviando mensaje con error para probar manejo de errores...")

	mensajeError := MovimientoMessage{
		ProductID:       uuid.Nil, // UUID inválido para probar validación
		TipoMovimiento:  "tipo_invalido",
		Cantidad:        -5, // Cantidad negativa
		ClientAccountID: uuid.New(),
		Origen:          "test_error",
		Timestamp:       time.Now(),
		Metadata: map[string]interface{}{
			"test_error": true,
			"purpose":    "testing error handling",
		},
	}

	bodyError, _ := json.Marshal(mensajeError)
	err = ch.Publish("", "movimientos_queue", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        bodyError,
	})

	if err != nil {
		log.Printf("❌ Error enviando mensaje de error: %v", err)
	} else {
		log.Printf("⚠️ Mensaje de error enviado para probar validaciones")
	}

	log.Println("🎉 Todos los mensajes de prueba enviados")
	log.Println("📋 Revisa los logs del API Movimiento para ver el procesamiento")
	log.Println("🌐 Consulta métricas en: http://localhost:8000/api/v1/metrics")
}