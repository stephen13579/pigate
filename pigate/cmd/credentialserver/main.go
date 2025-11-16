package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/kardianos/service"

	"pigate/pkg/config"
	"pigate/pkg/credentialparser"
	"pigate/pkg/messenger"
)

const application = "credentialserver"

type program struct {
	wg       sync.WaitGroup
	stopChan chan struct{}
}

func (p *program) Start(s service.Service) error {
	// Start should not block, so run the main logic in a goroutine.
	p.stopChan = make(chan struct{})
	p.wg.Add(1)
	go p.run()
	return nil
}

func (p *program) run() {
	defer p.wg.Done()

	// You can keep your existing main logic here, with a small tweak
	// to support stopping via p.stopChan instead of `select {}` forever.

	// 1) Parse command-line flags for config path
	var configFilePath string
	flag.StringVar(&configFilePath, "c", "/workspace/pigate/pkg/config",
		"Path to the configuration file")
	flag.Parse()

	// 2) Load configuration for credentialserver
	cfg := config.LoadConfig(configFilePath, application+"-config").(*config.CredentialServerConfig)

	// 3) Create messenger
	client := messenger.NewMQTTClient(cfg.MQTTBroker, application, cfg.Location_ID)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect to MQTT broker (%s): %v", cfg.MQTTBroker, err)
	}
	defer client.Disconnect()

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name)

	// 4) Parse credential file
	filePath, err := credentialparser.FindTextFile(cfg.FileWatcherPath)
	if err != nil {
		log.Printf("Did not find credential file on startup, this is fine.")
	} else {
		if err := credentialparser.HandleFile(filePath, connStr); err != nil {
			log.Printf("failed to handle file update: %s", err)
		} else {
			client.NotifyNewCredentials()
		}
	}

	// 5) Start FileWatcher for credential file
	fileWatcher := credentialparser.NewFileWatcher(cfg.FileWatcherPath, func(filePath string) {
		if err := credentialparser.HandleFile(filePath, connStr); err != nil {
			log.Printf("failed to handle file update: %s", err)
		} else {
			client.NotifyNewCredentials()
		}
	})

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if err := fileWatcher.Start(); err != nil {
			log.Printf("file watcher failed: %v", err)
		}
	}()

	// Block until service stop is requested
	<-p.stopChan
	log.Printf("credentialserver service stopping...")
	// If fileWatcher has a Stop/Close method, call it here.
	// fileWatcher.Stop()
}

func (p *program) Stop(s service.Service) error {
	// Signal run() to exit and wait for cleanup
	close(p.stopChan)
	p.wg.Wait()
	return nil
}

func main() {
	logFile, err := os.OpenFile("C:\\ProgramData\\CredentialServer\\credentialserver.log",
		os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	log.SetOutput(logFile)

	svcConfig := &service.Config{
		Name:        "CredentialServer",
		DisplayName: "Credential Server",
		Description: "Watches credential files and updates database / MQTT.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Support "install", "uninstall", "start", "stop" from command line
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			if err := s.Install(); err != nil {
				log.Fatalf("Install failed: %v", err)
			}
			log.Print("Service installed")
			return
		case "uninstall":
			if err := s.Uninstall(); err != nil {
				log.Fatalf("Uninstall failed: %v", err)
			}
			log.Print("Service uninstalled")
			return
		case "start":
			if err := s.Start(); err != nil {
				log.Fatalf("Start failed: %v", err)
			}
			log.Print("Service started")
			return
		case "stop":
			if err := s.Stop(); err != nil {
				log.Fatalf("Stop failed: %v", err)
			}
			log.Print("Service stopped")
			return
		}
	}

	// If no special arg, run normally (as a foreground process or service)
	if err := s.Run(); err != nil {
		log.Fatalf("Service run failed: %v", err)
	}
}
