package event

const (
	SyncConfig = "SyncConfig"
	PubConfig  = "PubConfig"
)

type Event struct {
	Type string
	Body map[string]interface{}
}

type Trigger chan *Event

func (t Trigger) Emit(event *Event) {
	t <- event
}

func (t Trigger) C() chan *Event {
	return t
}
