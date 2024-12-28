package credentialparser

import (
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	Directory string
	OnChange  func(string) // Callback function to handle new files
}

func NewFileWatcher(directory string, onChange func(string)) *FileWatcher {
	return &FileWatcher{
		Directory: directory,
		OnChange:  onChange,
	}
}

func (fw *FileWatcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(fw.Directory)
	if err != nil {
		return err
	}

	log.Printf("Watching directory: %s", fw.Directory)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// Check if a new file is created
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("New file detected: %s", event.Name)
				if filepath.Ext(event.Name) == ".txt" {
					go fw.OnChange(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %s", err)
		}
	}
}
