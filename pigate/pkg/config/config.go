package config

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

type CredentialServerConfig struct {
	MQTTBroker      string
	Location_ID     string
	Remote_DB_Table string
	FileWatcherPath string
	DB              DBConfig
}

type GateControllerConfig struct {
	MQTTBroker       string
	Location_ID      string
	Remote_DB_Table  string
	GateOpenDuration int
	RelayPin         int
	LocalDBPath      string
	DB               DBConfig
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
		// Get the env var name from config, then get its value from the environment
		DB_PASSWORD_ENV := v.GetString("DB_PASSWORD_ENV")
		dbPassword := ""
		if DB_PASSWORD_ENV != "" {
			dbPassword = os.Getenv(DB_PASSWORD_ENV)
		}
		return &CredentialServerConfig{
			MQTTBroker:      v.GetString("MQTT_BROKER"),
			Location_ID:     v.GetString("LOCATION_ID"),
			FileWatcherPath: v.GetString("FILE_WATCHER_PATH"),
			Remote_DB_Table: v.GetString("REMOTE_DB_TABLE"),
			DB: DBConfig{
				Host:     v.GetString("DB_HOST"),
				Port:     v.GetInt("DB_PORT"),
				Name:     v.GetString("DB_NAME"),
				User:     v.GetString("DB_USER"),
				Password: dbPassword,
			},
		}
	case "gatecontroller-config":
		// Get the env var name from config, then get its value from the environment
		DB_PASSWORD_ENV := v.GetString("DB_PASSWORD_ENV")
		dbPassword := ""
		if DB_PASSWORD_ENV != "" {
			dbPassword = os.Getenv(DB_PASSWORD_ENV)
		}
		return &GateControllerConfig{
			MQTTBroker:       v.GetString("MQTT_BROKER"),
			Location_ID:      v.GetString("LOCATION_ID"),
			GateOpenDuration: v.GetInt("GATE_OPEN_DURATION"),
			RelayPin:         v.GetInt("GATE_CONTROL_PIN"),
			LocalDBPath:      v.GetString("DATABASE_PATH"),
			Remote_DB_Table:  v.GetString("REMOTE_DB_TABLE"),
			DB: DBConfig{
				Host:     v.GetString("DB_HOST"),
				Port:     v.GetInt("DB_PORT"),
				Name:     v.GetString("DB_NAME"),
				User:     v.GetString("DB_USER"),
				Password: dbPassword,
			},
		}
	default:
		log.Fatalf("Unknown component: %s", component)
		return nil
	}
}
