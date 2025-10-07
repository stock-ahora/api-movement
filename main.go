package main

import (
    "api-movement/config"
    "api-movement/database"
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "log"

    "github.com/streadway/amqp"
)

func main() {
    log.Println("=== Starting Movement Consumer ===")

    cfg, err := config.LoadSecretManager(nil)
    if err != nil {
        log.Fatalf("❌ Error cargando config: %v", err)
    }

    // Conectar a PostgreSQL
    db, err := database.Connect(cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
    if err != nil {
        log.Fatalf("❌ Error conectando a PostgreSQL: %v", err)
    }
    defer db.Close()
    log.Println("✅ Conectado a PostgreSQL")

    // Conectar a RabbitMQ con TLS
    rootCAs, _ := x509.SystemCertPool()
    tlsConfig := &tls.Config{RootCAs: rootCAs}

    rabbitURL := fmt.Sprintf("amqps://%s:%s@%s:%s/%s",
        cfg.RabbitUser, cfg.RabbitPass, cfg.RabbitHost, cfg.RabbitPort, cfg.RabbitVHost)

    conn, err := amqp.DialTLS(rabbitURL, tlsConfig)
    if err != nil {
        log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
    }
    defer conn.Close()

    ch, err := conn.Channel()
    if err != nil {
        log.Fatalf("❌ Error creando canal: %v", err)
    }
    defer ch.Close()
    log.Println("✅ Conectado a RabbitMQ")

    // Declarar queue
    queueName := "movement.generated"
    queue, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
    if err != nil {
        log.Fatalf("❌ Error declarando queue: %v", err)
    }

    // Consumir mensajes
    msgs, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
    if err != nil {
        log.Fatalf("❌ Error iniciando consumer: %v", err)
    }

    log.Printf("✅ Esperando mensajes en queue '%s'...", queueName)

    forever := make(chan bool)

    go func() {
        for msg := range msgs {
            log.Printf("📦 Mensaje recibido: %s", string(msg.Body))
            msg.Ack(false)
        }
    }()

    <-forever
}