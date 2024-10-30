package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	MQTTBroker       string
	MQTTTopic        string
	GateOpenDuration int
	GPIOPin          int
	DatabasePath     string
	ServerURL        string
	HTTPPort         int
}

func LoadConfig() *Config {
	return &Config{
		MQTTBroker:       getEnv("MQTT_BROKER", "tcp://broker.hivemq.com:1883"),
		MQTTTopic:        getEnv("MQTT_TOPIC", "pigate/updates"),
		GateOpenDuration: getEnvInt("GATE_OPEN_DURATION", 10),
		GPIOPin:          getEnvInt("GPIO_PIN", 17),
		DatabasePath:     getEnv("DATABASE_PATH", "pigate.db"),
		ServerURL:        getEnv("SERVER_URL", "http://localhost"),
		HTTPPort:         getEnvInt("HTTP_PORT", 8080),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid integer for %s: %s. Using default %d.", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}
