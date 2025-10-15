package config

import (
	"api-movement/config_lib"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func LoadSecretManager(ctx context.Context) (*SecretApp, error) {

	secretID := getEnv("APP_SECRET_ID", "EMPTY")
	log.Printf("config_lib ID %s", secretID)

	if secretID == "" {
		return nil, fmt.Errorf("APP_SECRET_ID no definido")
	}

	//todo: cambiar la region por la variable de entorno
	region := "us-east-2"

	sm, err := config_lib.New(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("crear secrets manager: %w", err)
	}

	raw, err := sm.GetSecretString(ctx, secretID, "AWSCURRENT")
	if err != nil {
		return nil, fmt.Errorf("obtener secreto: %w", err)
	}

	//todo: eliminar este log
	log.Printf("Secreto obtenido: %s", raw)

	var cfg SecretApp
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("parsear secreto  JSON: %w", err)
	}
	return &cfg, nil
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No se encontró archivo .env: %v", err)
	}
}
