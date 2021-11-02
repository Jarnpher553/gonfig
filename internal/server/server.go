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
	"github.com/Jarnpher553/gonfig/internal/server/vars"
	"github.com/Jarnpher553/gonfig/internal/types"
	"github.com/Jarnpher553/gonfig/internal/utility/color"
	"github.com/Jarnpher553/gonfig/internal/utility/retry"
	"github.com/common-nighthawk/go-figure"
	"github.com/lesismal/arpc"
	"github.com/lesismal/arpc/extension/pubsub"
	alog "github.com/lesismal/arpc/log"
	"github.com/satori/go.uuid"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"log"
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

const (
	Password    = "U2FsdGVkX18Wi+guhMZL8DvCOqfA6j/MWMdUOv9tOvQ="
	TopicConfig = "config/%s/#%s"
)

type EventHandler func(map[string]interface{}) error

type Server struct {
	meta          *vars.ServerMetadata
	httpServer    *http.Server
	psServer      *pubsub.Server
	mux           *sync.Mutex
	slaves        []*vars.ServerMetadata
	store         store.Store
	listener      *listener.Listener
	rpcListener   *listener.Listener
	masterAddr    string
	serverMux     *http.ServeMux
	routes        []*route.Router
	trigger       event.Trigger
	eventHandlers map[string]EventHandler
}

func New() *Server {
	cfg := cmdflag.Parse()

	db, err := leveldb.OpenFile("./db", nil)
	if err != nil {
		log.Fatal("Leveldb open:", err)
	}
	start := strings.Index(cfg.Addr, ":")
	s := &Server{
		meta: &vars.ServerMetadata{
			ID:    uuid.NewV4(),
			Role:  cfg.Role,
			RAddr: cfg.Addr,
			LAddr: cfg.Addr[start:],
		},
		store:   &store.LeveldbStore{DB: db},
		trigger: make(chan *event.Event, 5),
	}
	s.eventHandlers = map[string]EventHandler{
		event.SyncConfig: s.eventSyncHandler,
		event.PubConfig:  s.eventPubHandler,
	}

	if s.meta.Role == vars.RoleMaster {
		s.mux = &sync.Mutex{}
		s.slaves = make([]*vars.ServerMetadata, 0)
	} else if s.meta.Role == vars.RoleSlave {
		s.masterAddr = cfg.MasterAddr
	}

	lgx := &logger.XLogger{}
	lgx.SetLevel(alog.LevelInfo)
	alog.SetLogger(lgx)
	psServer := pubsub.NewServer()
	psServer.Password = Password
	psServer.Handler.SetLogTag("[" + color.Green("PubSub") + "]")
	psServer.Handler.Handle("/echo", func(c *arpc.Context) {
		var in types.ConfigMeta
		if err := c.Bind(&in); err != nil {
			return
		}
		cfgName := fmt.Sprintf(TopicConfig, in.Name, strings.Join(in.Tag, "#"))
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

	if s.meta.Role == vars.RoleMaster {
		s.route(route.NewRouter(http.MethodPost, "/register"), handler.RegisterSlaveHandler)
		s.route(route.NewRouter(http.MethodPost, "/unregister"), handler.UnregisterSlaveHandler)
		s.route(route.NewRouter(http.MethodPost, "/push"), handler.PushConfigHandler)
	} else if s.meta.Role == vars.RoleSlave {
		s.route(route.NewRouter(http.MethodPost, "/sync"), handler.SyncConfigurationHandler)
		s.route(route.NewRouter(http.MethodGet, "/health"), handler.HealthHandler)
	}

	s.route(route.NewRouter(http.MethodPost, "/pull"), handler.PullConfigHandler)

	return s
}

func (s *Server) route(r *route.Router, handlerFunc handler.HandlerFunc) {
	s.routes = append(s.routes, r)
	s.serverMux.HandleFunc(r.URL, recovery(handlerFunc(&vars.ServiceCtx{
		Meta:    s.meta,
		Slaves:  s.slaves,
		Store:   s.store,
		Mux:     s.mux,
		Trigger: s.trigger,
	}, r.Method)))
}

func recovery(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Recover:", err)
			}
		}()
		f(w, r)
	}
}

func (s *Server) printRoutes() {
	for i, r := range s.routes {
		log.Printf("Route [%s] Method [%s] Path [%s]", color.Green(i), color.Green(r.Method), color.Green(r.URL))
	}
}

