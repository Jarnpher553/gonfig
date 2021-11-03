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
	"github.com/Jarnpher553/gonfig/internal/server/store"
	"github.com/Jarnpher553/gonfig/internal/server/types"
	"github.com/Jarnpher553/gonfig/internal/utility/color"
	"github.com/Jarnpher553/gonfig/internal/utility/retry"
	"github.com/common-nighthawk/go-figure"
	"github.com/lesismal/arpc"
	"github.com/lesismal/arpc/extension/pubsub"
	alog "github.com/lesismal/arpc/log"
	"github.com/satori/go.uuid"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
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

type EventHandler func(map[string]interface{}) error

type Server struct {
	meta          *types.ServerMetadata
	httpServer    *http.Server
	psServer      *pubsub.Server
	mux           *sync.Mutex
	slaves        []*types.ServerMetadata
	store         store.Store
	listener      *listener.Listener
	rpcListener   *listener.Listener
	masterAddr    string
	serverMux     *http.ServeMux
	routes        []*route.Router
	trigger       event.Trigger
	eventHandlers map[string]EventHandler
	logger        *logger.XLogger
}

func New() *Server {
	logx := &logger.XLogger{}

	cfg := cmdflag.Parse(logx)

	logx.SetLevel(alog.LevelInfo)

	db, err := leveldb.OpenFile("./db", nil)
	if err != nil {
		logx.Fatal("Leveldb open: %s", err)
	}
	start := strings.Index(cfg.Addr, ":")
	s := &Server{
		meta: &types.ServerMetadata{
			ID:    uuid.NewV4(),
			Role:  cfg.Role,
			RAddr: cfg.Addr,
			LAddr: cfg.Addr[start:],
		},
		store:   &store.LeveldbStore{DB: db},
		trigger: make(chan *event.Event, 5),
		logger:  logx,
	}
	s.eventHandlers = map[string]EventHandler{
		event.SyncConfig: s.eventSyncHandler,
		event.PubConfig:  s.eventPubHandler,
	}

	if s.meta.Role == types.RoleMaster {
		s.mux = &sync.Mutex{}
		s.slaves = make([]*types.ServerMetadata, 0)
	} else if s.meta.Role == types.RoleSlave {
		s.masterAddr = cfg.MasterAddr
	}

	alog.SetLogger(logx)
	psServer := pubsub.NewServer()
	psServer.Password = types.PubSubPassword
	psServer.Handler.SetLogTag("[" + color.Green("PubSub") + "]")
	psServer.Handler.Handle("/echo", func(c *arpc.Context) {
		var in types.ConfigMeta
		if err := c.Bind(&in); err != nil {
			return
		}
		cfgName := fmt.Sprintf(types.ConfigFormat, in.Name, strings.Join(in.Tags, "#"))
		v, err := s.store.Get([]byte(cfgName))
		if err != nil {
			return
		}

		c.Client.Set("config_meta", &in)
		c.Write(v)
	})

	s.psServer = psServer

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
		s.route(route.NewRouter(http.MethodPost, "/register"), handler.RegisterSlaveHandler)
		s.route(route.NewRouter(http.MethodPost, "/unregister"), handler.UnregisterSlaveHandler)
		s.route(route.NewRouter(http.MethodPost, "/push"), handler.PushConfigHandler)
	} else if s.meta.Role == types.RoleSlave {
		s.route(route.NewRouter(http.MethodPost, "/sync"), handler.SyncConfigurationHandler)
		s.route(route.NewRouter(http.MethodGet, "/health"), handler.HealthHandler)
	}

	s.route(route.NewRouter(http.MethodPost, "/pull"), handler.PullConfigHandler)

	return s
}

