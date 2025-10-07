package config

import "os"

type Config struct {
    RabbitUser  string
    RabbitPass  string
    RabbitHost  string
    RabbitPort  string
    RabbitVHost string
    DBHost      string
    DBPort      string
    DBUser      string
    DBPass      string
    DBName      string
}

func getEnv(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}