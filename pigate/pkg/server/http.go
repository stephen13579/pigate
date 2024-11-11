package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"pigate/pkg/config"
)

var mqttPublisher MQTTClientInterface

// StartHTTPServer initializes the HTTP server and sets up the routes.
func StartHTTPServer(cfg *config.Config, publisher MQTTClientInterface) error {
	mqttPublisher = publisher // Assign the MQTT publisher

	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/credentials", serveCredentials)

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	log.Printf("Starting HTTP server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// handleUpload handles file uploads and notifies gate controllers via MQTT.
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse and save the uploaded file
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save the file to resources/credentials.json
	outFile, err := os.Create("resources/credentials.json")
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	_, err = outFile.ReadFrom(file)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	// Notify gate controllers via MQTT
	err = NotifyGateControllers(mqttPublisher, "gate/notifications")
	if err != nil {
		log.Printf("Failed to notify gate controllers: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}

// serveCredentials serves the credentials file to clients.
func serveCredentials(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "resources/credentials.json")
}
