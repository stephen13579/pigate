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

			log.Printf("fsnotify event: %s %v", event.Name, event.Op)

			// Trigger on new .txt files *or* changes to existing ones
			if filepath.Ext(event.Name) == ".txt" &&
				(event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0) {

				log.Printf("txt file changed: %s (op=%v)", event.Name, event.Op)
				go fw.OnChange(event.Name)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("Watcher error: %s", err)
		}
	}
}
