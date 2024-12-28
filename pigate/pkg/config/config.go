package config

import (
	"context"
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	MQTTBroker       string
	MQTTTopic        string
	GateOpenDuration int
	GPIOPin          int
	DatabasePath     string
	ServerURL        string
	HTTPPort         int
	S3Bucket         string
	AppContext       context.Context
}

func LoadConfig(configPath string) *Config {
	// Initialize Viper
	v := viper.New()
	v.SetConfigName("config") // Name of the config file without extension
	v.SetConfigType("toml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()         // Automatically bind environment variables
	v.SetEnvPrefix("PIGATE") // Environment variable prefix

	// Default values
	v.SetDefault("MQTT_BROKER", "tcp://emqx:1883")
	v.SetDefault("MQTT_TOPIC", "pigate/updates")
	v.SetDefault("GATE_OPEN_DURATION", 30)
	v.SetDefault("GPIO_PIN", 17)
	v.SetDefault("DATABASE_PATH", "pigate.db")
	v.SetDefault("SERVER_URL", "http://localhost")
	v.SetDefault("HTTP_PORT", 8080)
	v.SetDefault("S3_CREDENTIAL_BUCKET", "pigate-speedway-self-storage")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults and environment variables: %v", err)
	}

	// Context with timeout
	appContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return &Config{
		MQTTBroker:       v.GetString("MQTT_BROKER"),
		MQTTTopic:        v.GetString("MQTT_TOPIC"),
		GateOpenDuration: v.GetInt("GATE_OPEN_DURATION"),
		GPIOPin:          v.GetInt("GPIO_PIN"),
		DatabasePath:     v.GetString("DATABASE_PATH"),
		ServerURL:        v.GetString("SERVER_URL"),
		HTTPPort:         v.GetInt("HTTP_PORT"),
		S3Bucket:         v.GetString("S3_CREDENTIAL_BUCKET"),
		AppContext:       appContext,
	}
}
