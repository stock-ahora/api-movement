package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"api-movimiento/config"
	"api-movimiento/database"
	"api-movimiento/handlers"
	"api-movimiento/services"

	"github.com/gin-gonic/gin"
)

func main() {
	// Obtener configuración de AWS Secret Manager
	cfg, err := config.LoadLocalConfig()
	if err != nil {
		log.Fatalf("❌ Error obteniendo configuración: %v", err)
	}

	// Conectar a PostgreSQL
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Error conectando a base de datos: %v", err)
	}
	defer db.Close()

	// Inicializar tablas
	if err := database.InitTables(db); err != nil {
		log.Fatalf("❌ Error inicializando base de datos: %v", err)
	}

	// Conectar a RabbitMQ
	conn, ch, err := config.NewRabbitMQ(cfg.RabbitMQConfig)
	if err != nil {
		log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// Declarar colas
	err = services.DeclareQueues(ch)
	if err != nil {
		log.Fatalf("❌ Error declarando colas: %v", err)
	}

	// Crear servicio de movimientos
	movService := services.NewMovimientoService(db, cfg, ch)

	// Iniciar consumer de RabbitMQ en goroutine
	go func() {
		log.Println("🐰 Iniciando consumer de RabbitMQ...")
		if err := movService.StartConsumer(); err != nil {
			log.Fatalf("❌ Error iniciando consumer: %v", err)
		}
	}()

	// Configurar router HTTP (solo para consultas)
	router := setupRouter(movService)

	// Canal para manejar señales del sistema
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor HTTP en goroutine
	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8000"
		}

		log.Printf("🚀 Servidor API Movimientos iniciando en puerto %s", port)
		log.Printf("📊 Endpoints disponibles:")
		log.Printf("   GET  /api/v1/movimientos - Listar movimientos")
		log.Printf("   GET  /api/v1/movimientos/producto/{id}/trazabilidad - Trazabilidad completa")
		log.Printf("   GET  /api/v1/movimientos/sku/{id} - Movimientos por SKU")
		log.Printf("   GET  /api/v1/health - Health check")
		log.Printf("   GET  /api/v1/metrics - Métricas del sistema")
		log.Printf("🐰 Consumer RabbitMQ escuchando en: movimientos_queue")

		if err := router.Run(":" + port); err != nil {
			log.Fatalf("❌ Error iniciando servidor: %v", err)
		}
	}()

	// Esperar señal de terminación
	<-sigChan
	log.Println("🛑 Cerrando aplicación...")
}

func setupRouter(movService *services.MovimientoService) *gin.Engine {
	router := gin.Default()

	// Middlewares
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Crear handlers
	movHandler := handlers.NewMovimientoHandler(movService)
	healthHandler := handlers.NewHealthHandler(movService)

	// Rutas API (solo consultas)
	api := router.Group("/api/v1")
	{
		// Consultas de movimientos
		api.GET("/movimientos", movHandler.ListarMovimientos)
		api.GET("/movimientos/producto/:product_id/trazabilidad", movHandler.ObtenerTrazabilidad)
		api.GET("/movimientos/sku/:sku_id", movHandler.ObtenerMovimientosSku)
		api.GET("/movimientos/request/:request_id", movHandler.ObtenerMovimientosRequest)

		// Health y métricas
		api.GET("/health", healthHandler.HealthCheck)
		api.GET("/metrics", healthHandler.GetMetrics)
	}

	return router
}