func (s *Server) Serve() {
	ln, err := listener.NewListener(s.meta.LAddr)
	if err != nil {
		log.Fatal("Create listener:", err)
	}
	s.listener = ln

	figure.NewColorFigure(string(s.meta.Role), "", "green", true).Print()
	log.Printf("Server [%s]/[%s] listening on [%s]", color.Green(s.meta.Role), color.Green(s.meta.ID), color.Green(s.meta.LAddr))

	s.printRoutes()

	go func() {
		if err := s.httpServer.Serve(ln); err != nil {
			log.Println("Listen:", err)
		}
	}()

	rpcPort, _ := strconv.Atoi(strings.Split(s.meta.LAddr, ":")[1])
	rpcLn, err := listener.NewListener(fmt.Sprintf(":%d", rpcPort+1))
	if err != nil {
		log.Fatalln("Create rpc listener:", err)
	}
	s.rpcListener = rpcLn
	go func() {
		if err := s.psServer.Serve(rpcLn); err != nil {
			log.Println("PubSub Listen:", err)
		}
	}()

	if s.meta.Role == vars.RoleSlave {
		err := s.register()
		if err != nil {
			log.Fatalln("Register slave:", err)
		}
	} else {
		s.loadSlaves()
		go s.execEvent()
		go s.healthCheck()
	}

	notify := make(chan os.Signal)
	signal.Notify(notify, syscall.SIGINT, syscall.SIGTERM)

	<-notify

	if s.meta.Role == vars.RoleSlave {
		err := s.unregister()
		if err != nil {
			log.Println("Unregister slave:", err)
		}
	}

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := s.psServer.Shutdown(ctx2); err != nil {
		log.Fatal("PubSub Server forced to shutdown: ", err)
	}

	log.Println("Server exiting.")
}

func (s *Server) register() error {
	url := fmt.Sprintf("http://%s/register", s.masterAddr)

	self := &vars.SlaveMetaReq{
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

	self := &vars.SlaveMetaReq{
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
			go func(i int, sl *vars.ServerMetadata) {
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
					log.Printf("Slave id:[%s] addr:[%s] health check failure", color.Green(sl.ID), color.Green(sl.RAddr))
					ids = append(ids, sl.ID)
					sl.ID = uuid.Nil
					checkFailureCount++
				}
			}(idx, slave)
		}
		g.Wait()

		if checkFailureCount != 0 {
			for _, id := range ids {
				_ = s.store.Delete([]byte(fmt.Sprintf("slave/%s", id)))
			}

			s.mux.Lock()
			slaves := make([]*vars.ServerMetadata, 0, len(s.slaves))
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

	log.Println(echo)
}

func (s *Server) loadSlaves() {
	pairs, err := s.store.Items(util.BytesPrefix([]byte("slave/")))
	if err != nil {
		log.Fatalln("Load Slave:", err)
	}
	for _, kv := range pairs {
		k := string(kv.Key)
		v := string(kv.Value)

		id, _ := uuid.FromString(strings.Split(k, "/")[1])
		s.slaves = append(s.slaves, &vars.ServerMetadata{
			ID:    id,
			Role:  vars.RoleSlave,
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
		go func(sl *vars.ServerMetadata) {
			defer g.Done()
			err := retry.Retry(3, func() error {
				url := fmt.Sprintf("http://%s/sync", sl.RAddr)

				req := &vars.SyncConfigReq{
					Datum: make([]*vars.PushConfigReq, 0),
				}
				pairs, err := s.store.Items(util.BytesPrefix([]byte("config/")))
				for _, kv := range pairs {
					var pr vars.PushConfigReq
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
				log.Printf("Slave id:[%s] addr:[%s] sync error: %s", color.Green(sl.ID), color.Green(sl.RAddr), err)
				return
			}
			log.Printf("Slave id:[%s] addr:[%s] sync success", color.Green(sl.ID), color.Green(sl.RAddr))
		}(slave)
	}
	g.Wait()
	return nil
}

func (s *Server) eventPubHandler(param map[string]interface{}) error {
	err := s.psServer.Publish(param["cfgName"].(string), param["cfgMeta"])
	if err != nil {
		log.Printf("Publish [%s] failure: %s", param["cfgName"], err)
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
