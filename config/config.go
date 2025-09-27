package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/streadway/amqp"
)

// MQConfig configuración de RabbitMQ
type MQConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	VHost    string `json:"vhost"`
}

// Config configuración general de la aplicación
type Config struct {
	DatabaseURL     string   `json:"database_url"`
	EncryptionKey   string   `json:"encryption_key"`
	StockAPIURL     string   `json:"stock_api_url"`
	StockAPIKey     string   `json:"stock_api_key"`
	RedisURL        string   `json:"redis_url"`
	RabbitMQConfig  MQConfig `json:"rabbitmq_config"`
}

// LoadFromSecretManager carga configuración desde AWS Secret Manager
func LoadFromSecretManager(secretName string) (*Config, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		return nil, fmt.Errorf("error creando sesión AWS: %w", err)
	}

	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := svc.GetSecretValue(input)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo secreto: %w", err)
	}

	var config Config
	err = json.Unmarshal([]byte(*result.SecretString), &config)
	if err != nil {
		return nil, fmt.Errorf("error parseando configuración: %w", err)
	}

	log.Println("✅ Configuración cargada desde AWS Secret Manager")
	return &config, nil
}

// NewRabbitMQ crea conexión a RabbitMQ con TLS
func NewRabbitMQ(mq MQConfig) (*amqp.Connection, *amqp.Channel, error) {
	rootCAs, _ := x509.SystemCertPool()

	tlsConfig := &tls.Config{
		RootCAs: rootCAs,
	}

	url := fmt.Sprintf("amqps://%s:%s@%s:%d/%s", mq.User, mq.Password, mq.Host, mq.Port, mq.VHost)

	conn, err := amqp.DialTLS(url, tlsConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("error conectando a RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("error creando canal: %w", err)
	}

	log.Println("✅ Conectado a RabbitMQ con TLS")
	return conn, ch, nil
}

// QueueNames nombres de las colas
var QueueNames = struct {
	MovimientosQueue    string
	StockUpdatesQueue   string
	OCRProcessedQueue   string
	NotificationsQueue  string
}{
	MovimientosQueue:   "movimientos_queue",
	StockUpdatesQueue:  "stock_updates_queue",
	OCRProcessedQueue:  "ocr_processed_queue",
	NotificationsQueue: "notifications_queue",
}