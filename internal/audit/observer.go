package audit

// Receiver receives audit events (observer).
type Receiver interface {
	Notify(event Event) error
}

// Subject notifies attached receivers on audit events.
type Subject struct {
	receivers []Receiver
}

// NewSubject returns a subject with no receivers.
func NewSubject() *Subject {
	return &Subject{
		receivers: nil,
	}
}

// Attach adds an audit receiver.
func (s *Subject) Attach(r Receiver) {
	if r == nil {
		return
	}
	s.receivers = append(s.receivers, r)
}

// NotifyAll sends the event to all attached receivers.
func (s *Subject) NotifyAll(event Event) {
	for _, r := range s.receivers {
		_ = r.Notify(event)
	}
}

// NewSubjectFromConfig builds a subject with file and/or HTTP receivers if paths are set.
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
