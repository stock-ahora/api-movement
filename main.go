// main.go
package main

import (
	"api-movement/api"
	"api-movement/config"
	"api-movement/consumer"
	"api-movement/database"
	"api-movement/services"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

func main() {
	log.Println("=== Iniciando API y Consumidor de Movimientos ===")

	ctx := context.Background()
	cfg := getSecrets(ctx)

	// 2. Conectar a la base de datos (una sola vez)
	db, err := database.Connect(cfg.User, cfg.Pass, cfg.Host, cfg.Name, cfg.Port)
	if err != nil {
		log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
	}
	defer db.Close()
	log.Println("✅ Conectado a PostgreSQL")

	// 3. Inicializar servicios
	movimientoService := services.NewMovimientoService(db)

	// Usamos un errgroup para manejar el ciclo de vida del consumidor y el servidor API
	// Si uno falla, el otro también se detendrá.
	g, ctx := errgroup.WithContext(context.Background())

	// Goroutine para el consumidor de RabbitMQ
	g.Go(func() error {
		return startRabbitMQConsumer(ctx, cfg, movimientoService)

	})

	// Goroutine para el servidor HTTP API
	g.Go(func() error {
		return startAPIServer(ctx, cfg, movimientoService)
	})

	// Esperar a que una señal de interrupción (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-c:
		log.Println("🔌 Apagado solicitado por señal...")
	case <-ctx.Done():
		log.Println("🔌 Apagado por error en un servicio...")
	}

	// Esperar a que el errgroup termine
	if err := g.Wait(); err != nil && err != context.Canceled && err != http.ErrServerClosed {
		log.Fatalf("💥 Error fatal en un servicio: %v", err)
	}

	log.Println("👋 Aplicación detenida correctamente.")
}

func startRabbitMQConsumer(ctx context.Context, cfg *config.SecretApp, svc *services.MovimientoService) error {

	rootCAs, _ := x509.SystemCertPool()
	tlsConfig := &tls.Config{RootCAs: rootCAs}
	rabbitURL := fmt.Sprintf("amqps://%s:%s@%s:%d/",
		cfg.MQ_USER, cfg.MQ_PASSWORD, cfg.MQ_HOST, cfg.MQ_PORT)

	conn, err := amqp.DialTLS(rabbitURL, tlsConfig)
	if err != nil {
		return fmt.Errorf("error conectando a RabbitMQ: %w", err)
	}
	defer conn.Close()
	log.Println("✅ Conectado a RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("error creando canal: %w", err)
	}
	defer ch.Close()

	queueName := "movement.generated"
	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("error declarando queue: %w", err)
	}

	// Escuchar por la cancelación del contexto para detener el consumidor
	go func() {
		<-ctx.Done()
		log.Println("🔌 Deteniendo consumidor de RabbitMQ...")
		ch.Close()
		conn.Close()
	}()

	consum := consumer.Start(ch, queueName, svc)

	return consum
}

func startAPIServer(ctx context.Context, cfg *config.SecretApp, svc *services.MovimientoService) error {
	handler := api.NewMovementHandler(svc)
	router := api.NewRouter(handler)

	server := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		log.Println("🔌 Deteniendo servidor API...")
		server.Shutdown(context.Background())
	}()

	log.Println("🚀 Servidor HTTP escuchando en el puerto :8081")
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("error en servidor API: %w", err)
	}
	return nil
}

func getSecrets(ctx context.Context) *config.SecretApp {
	cfg, err := config.LoadSecretManager(ctx)
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	return cfg
}
