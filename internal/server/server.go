package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/logger"
	"github.com/Jarnpher553/gonfig/internal/server/cmdflag"
	"github.com/Jarnpher553/gonfig/internal/server/event"
	"github.com/Jarnpher553/gonfig/internal/server/handler"
	"github.com/Jarnpher553/gonfig/internal/server/listener"
	"github.com/Jarnpher553/gonfig/internal/server/route"
	"github.com/Jarnpher553/gonfig/internal/server/rpchandler"
	"github.com/Jarnpher553/gonfig/internal/server/types"
	"github.com/Jarnpher553/gonfig/internal/store"
	"github.com/Jarnpher553/gonfig/internal/util/color"
	"github.com/Jarnpher553/gonfig/internal/util/retry"
	"github.com/common-nighthawk/go-figure"
	"github.com/lesismal/arpc/extension/pubsub"
	alog "github.com/lesismal/arpc/log"
	"github.com/satori/go.uuid"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type eventHandler func(map[string]interface{}) error

//Server gonfig server
type Server struct {
	meta          *types.ServerMetadata
	httpServer    *http.Server
	psServer      *pubsub.Server
	mux           *sync.Mutex
	slaves        *types.Slaves
	store         store.Store
	listener      *listener.Listener
	rpcListener   *listener.Listener
	masterAddr    string
	serverMux     *http.ServeMux
	httpRouters   []*route.Router
	rpcRouters    []string
	trigger       event.Trigger
	eventHandlers map[string]eventHandler
	logger        *logger.XLogger
	rpcLogger     *logger.XLogger
}

//New construct Server
func New(args ...interface{}) *Server {
	logx := &logger.XLogger{}

	var cfg *types.ServerCfg
	var persist store.Store
	for _, arg := range args {
		switch v := arg.(type) {
		case *types.ServerCfg:
			if v != nil {
				cfg = v
			}
		case store.Store:
			if v != nil {
				persist = v
			}
		}
	}
	if cfg == nil {
		cfg = cmdflag.Parse(logx)
	}
	if persist == nil {
		leveldbStore, err := store.NewLeveldbStore(store.StorageFile)
		if err != nil {
			logx.Fatal("Leveldb open: %s", err)
		}
		persist = leveldbStore
	}

	logx.SetLevel(alog.LevelInfo)
	logx.SetModName("HttpServer")

	start := strings.Index(cfg.Addr, ":")
	s := &Server{
		meta: &types.ServerMetadata{
			ID:    uuid.NewV4(),
			Role:  cfg.Role,
			RAddr: cfg.Addr,
			LAddr: cfg.Addr[start:],
		},
		store:       persist,
		trigger:     make(chan *event.Event, 5),
		logger:      logx,
		httpRouters: make([]*route.Router, 0),
		rpcRouters:  make([]string, 0),
	}
	s.eventHandlers = map[string]eventHandler{
		event.SyncConfig: s.eventSyncHandler,
		event.PubConfig:  s.eventPubHandler,
	}

	if s.meta.Role == types.RoleMaster {
		s.mux = &sync.Mutex{}
		slaves := make([]*types.ServerMetadata, 0)
		s.slaves = &slaves
	} else if s.meta.Role == types.RoleSlave {
		s.masterAddr = cfg.MasterAddr
	}

	logx2 := &logger.XLogger{}
	logx2.SetLevel(alog.LevelInfo)
	alog.SetLogger(logx2)
	s.rpcLogger = logx2
	psServer := pubsub.NewServer()
	psServer.Password = types.PubSubPassword
	psServer.Handler.SetLogTag("[" + color.Green("RpcServer") + "]")
	s.psServer = psServer

	s.rpcRoute("/echo", rpchandler.EchoHandler)

	serverMux := http.NewServeMux()
	s.httpServer = &http.Server{
		Addr:           s.meta.LAddr,
		Handler:        serverMux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.serverMux = serverMux

	if s.meta.Role == types.RoleMaster {
		s.httpRoute(route.NewRouter(http.MethodPost, "/register"), handler.RegisterSlaveHandler)
		s.httpRoute(route.NewRouter(http.MethodPost, "/unregister"), handler.UnregisterSlaveHandler)
		s.httpRoute(route.NewRouter(http.MethodPost, "/push"), handler.PushConfigHandler)
	} else if s.meta.Role == types.RoleSlave {
		s.httpRoute(route.NewRouter(http.MethodPost, "/sync"), handler.SyncConfigurationHandler)
		s.httpRoute(route.NewRouter(http.MethodGet, "/health"), handler.HealthHandler)
	}

	s.httpRoute(route.NewRouter(http.MethodPost, "/pull"), handler.PullConfigHandler)

	return s
}

func (s *Server) rpcRoute(methodName string, handlerFunc rpchandler.RpcHandlerFunc) {
	s.rpcRouters = append(s.rpcRouters, methodName)
	serviceCtx := &types.ServiceCtx{
		Meta:    s.meta,
		Slaves:  s.slaves,
		Store:   s.store,
		Mux:     s.mux,
		Trigger: s.trigger,
		Logger:  s.logger,
	}
	s.psServer.Handler.Handle(methodName, handlerFunc(serviceCtx))
}

func (s *Server) httpRoute(r *route.Router, handlerFunc handler.HandlerFunc) {
	s.httpRouters = append(s.httpRouters, r)
	serviceCtx := &types.ServiceCtx{
		Meta:    s.meta,
		Slaves:  s.slaves,
		Store:   s.store,
		Mux:     s.mux,
		Trigger: s.trigger,
		Logger:  s.logger,
	}
	s.serverMux.HandleFunc(r.URL, s.recovery(handlerFunc(serviceCtx, r.Method)))
}

func (s *Server) recovery(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("Recover: %s", err)
			}
		}()
		f(w, r)
	}
}