func (s *Server) route(r *route.Router, handlerFunc handler.HandlerFunc) {
	s.routes = append(s.routes, r)
	s.serverMux.HandleFunc(r.URL, s.recovery(handlerFunc(&types.ServiceCtx{
		Meta:    s.meta,
		Slaves:  s.slaves,
		Store:   s.store,
		Mux:     s.mux,
		Trigger: s.trigger,
		Logger:  s.logger,
	}, r.Method)))
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
	for i, r := range s.routes {
		s.logger.Info("Route [%s] Method [%s] Path [%s]", color.Green(i), color.Green(r.Method), color.Green(r.URL))
	}
}

func (s *Server) Serve() {
	ln, err := listener.NewListener(s.meta.LAddr)
	if err != nil {
		s.logger.Fatal("Create listener: %s", err)
	}
	s.listener = ln

	figure.NewColorFigure(string(s.meta.Role), "", "green", true).Print()
	s.logger.Info("Server [%s]/[%s] listening on [%s]", color.Green(s.meta.Role), color.Green(s.meta.ID), color.Green(s.meta.LAddr))

	s.printRoutes()

	go func() {
		if err := s.httpServer.Serve(ln); err != nil {
			s.logger.Info("Listen: %s", err)
		}
	}()

	rpcPort, _ := strconv.Atoi(strings.Split(s.meta.LAddr, ":")[1])
	rpcLn, err := listener.NewListener(fmt.Sprintf(":%d", rpcPort+1))
	if err != nil {
		s.logger.Fatal("Create rpc listener: %s", err)
	}
	s.rpcListener = rpcLn
	go func() {
		if err := s.psServer.Serve(rpcLn); err != nil {
			s.logger.Info("PubSub Listen: %s", err)
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
		s.logger.Fatal("Server forced to shutdown: %s", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := s.psServer.Shutdown(ctx2); err != nil {
		s.logger.Fatal("PubSub Server forced to shutdown: %s", err)
	}

	s.logger.Info("Server exiting.")
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
	duration := time.Second * 5
	t := time.NewTimer(duration)
	for {
		t.Reset(duration)
		<-t.C
		g := sync.WaitGroup{}
		g.Add(len(s.slaves))

		var checkFailureCount int
		ids := make([]uuid.UUID, 0)
		for idx, slave := range s.slaves {
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
				_ = s.store.Delete([]byte(fmt.Sprintf(types.SlaveFormat, id)))
			}

			s.mux.Lock()
			slaves := make([]*types.ServerMetadata, 0, len(s.slaves))
			for _, slave := range s.slaves {
				if slave.ID != uuid.Nil {
					slaves = append(slaves, slave)
				}
			}
			s.slaves = slaves
			s.mux.Unlock()
		}

		s.printSlaves()
	}
}

func (s *Server) printSlaves() {
	echo := "Slaves print [\n"
	for _, slave := range s.slaves {
		echo += fmt.Sprintf("\tSlave id:[%s] addr:[%s]\n", color.Green(slave.ID), color.Green(slave.RAddr))
	}
	echo += "]"

	s.logger.Info(echo)
}

func (s *Server) loadSlaves() {
	pairs, err := s.store.Items(util.BytesPrefix([]byte("slave/")))
	if err != nil {
		s.logger.Info("Load Slave: %s", err)
	}
	for _, kv := range pairs {
		k := string(kv.Key)
		v := string(kv.Value)

		id, _ := uuid.FromString(strings.Split(k, "/")[1])
		s.slaves = append(s.slaves, &types.ServerMetadata{
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
	g.Add(len(s.slaves))
	for _, slave := range s.slaves {
		go func(sl *types.ServerMetadata) {
			defer g.Done()
			err := retry.Retry(3, func() error {
				url := fmt.Sprintf("http://%s/sync", sl.RAddr)

				req := &types.SyncConfigReq{
					Datum: make([]*types.PushConfigReq, 0),
				}
				pairs, err := s.store.Items(util.BytesPrefix([]byte("config/")))
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

func (s *Server) pubConfig(topic string) error {
	v, err := s.store.Get([]byte(topic))
	if err != nil {
		return err
	}

	err = s.psServer.Publish(topic, v)
	if err != nil {
		return err
	}
	return nil
}
