package audit

// Event — событие аудита в формате для логирования.
type Event struct {
	TS        int64    `json:"ts"`         // unix timestamp события
	Metrics   []string `json:"metrics"`   // наименования полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}
