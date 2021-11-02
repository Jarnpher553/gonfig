package vars

import (
	"github.com/Jarnpher553/gonfig/internal/server/event"
	"github.com/Jarnpher553/gonfig/internal/server/store"
	"sync"
)

type ServiceCtx struct {
	Meta    *ServerMetadata
	Slaves  []*ServerMetadata
	Store   store.Store
	Mux     *sync.Mutex
	Trigger event.Trigger
}
