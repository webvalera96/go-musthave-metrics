package audit

// Receiver — приёмник событий аудита (наблюдатель).
type Receiver interface {
	// Notify отправляет событие аудита в приёмник.
	Notify(event Event) error
}

// Subject — субъект, уведомляющий зарегистрированных наблюдателей о событиях аудита.
type Subject struct {
	receivers []Receiver
}

// NewSubject создаёт новый субъект без приёмников.
func NewSubject() *Subject {
	return &Subject{
		receivers: nil,
	}
}

// Attach добавляет приёмник аудита.
func (s *Subject) Attach(r Receiver) {
	if r == nil {
		return
	}
	s.receivers = append(s.receivers, r)
}

// NotifyAll отправляет событие во все зарегистрированные приёмники.
// Вызывается после успешной обработки метрик. Если приёмников нет, ничего не делает.
func (s *Subject) NotifyAll(event Event) {
	for _, r := range s.receivers {
		_ = r.Notify(event)
	}
}

// NewSubjectFromConfig создаёт субъект и подключает приёмники по путям/URL.
// auditFile — путь к файлу для логов (если пусто, файловый приёмник не добавляется).
// auditURL — URL для отправки логов (если пусто, HTTP-приёмник не добавляется).
func NewSubjectFromConfig(auditFile, auditURL string) *Subject {
	s := NewSubject()
	if auditFile != "" {
		s.Attach(NewFileReceiver(auditFile))
	}
	if auditURL != "" {
		s.Attach(NewHTTPReceiver(auditURL))
	}
	return s
}
