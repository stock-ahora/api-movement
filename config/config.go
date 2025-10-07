package config

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type AppConfig struct {
	DBUser     string `json:"username"`
	DBPassword string `json:"password"`
	DBHost     string `json:"host"`
	DBPort     int    `json:"port"`
	DBName     string `json:"dbname"`
	MQHost     string `json:"MQ_HOST"`
	MQUser     string `json:"MQ_USER"`
	MQPassword string `json:"MQ_PASSWORD"`
	MQPort     int    `json:"MQ_PORT"`
	RabbitVHost string `json:"RMQ_VHOST"`
}

func LoadSecrets(secretName, region string) (*AppConfig, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	svc := secretsmanager.NewFromConfig(cfg)

	resp, err := svc.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		return nil, err
	}

	var appConfig AppConfig
	if err := json.Unmarshal([]byte(*resp.SecretString), &appConfig); err != nil {
		return nil, err
	}

	log.Println("✅ Secrets cargados correctamente")
	return &appConfig, nil
}
