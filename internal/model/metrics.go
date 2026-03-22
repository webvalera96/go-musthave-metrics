package models

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// Metrics is a single gauge or counter metric (ID, type, value/delta).
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
