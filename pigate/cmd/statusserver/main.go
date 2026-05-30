package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"pigate/pkg/config"
	"pigate/pkg/messenger"
)

const application = "statusserver"

//go:embed static
var staticFiles embed.FS

type statusState struct {
	mu                 sync.RWMutex
	locationID         string
	gateStatus         string
	gateStatusAt       *time.Time
	credentialStatus   string
	credentialStatusAt *time.Time
	lastCommand        string
	lastCommandAt      *time.Time
}

type statusSnapshot struct {
	LocationID         string     `json:"location_id"`
	GateStatus         string     `json:"gate_status"`
	GateStatusAt       *time.Time `json:"gate_status_at,omitempty"`
	CredentialStatus   string     `json:"credential_status"`
	CredentialStatusAt *time.Time `json:"credential_status_at,omitempty"`
	LastCommand        string     `json:"last_command,omitempty"`
	LastCommandAt      *time.Time `json:"last_command_at,omitempty"`
	MQTTConnected      bool       `json:"mqtt_connected"`
	DBConnected        bool       `json:"db_connected"`
	DBError            string     `json:"db_error,omitempty"`
	ServerTime         time.Time  `json:"server_time"`
}

type dbHealth struct {
	connected bool
	err       string
}

type statusStore struct {
	db *sql.DB
}

type app struct {
	state *statusState
	store *statusStore
	mqtt  *messenger.MQTTClient
}

func main() {
	var configFilePath string
	flag.StringVar(&configFilePath, "c", "/workspace/pigate/pkg/config", "Path to the configuration file")
	flag.Parse()

	cfg := config.LoadConfig(configFilePath, application+"-config").(*config.StatusServerConfig)
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = "127.0.0.1:8090"
	}

	state := newStatusState(cfg.Location_ID)
	store := newStatusStore(connString(cfg.DB))
	if err := store.initSchema(context.Background()); err != nil {
		log.Printf("Status schema initialization failed: %v", err)
	}
	if err := store.loadLatest(context.Background(), state); err != nil {
		log.Printf("Status latest load failed: %v", err)
	}
	defer store.Close()

	client := messenger.NewMQTTClientWithCredentials(
		cfg.MQTT.Broker,
		application,
		cfg.Location_ID,
		cfg.MQTT.Username,
		cfg.MQTT.Password,
	)
	if err := client.Connect(); err != nil {
		log.Printf("Failed to connect to MQTT broker (%s): %v", cfg.MQTT.Broker, err)
	}

	serverApp := &app{
		state: state,
		store: store,
		mqtt:  client,
	}
	serverApp.subscribeToStatus()

	static, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to prepare static files: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/status", serverApp.handleStatus)
	mux.HandleFunc("POST /api/command", serverApp.handleCommand)
	mux.HandleFunc("GET /healthz", serverApp.handleHealth)
	mux.Handle("/", http.FileServer(http.FS(static)))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("PiGate status page listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Status server failed: %v", err)
	}
}

func newStatusState(locationID string) *statusState {
	return &statusState{
		locationID:       locationID,
		gateStatus:       "unknown",
		credentialStatus: "unknown",
	}
}

func (s *statusState) setGateStatus(status string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gateStatus = status
	s.gateStatusAt = &at
}

func (s *statusState) setCredentialStatus(status string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentialStatus = status
	s.credentialStatusAt = &at
}

func (s *statusState) setCommand(command string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastCommand = command
	s.lastCommandAt = &at
}

func (s *statusState) applyLatest(latest persistedLatest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if latest.gateStatus != "" {
		s.gateStatus = latest.gateStatus
	}
	s.gateStatusAt = latest.gateStatusAt
	if latest.credentialStatus != "" {
		s.credentialStatus = latest.credentialStatus
	}
	s.credentialStatusAt = latest.credentialStatusAt
	s.lastCommand = latest.lastCommand
	s.lastCommandAt = latest.lastCommandAt
}

