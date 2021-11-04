package types

import (
	"github.com/Jarnpher553/gonfig/internal/logger"
	"github.com/Jarnpher553/gonfig/internal/server/event"
	"github.com/Jarnpher553/gonfig/internal/store"
	"sync"
)

//ServiceCtx server context used by http handler
type ServiceCtx struct {
	Meta    *ServerMetadata
	Slaves  *Slaves
	Store   store.Store
	Mux     *sync.Mutex
	Trigger event.Trigger
	Logger  *logger.XLogger
}
