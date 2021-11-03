package handler

import (
	"encoding/json"
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/server/event"
	"github.com/Jarnpher553/gonfig/internal/server/types"
	"github.com/Jarnpher553/gonfig/internal/util/color"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type HandlerFunc func(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request)

// RegisterSlaveHandler 注册从节点
func RegisterSlaveHandler(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		var meta types.SlaveMetaReq
		err = json.Unmarshal(body, &meta)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		slave := types.ServerMetadata{
			ID:    meta.ID,
			Role:  types.Role(meta.Role),
			RAddr: meta.Addr,
		}

		err = s.Store.Put([]byte(fmt.Sprintf(types.SlaveFormat, slave.ID)), []byte(fmt.Sprintf("%s", slave.RAddr)))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		s.Mux.Lock()
		defer s.Mux.Unlock()
		*s.Slaves = append(*s.Slaves, &slave)

		s.Logger.Info("Slave id:[%s] addr:[%s] online", color.Green(slave.ID), color.Green(slave.RAddr))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	}
}

// UnregisterSlaveHandler 注销从节点
func UnregisterSlaveHandler(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		var meta types.SlaveMetaReq
		err = json.Unmarshal(body, &meta)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		s.Mux.Lock()
		defer s.Mux.Unlock()
		index := -1
		for idx, slave := range *s.Slaves {
			if slave.ID == meta.ID {
				index = idx
				break
			}
		}

		if index >= 0 {
			err := s.Store.Delete([]byte(fmt.Sprintf(types.SlaveFormat, meta.ID)))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
				return
			}
			if index == len(*s.Slaves)-1 {
				*s.Slaves = (*s.Slaves)[:index]
			} else {
				*s.Slaves = append((*s.Slaves)[:index], (*s.Slaves)[index+1:]...)
			}

			s.Logger.Info("Slave id:[%s] addr:[%s] offline", color.Green(meta.ID), color.Green(meta.Addr))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	}
}

// SyncConfigurationHandler 主从同步配置
func SyncConfigurationHandler(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		var req types.SyncConfigReq
		err = json.Unmarshal(body, &req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}
		for _, c := range req.Datum {
			err = s.Store.Put([]byte(fmt.Sprintf(types.ConfigFormat, c.Name, strings.Join(c.Tag, "#"))), []byte(c.Body))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
				return
			}
			log.Printf("Slave config name:[%s] tag:[%s] sync success", color.Green(c.Name), color.Green(c.Tag))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	}
}

// PushConfigHandler 推送配置
func PushConfigHandler(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		var c types.PushConfigReq
		err = json.Unmarshal(body, &c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		cfgName := fmt.Sprintf(types.ConfigFormat, c.Name, strings.Join(c.Tag, "#"))
		err = s.Store.Put([]byte(cfgName), []byte(c.Body))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))

		s.Trigger.Emit(&event.Event{Type: event.PubConfig, Body: map[string]interface{}{"cfgName": cfgName, "cfgMeta": c.Body}})
		s.Trigger.Emit(&event.Event{Type: event.SyncConfig})
	}
}

// PullConfigHandler 拉去配置
func PullConfigHandler(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		var c types.PullConfigReq
		err = json.Unmarshal(body, &c)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		v, err := s.Store.Get([]byte(fmt.Sprintf("%s#%s", c.Name, strings.Join(c.Tag, "#"))))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		var resp types.PullConfigResp
		resp.Body = string(v)
		respBytes, err := json.Marshal(&resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(respBytes)
	}
}

func HealthHandler(s *types.ServiceCtx, method string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(http.StatusText(http.StatusMethodNotAllowed)))
			return
		}

		ids := r.URL.Query()["id"]
		id := ids[0]
		if id == s.Meta.ID.String() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(http.StatusText(http.StatusOK)))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(http.StatusText(http.StatusInternalServerError)))
		}
	}
}
