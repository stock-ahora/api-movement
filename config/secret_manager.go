package config

import (
    "context"
    "log"
    "os"
    "github.com/joho/godotenv"
)

func init() {
    godotenv.Load()
}

func LoadSecretManager(ctx context.Context) (*Config, error) {
    log.Println("Cargando configuración desde .env")

    return &Config{
        RabbitUser:  os.Getenv("RABBIT_USER"),
        RabbitPass:  os.Getenv("RABBIT_PASSWORD"),
        RabbitHost:  os.Getenv("RABBIT_HOST"),
        RabbitPort:  os.Getenv("RABBIT_PORT"),
        RabbitVHost: os.Getenv("RABBIT_VHOST"),
        DBHost:      os.Getenv("DB_HOST"),
        DBPort:      os.Getenv("DB_PORT"),
        DBUser:      os.Getenv("DB_USER"),
        DBPass:      os.Getenv("DB_PASSWORD"),
        DBName:      os.Getenv("DB_NAME"),
    }, nil
}