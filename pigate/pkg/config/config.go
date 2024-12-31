package config

import (
	"log"

	"github.com/spf13/viper"
)

type CredentialServerConfig struct {
	MQTTBroker         string
	Location_ID        string
	CredentialFileName string
	FileWatcherPath    string
}

type GateControllerConfig struct {
	MQTTBroker         string
	Location_ID        string
	CredentialFileName string
	GateOpenDuration   int
	GPIOPin            int
	DatabasePath       string
}

func LoadConfig(configPath, component string) interface{} {
	v := viper.New()
	v.SetConfigName(component) // Use component-specific config file
	v.SetConfigType("toml")
	v.AddConfigPath(configPath)
	v.AutomaticEnv()         // Bind environment variables
	v.SetEnvPrefix("PIGATE") // Environment variable prefix

	if err := v.ReadInConfig(); err != nil {
		log.Printf("Config file not found for %s, using defaults and environment variables: %v", component, err)
	}

	switch component {
	case "credentialserver-config":
		return &CredentialServerConfig{
			MQTTBroker:         v.GetString("MQTT_BROKER"),
			Location_ID:        v.GetString("LOCATION_ID"),
			CredentialFileName: v.GetString("CREDENTIAL_FILE_NAME"),
			FileWatcherPath:    v.GetString("FILE_WATCHER_PATH"),
		}
	case "gatecontroller-config":
		return &GateControllerConfig{
			MQTTBroker:         v.GetString("MQTT_BROKER"),
			Location_ID:        v.GetString("LOCATION_ID"),
			CredentialFileName: v.GetString("CREDENTIAL_FILE_NAME"),
			GateOpenDuration:   v.GetInt("GATE_OPEN_DURATION"),
			GPIOPin:            v.GetInt("GPIO_PIN"),
			DatabasePath:       v.GetString("DATABASE_PATH"),
		}
	default:
		log.Fatalf("Unknown component: %s", component)
		return nil
	}
}