func (s *statusState) snapshot(mqttConnected bool, health dbHealth) statusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return statusSnapshot{
		LocationID:         s.locationID,
		GateStatus:         s.gateStatus,
		GateStatusAt:       s.gateStatusAt,
		CredentialStatus:   s.credentialStatus,
		CredentialStatusAt: s.credentialStatusAt,
		LastCommand:        s.lastCommand,
		LastCommandAt:      s.lastCommandAt,
		MQTTConnected:      mqttConnected,
		DBConnected:        health.connected,
		DBError:            health.err,
		ServerTime:         time.Now(),
	}
}

func newStatusStore(connStr string) *statusStore {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Failed to create Postgres client: %v", err)
		return &statusStore{}
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &statusStore{db: db}
}

func (s *statusStore) Close() {
	if s.db != nil {
		_ = s.db.Close()
	}
}

func (s *statusStore) initSchema(parent context.Context) error {
	if s.db == nil {
		return errors.New("Postgres client is not configured")
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS pigate_status_events (
			id BIGSERIAL PRIMARY KEY,
			location_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			topic TEXT NOT NULL,
			payload TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS pigate_status_events_location_created_idx
			ON pigate_status_events (location_id, created_at DESC);`,
		`CREATE TABLE IF NOT EXISTS pigate_status_latest (
			location_id TEXT PRIMARY KEY,
			gate_status TEXT NOT NULL DEFAULT 'unknown',
			gate_status_at TIMESTAMPTZ,
			credential_status TEXT NOT NULL DEFAULT 'unknown',
			credential_status_at TIMESTAMPTZ,
			last_command TEXT,
			last_command_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (s *statusStore) health(parent context.Context) dbHealth {
	if s.db == nil {
		return dbHealth{connected: false, err: "Postgres client is not configured"}
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		return dbHealth{connected: false, err: err.Error()}
	}
	return dbHealth{connected: true}
}

type persistedLatest struct {
	gateStatus         string
	gateStatusAt       *time.Time
	credentialStatus   string
	credentialStatusAt *time.Time
	lastCommand        string
	lastCommandAt      *time.Time
}

func (s *statusStore) loadLatest(parent context.Context, state *statusState) error {
	if s.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()

	var latest persistedLatest
	var gateStatusAt sql.NullTime
	var credentialStatusAt sql.NullTime
	var lastCommand sql.NullString
	var lastCommandAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT gate_status, gate_status_at, credential_status, credential_status_at, last_command, last_command_at
		FROM pigate_status_latest
		WHERE location_id = $1
	`, state.locationID).Scan(
		&latest.gateStatus,
		&gateStatusAt,
		&latest.credentialStatus,
		&credentialStatusAt,
		&lastCommand,
		&lastCommandAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if gateStatusAt.Valid {
		latest.gateStatusAt = &gateStatusAt.Time
	}
	if credentialStatusAt.Valid {
		latest.credentialStatusAt = &credentialStatusAt.Time
	}
	if lastCommand.Valid {
		latest.lastCommand = lastCommand.String
	}
	if lastCommandAt.Valid {
		latest.lastCommandAt = &lastCommandAt.Time
	}
	state.applyLatest(latest)
	return nil
}

func (s *statusStore) recordGateStatus(parent context.Context, locationID, topic, payload string, at time.Time) error {
	if err := s.recordEvent(parent, locationID, "gate_status", topic, payload, at); err != nil {
		return err
	}
	return s.upsertGateStatus(parent, locationID, payload, at)
}

func (s *statusStore) recordCredentialStatus(parent context.Context, locationID, topic, payload string, at time.Time) error {
	if err := s.recordEvent(parent, locationID, "credential_status", topic, payload, at); err != nil {
		return err
	}
	return s.upsertCredentialStatus(parent, locationID, payload, at)
}

func (s *statusStore) recordCommand(parent context.Context, locationID, topic, payload string, at time.Time) error {
	if err := s.recordEvent(parent, locationID, "gate_command", topic, payload, at); err != nil {
		return err
	}
	return s.upsertCommand(parent, locationID, payload, at)
}

func (s *statusStore) recordEvent(parent context.Context, locationID, eventType, topic, payload string, at time.Time) error {
	if s.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pigate_status_events (location_id, event_type, topic, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, locationID, eventType, topic, payload, at)
	return err
}

func (s *statusStore) upsertGateStatus(parent context.Context, locationID, status string, at time.Time) error {
	if s.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pigate_status_latest (location_id, gate_status, gate_status_at, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (location_id) DO UPDATE SET
			gate_status = EXCLUDED.gate_status,
			gate_status_at = EXCLUDED.gate_status_at,
			updated_at = NOW()
	`, locationID, status, at)
	return err
}

func (s *statusStore) upsertCredentialStatus(parent context.Context, locationID, status string, at time.Time) error {
	if s.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pigate_status_latest (location_id, credential_status, credential_status_at, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (location_id) DO UPDATE SET
			credential_status = EXCLUDED.credential_status,
			credential_status_at = EXCLUDED.credential_status_at,
			updated_at = NOW()
	`, locationID, status, at)
	return err
}

func (s *statusStore) upsertCommand(parent context.Context, locationID, command string, at time.Time) error {
	if s.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pigate_status_latest (location_id, last_command, last_command_at, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (location_id) DO UPDATE SET
			last_command = EXCLUDED.last_command,
			last_command_at = EXCLUDED.last_command_at,
			updated_at = NOW()
	`, locationID, command, at)
	return err
}

func (a *app) subscribeToStatus() {
	if err := a.mqtt.SubscribePigateStatus(func(topic, status string) {
		now := time.Now()
		a.state.setGateStatus(status, now)
		if err := a.store.recordGateStatus(context.Background(), a.state.locationID, topic, status, now); err != nil {
			log.Printf("Failed to persist gate status: %v", err)
		}
	}); err != nil {
		log.Printf("Failed to subscribe to gate status: %v", err)
	}

	if err := a.mqtt.SubscribeCredentialStatus(func(topic, status string) {
		now := time.Now()
		a.state.setCredentialStatus(status, now)
		if err := a.store.recordCredentialStatus(context.Background(), a.state.locationID, topic, status, now); err != nil {
			log.Printf("Failed to persist credential status: %v", err)
		}
	}); err != nil {
		log.Printf("Failed to subscribe to credential status: %v", err)
	}
}

func (a *app) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.state.snapshot(a.mqtt.IsConnected(), a.store.health(r.Context())))
}

type commandRequest struct {
	Command string `json:"command"`
}

type commandResponse struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
}

