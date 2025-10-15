package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"

	"github.com/streadway/amqp"
)

func NewRabbitMq(cfg *SecretApp) (*amqp.Connection, *amqp.Channel) {
	rootCAs, _ := x509.SystemCertPool()
	tlsConfig := &tls.Config{RootCAs: rootCAs}

	url := fmt.Sprintf("amqps://%s:%s@%s:%d/%s", cfg.MQ_USER, cfg.MQ_PASSWORD, cfg.MQ_HOST, cfg.MQ_PORT, cfg.MQ_VHOST)
	conn, err := amqp.DialTLS(url, tlsConfig)
	if err != nil {
		log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Error creando canal: %v", err)
	}
	return conn, ch
}
