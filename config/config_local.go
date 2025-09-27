package config

// LoadLocalConfig carga configuración local para desarrollo
func LoadLocalConfig() (*Config, error) {
	return &Config{
		DatabaseURL:   "postgresql://postgres:password123@localhost:5432/movimientos_db",
		EncryptionKey: "dGVzdF9lbmNyeXB0aW9uX2tleV8zMl9ieXRlcw==", // 32 bytes base64
		StockAPIURL:   "http://localhost:8001",
		StockAPIKey:   "test-api-key",
		RedisURL:      "redis://localhost:6379/0",
		RabbitMQConfig: MQConfig{
			User:     "admin",
			Password: "admin123",
			Host:     "localhost",
			Port:     5672, // Sin TLS para desarrollo local
			VHost:    "/",
		},
	}, nil
}