func (s *Server) printRoutes() {
	for i, r := range s.httpRouters {
		s.logger.Info("HttpRouter [%s] Method [%s] Path [%s]", color.Green(i), color.Green(r.Method), color.Green(r.URL))
	}
	for i, v := range s.rpcRouters {
		s.rpcLogger.Info("%s RpcRouter [%s] Method [%s]", s.psServer.Handler.LogTag(), color.Green(i), color.Green(v))
	}
}

//Serve run server
func (s *Server) Serve() {
	ln, err := listener.New(s.meta.LAddr)
	if err != nil {
		s.logger.Fatal("Create listener: %s", err)
	}
	s.listener = ln

	figure.NewColorFigure(string(s.meta.Role), "", "green", true).Print()
	s.printRoutes()

	s.logger.Info("[%s]/[%s] listening on [%s]", color.Green(s.meta.Role), color.Green(s.meta.ID), color.Green(s.meta.LAddr))
	go func() {
		if err := s.httpServer.Serve(ln); err != nil {
			s.logger.Info("Listen: %s", err)
		}
	}()

	rpcPort, _ := strconv.Atoi(strings.Split(s.meta.LAddr, ":")[1])
	rpcLn, err := listener.New(fmt.Sprintf(":%d", rpcPort+1))
	if err != nil {
		s.logger.Fatal("Create rpc listener: %s", err)
	}
	s.rpcListener = rpcLn
	go func() {
		if err := s.psServer.Serve(rpcLn); err != nil {
			s.rpcLogger.Info("%s Listen: %s", s.psServer.Handler.LogTag(), err)
		}
	}()

	if s.meta.Role == types.RoleSlave {
		err := s.register()
		if err != nil {
			s.logger.Fatal("Register slave: %s", err)
		}
	} else {
		s.loadSlaves()
		go s.execEvent()
		go s.healthCheck()
	}

	notify := make(chan os.Signal)
	signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)

	<-notify

	if s.meta.Role == types.RoleSlave {
		err := s.unregister()
		if err != nil {
			s.logger.Info("Unregister slave: %s", err)
		}
	}

	s.logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Fatal("Forced to shutdown: %s", err)
	}

	s.rpcLogger.Info("%s Shutting down server...", s.psServer.Handler.LogTag())
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := s.psServer.Shutdown(ctx2); err != nil {
		s.rpcLogger.Fatal("%s Forced to shutdown: %s", s.psServer.Handler.LogTag(), err)
	}

	s.logger.Info("Server exiting")
	s.rpcLogger.Info("%s Server exiting", s.psServer.Handler.LogTag())
}

func (s *Server) register() error {
	url := fmt.Sprintf("http://%s/register", s.masterAddr)

	self := &types.SlaveMetaReq{
		ID:   s.meta.ID,
		Addr: s.meta.RAddr,
		Role: string(s.meta.Role),
	}
	jsonBytes, err := json.Marshal(self)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application:json", strings.NewReader(string(jsonBytes)))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("register failure")
	}

	return nil
}

func (s *Server) unregister() error {
	url := fmt.Sprintf("http://%s/unregister", s.masterAddr)

	self := &types.SlaveMetaReq{
		ID:   s.meta.ID,
		Addr: s.meta.RAddr,
		Role: string(s.meta.RAddr),
	}
	jsonBytes, err := json.Marshal(self)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application:json", strings.NewReader(string(jsonBytes)))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("unregister failure")
	}

	return nil
}