func (a *app) handleCommand(w http.ResponseWriter, r *http.Request) {
	var req commandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid command request")
		return
	}

	command, mqttCommand, err := normalizeCommand(req.Command)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	switch command {
	case "open":
		err = a.mqtt.CommandOpen()
	case "lock_open":
		err = a.mqtt.CommandLockOpen()
	case "close":
		err = a.mqtt.CommandClose()
	default:
		err = fmt.Errorf("unsupported command %q", command)
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	now := time.Now()
	a.state.setCommand(command, now)
	topic := fmt.Sprintf(messenger.TopicPigateCommand, a.state.locationID)
	if err := a.store.recordCommand(r.Context(), a.state.locationID, topic, mqttCommand, now); err != nil {
		log.Printf("Failed to persist command: %v", err)
	}

	writeJSON(w, http.StatusOK, commandResponse{OK: true, Command: command})
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func normalizeCommand(command string) (uiCommand string, mqttCommand string, err error) {
	switch strings.ToLower(strings.TrimSpace(command)) {
	case "open":
		return "open", messenger.CommandOpenMessage, nil
	case "lock_open", "locked_open", "hold_open":
		return "lock_open", messenger.CommandHoldOpenMessage, nil
	case "close":
		return "close", messenger.CommandCloseMessage, nil
	default:
		return "", "", fmt.Errorf("command must be open, lock_open, or close")
	}
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func connString(db config.DBConfig) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.Name)
}
