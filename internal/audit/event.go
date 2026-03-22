package audit

// Event is an audit log entry (timestamp, metric names, client IP).
type Event struct {
	TS        int64    `json:"ts"`         // unix timestamp события
	Metrics   []string `json:"metrics"`   // наименования полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}
