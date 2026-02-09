package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPReceiver sends audit events via POST to a URL.
type HTTPReceiver struct {
	url    string
	client *http.Client
}

// NewHTTPReceiver returns a receiver that POSTs to the given URL.
func NewHTTPReceiver(url string) *HTTPReceiver {
	return &HTTPReceiver{
		url:    url,
		client: &http.Client{},
	}
}

// Notify отправляет событие методом POST по сконфигурированному URL.
func (h *HTTPReceiver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, h.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &HTTPAuditError{StatusCode: resp.StatusCode}
	}
	return nil
}

// HTTPAuditError — ошибка при отправке аудита по HTTP (неуспешный статус).
type HTTPAuditError struct {
	StatusCode int
}

func (e *HTTPAuditError) Error() string {
	return fmt.Sprintf("audit HTTP request failed with status: %d", e.StatusCode)
}