func (s *Server) healthCheck() {
	duration := time.Second * 10
	t := time.NewTimer(duration)
	for {
		t.Reset(duration)
		<-t.C
		g := sync.WaitGroup{}
		g.Add(len(*s.slaves))

		var checkFailureCount int
		ids := make([]uuid.UUID, 0)
		for idx, slave := range *s.slaves {
			go func(i int, sl *types.ServerMetadata) {
				defer g.Done()
				url := fmt.Sprintf("http://%s/health?id=%s", sl.RAddr, sl.ID)
				var resp *http.Response
				var err error
				err = retry.Retry(3, func() error {
					resp, err = http.Get(url)
					if err != nil {
						return err
					}
					return nil
				}, 1*time.Second)
				if err != nil || resp.StatusCode != 200 {
					s.logger.Info("Slave id:[%s] addr:[%s] health check failure", color.Green(sl.ID), color.Green(sl.RAddr))
					ids = append(ids, sl.ID)
					sl.ID = uuid.Nil
					checkFailureCount++
				}
			}(idx, slave)
		}
		g.Wait()

		if checkFailureCount != 0 {
			for _, id := range ids {
				key := fmt.Sprintf(types.SlaveFormat, id.String())
				_ = s.store.Delete([]byte(key))
			}

			s.mux.Lock()
			slaves := make([]*types.ServerMetadata, 0, len(*s.slaves))
			for _, slave := range *s.slaves {
				if slave.ID != uuid.Nil {
					slaves = append(slaves, slave)
				}
			}
			*s.slaves = slaves
			s.mux.Unlock()
		}

		s.printSlaves()
	}
}

func (s *Server) printSlaves() {
	echo := "Slaves ["
	for _, slave := range *s.slaves {
		echo += fmt.Sprintf("{ Slave id:[%s] addr:[%s] } ", color.Green(slave.ID), color.Green(slave.RAddr))
	}
	strings.TrimSuffix(echo, " ")
	echo += "]"

	s.logger.Info(echo)
}

func (s *Server) loadSlaves() {
	pairs, err := s.store.Items("slave/")
	if err != nil {
		s.logger.Info("Load Slave: %s", err)
	}
	for _, kv := range pairs {
		k := string(kv.Key)
		v := string(kv.Value)

		id, _ := uuid.FromString(strings.Split(k, "/")[1])
		*s.slaves = append(*s.slaves, &types.ServerMetadata{
			ID:    id,
			Role:  types.RoleSlave,
			RAddr: v,
		})
	}
}

func (s *Server) execEvent() {
	for ev := range s.trigger.C() {
		_ = s.eventHandlers[ev.Type](ev.Body)
	}
}

func (s *Server) eventSyncHandler(param map[string]interface{}) error {
	g := sync.WaitGroup{}
	g.Add(len(*s.slaves))
	for _, slave := range *s.slaves {
		go func(sl *types.ServerMetadata) {
			defer g.Done()
			err := retry.Retry(3, func() error {
				url := fmt.Sprintf("http://%s/sync", sl.RAddr)

				req := &types.SyncConfigReq{
					Datum: make([]*types.PushConfigReq, 0),
				}
				pairs, err := s.store.Items("config/")
				if err != nil {
					return err
				}
				for _, kv := range pairs {
					var pr types.PushConfigReq
					sKey := string(kv.Key)
					splitKey := strings.Split(sKey, "#")
					reg := regexp.MustCompile("config/(.*)/")
					pr.Name = reg.FindStringSubmatch(splitKey[0])[1]
					for _, v := range splitKey[1:] {
						pr.Tag = append(pr.Tag, v)
					}
					pr.Body = string(kv.Value)
					req.Datum = append(req.Datum, &pr)
				}

				jsonBytes, err := json.Marshal(req)
				if err != nil {
					return err
				}

				resp, err := http.Post(url, "application:json", strings.NewReader(string(jsonBytes)))
				if err != nil {
					return err
				}

				if resp.StatusCode != 200 {
					return errors.New("register failure")
				}
				return nil
			})

			if err != nil {
				s.logger.Info("Slave id:[%s] addr:[%s] sync error: %s", color.Green(sl.ID), color.Green(sl.RAddr), err)
				return
			}
			s.logger.Info("Slave id:[%s] addr:[%s] sync success", color.Green(sl.ID), color.Green(sl.RAddr))
		}(slave)
	}
	g.Wait()
	return nil
}

func (s *Server) eventPubHandler(param map[string]interface{}) error {
	err := s.psServer.Publish(param["cfgName"].(string), param["cfgMeta"])
	if err != nil {
		s.logger.Info("Publish [%s] failure: %s", param["cfgName"], err)
		return err
	}

	return nil
}
