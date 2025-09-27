package services

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"api-movimiento/config"
	"api-movimiento/models"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

// MovimientoService servicio principal para manejar movimientos
type MovimientoService struct {
	db       *sql.DB
	config   *config.Config
	rabbitCh *amqp.Channel
}

// NewMovimientoService crea nueva instancia del servicio
func NewMovimientoService(db *sql.DB, cfg *config.Config, rabbitCh *amqp.Channel) *MovimientoService {
	return &MovimientoService{
		db:       db,
		config:   cfg,
		rabbitCh: rabbitCh,
	}
}

// DeclareQueues declara todas las colas necesarias
func DeclareQueues(ch *amqp.Channel) error {
	queues := []string{
		config.QueueNames.MovimientosQueue,
		config.QueueNames.StockUpdatesQueue,
		config.QueueNames.OCRProcessedQueue,
		config.QueueNames.NotificationsQueue,
	}

	for _, queueName := range queues {
		_, err := ch.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			return fmt.Errorf("error declarando cola %s: %w", queueName, err)
		}
	}

	log.Printf("✅ Colas declaradas: %v", queues)
	return nil
}

// StartConsumer inicia el consumer de RabbitMQ
func (s *MovimientoService) StartConsumer() error {
	// Configurar QoS
	err := s.rabbitCh.Qos(
		5,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("error configurando QoS: %w", err)
	}

	// Consumer principal para movimientos
	msgs, err := s.rabbitCh.Consume(
		config.QueueNames.MovimientosQueue, // queue
		"movimiento-consumer",               // consumer
		false,                               // auto-ack
		false,                               // exclusive
		false,                               // no-local
		false,                               // no-wait
		nil,                                 // args
	)
	if err != nil {
		return fmt.Errorf("error iniciando consumer: %w", err)
	}

	// Procesar mensajes
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("📨 Mensaje recibido: %s", string(d.Body))

			if err := s.procesarMensajeMovimiento(d.Body); err != nil {
				log.Printf("❌ Error procesando mensaje: %v", err)
				// Rechazar mensaje y enviarlo a cola de error
				d.Nack(false, false)
			} else {
				log.Printf("✅ Mensaje procesado exitosamente")
				d.Ack(false)
			}
		}
	}()

	log.Printf("🐰 Consumer iniciado. Esperando mensajes en %s...", config.QueueNames.MovimientosQueue)
	<-forever

	return nil
}

// procesarMensajeMovimiento procesa un mensaje de movimiento desde RabbitMQ
func (s *MovimientoService) procesarMensajeMovimiento(body []byte) error {
	var msg models.MovimientoMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("error parseando mensaje: %w", err)
	}

	// Validar mensaje
	if err := s.validarMensaje(&msg); err != nil {
		return fmt.Errorf("mensaje inválido: %w", err)
	}

	// Procesar movimiento
	movimiento, err := s.ProcesarMovimiento(&msg)
	if err != nil {
		return fmt.Errorf("error procesando movimiento: %w", err)
	}

	// Enviar notificación
	go s.enviarNotificacion(movimiento)

	log.Printf("✅ Movimiento procesado: ID=%d, Producto=%s, Tipo=%s, Cantidad=%d",
		movimiento.ID, movimiento.ProductID, movimiento.TipoMovimiento, movimiento.Cantidad)

	return nil
}

// validarMensaje valida la estructura del mensaje
func (s *MovimientoService) validarMensaje(msg *models.MovimientoMessage) error {
	if msg.ProductID == uuid.Nil {
		return fmt.Errorf("product_id requerido")
	}
	if msg.ClientAccountID == uuid.Nil {
		return fmt.Errorf("client_account_id requerido")
	}
	if msg.TipoMovimiento == "" {
		return fmt.Errorf("tipo_movimiento requerido")
	}
	if msg.TipoMovimiento != "entrada" && msg.TipoMovimiento != "salida" && msg.TipoMovimiento != "ajuste" {
		return fmt.Errorf("tipo_movimiento debe ser: entrada, salida o ajuste")
	}
	if msg.Cantidad <= 0 {
		return fmt.Errorf("cantidad debe ser mayor a 0")
	}
	return nil
}

// ProcesarMovimiento procesa un movimiento completo
func (s *MovimientoService) ProcesarMovimiento(msg *models.MovimientoMessage) (*models.Movimiento, error) {
	// Iniciar transacción
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}
	defer tx.Rollback()

	// 1. Obtener stock actual del producto
	product, err := s.getProductStock(msg.ProductID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo producto: %w", err)
	}

	// 2. Calcular nuevo stock
	cantidadAnterior := product.Stock
	cantidadNueva, err := s.calcularNuevoStock(msg.TipoMovimiento, cantidadAnterior, msg.Cantidad)
	if err != nil {
		return nil, err
	}

	// 3. Actualizar stock en API Stock
	if err := s.updateProductStock(msg.ProductID, cantidadNueva); err != nil {
		return nil, fmt.Errorf("error actualizando stock: %w", err)
	}

	// 4. Crear movimiento en BD
	movimiento, err := s.crearMovimiento(tx, msg, cantidadAnterior, cantidadNueva)
	if err != nil {
		return nil, fmt.Errorf("error creando movimiento: %w", err)
	}

	// 5. Confirmar transacción
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error confirmando transacción: %w", err)
	}

	return movimiento, nil
}

