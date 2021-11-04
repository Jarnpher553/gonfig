package server

import (
	"github.com/Jarnpher553/gonfig/internal/server/types"
	"github.com/Jarnpher553/gonfig/internal/store"
	"testing"
)

func TestNew(t *testing.T) {
	leveldbStore, _ := store.NewLeveldbStore(store.StorageMem)
	t.Run("NewMaster_1", func(t *testing.T) {
		server := New(&types.ServerCfg{
			Addr: ":8888",
			Role: types.RoleMaster,
		}, leveldbStore)
		if server == nil {
			t.FailNow()
		}
	})

	t.Run("NewMaster_2", func(t *testing.T) {
		server := New(&types.ServerCfg{
			Addr: ":8888",
			Role: types.RoleMaster,
		}, store.NewMapStore())
		if server == nil {
			t.FailNow()
		}
	})

	t.Run("NewSlave_1", func(t *testing.T) {
		server := New(&types.ServerCfg{
			Addr:       ":7777",
			Role:       types.RoleSlave,
			MasterAddr: "127.0.0.1:8888",
		}, leveldbStore)
		if server == nil {
			t.FailNow()
		}
	})

	t.Run("NewSlave_2", func(t *testing.T) {
		server := New(&types.ServerCfg{
			Addr:       ":7777",
			Role:       types.RoleSlave,
			MasterAddr: "127.0.0.1:8888",
		}, store.NewMapStore())
		if server == nil {
			t.FailNow()
		}
	})
}
