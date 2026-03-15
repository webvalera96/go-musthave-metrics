package audit

import (
	"encoding/json"
	"os"
	"sync"
)

// FileReceiver appends audit events as JSON lines to a file.
type FileReceiver struct {
	path string
	mu   sync.Mutex
}

// NewFileReceiver returns a receiver that writes to the given path.
func NewFileReceiver(path string) *FileReceiver {
	return &FileReceiver{path: path}
}

// Notify добавляет событие в конец файла на новой строке.
func (f *FileReceiver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	line := append(data, '\n')

	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(line)
	return err
}