// calcularNuevoStock calcula el nuevo stock según el tipo de movimiento
func (s *MovimientoService) calcularNuevoStock(tipoMovimiento string, stockActual, cantidad int64) (int64, error) {
	switch tipoMovimiento {
	case "entrada":
		return stockActual + cantidad, nil
	case "salida":
		if stockActual < cantidad {
			return 0, fmt.Errorf("stock insuficiente: actual=%d, solicitado=%d", stockActual, cantidad)
		}
		return stockActual - cantidad, nil
	case "ajuste":
		return cantidad, nil
	default:
		return 0, fmt.Errorf("tipo de movimiento no válido: %s", tipoMovimiento)
	}
}

// crearMovimiento crea el registro de movimiento en la BD
func (s *MovimientoService) crearMovimiento(tx *sql.Tx, msg *models.MovimientoMessage, cantidadAnterior, cantidadNueva int64) (*models.Movimiento, error) {
	// Encriptar datos sensibles
	datosEncriptados, err := s.encryptData(msg)
	if err != nil {
		return nil, fmt.Errorf("error encriptando datos: %w", err)
	}

	// Query de inserción
	query := `
		INSERT INTO movimientos (
			product_id, sku_id, request_id, tipo_movimiento, cantidad,
			cantidad_anterior, cantidad_nueva, fecha_movimiento, usuario_id,
			motivo, client_account_id, document_id, origen, datos_encriptados,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id`

	now := time.Now()
	var movimientoID int

	err = tx.QueryRow(
		query,
		msg.ProductID,
		msg.SkuID,
		msg.RequestID,
		msg.TipoMovimiento,
		msg.Cantidad,
		cantidadAnterior,
		cantidadNueva,
		msg.Timestamp,
		msg.UsuarioID,
		msg.Motivo,
		msg.ClientAccountID,
		msg.DocumentID,
		msg.Origen,
		datosEncriptados,
		now,
		now,
	).Scan(&movimientoID)

	if err != nil {
		return nil, fmt.Errorf("error insertando movimiento: %w", err)
	}

	// Crear objeto de respuesta
	movimiento := &models.Movimiento{
		ID:               movimientoID,
		ProductID:        msg.ProductID,
		SkuID:            msg.SkuID,
		RequestID:        msg.RequestID,
		TipoMovimiento:   msg.TipoMovimiento,
		Cantidad:         msg.Cantidad,
		CantidadAnterior: cantidadAnterior,
		CantidadNueva:    cantidadNueva,
		FechaMovimiento:  msg.Timestamp,
		UsuarioID:        msg.UsuarioID,
		Motivo:           msg.Motivo,
		ClientAccountID:  msg.ClientAccountID,
		DocumentID:       msg.DocumentID,
		Origen:           msg.Origen,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	return movimiento, nil
}

// getProductStock obtiene el producto del API Stock
func (s *MovimientoService) getProductStock(productID uuid.UUID) (*models.Product, error) {
	url := fmt.Sprintf("%s/products/%s", s.config.StockAPIURL, productID.String())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.config.StockAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error obteniendo producto: status %d", resp.StatusCode)
	}

	var product models.Product
	err = json.NewDecoder(resp.Body).Decode(&product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

// updateProductStock actualiza el stock en el API Stock
func (s *MovimientoService) updateProductStock(productID uuid.UUID, nuevoStock int64) error {
	url := fmt.Sprintf("%s/products/%s", s.config.StockAPIURL, productID.String())

	updateData := models.StockUpdateRequest{
		Stock: nuevoStock,
	}

	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.config.StockAPIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error actualizando stock: status %d", resp.StatusCode)
	}

	return nil
}

// encryptData encripta datos sensibles
func (s *MovimientoService) encryptData(msg *models.MovimientoMessage) (string, error) {
	dataToEncrypt := map[string]interface{}{
		"product_id":         msg.ProductID,
		"sku_id":            msg.SkuID,
		"cantidad":          msg.Cantidad,
		"usuario_id":        msg.UsuarioID,
		"client_account_id": msg.ClientAccountID,
		"request_id":        msg.RequestID,
		"document_id":       msg.DocumentID,
		"metadata":          msg.Metadata,
	}

	jsonData, err := json.Marshal(dataToEncrypt)
	if err != nil {
		return "", err
	}

	return encrypt(string(jsonData), s.config.EncryptionKey)
}

// enviarNotificacion envía notificación a cola de notificaciones
func (s *MovimientoService) enviarNotificacion(movimiento *models.Movimiento) {
	notification := models.NotificationMessage{
		Type:      "movimiento_procesado",
		ProductID: movimiento.ProductID,
		Message:   fmt.Sprintf("Movimiento %s procesado: %d unidades", movimiento.TipoMovimiento, movimiento.Cantidad),
		Data: map[string]interface{}{
			"movimiento_id":     movimiento.ID,
			"tipo_movimiento":   movimiento.TipoMovimiento,
			"cantidad":          movimiento.Cantidad,
			"stock_anterior":    movimiento.CantidadAnterior,
			"stock_nuevo":       movimiento.CantidadNueva,
			"client_account_id": movimiento.ClientAccountID,
		},
		Timestamp: time.Now(),
		Severity:  "info",
	}

	body, err := json.Marshal(notification)
	if err != nil {
		log.Printf("❌ Error serializando notificación: %v", err)
		return
	}

	err = s.rabbitCh.Publish(
		"",                               // exchange
		config.QueueNames.NotificationsQueue, // routing key
		false,                            // mandatory
		false,                            // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		log.Printf("❌ Error publicando notificación: %v", err)
	} else {
		log.Printf("📢 Notificación enviada para movimiento ID: %d", movimiento.ID)
	}
}

// Funciones de encriptación (AES)
func encrypt(plaintext, key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(ciphertext, key string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	nonce, cipherBytes := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, cipherBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}