package rpchandler

import (
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/server/types"
	"github.com/lesismal/arpc"
	"strings"
)

type RpcHandlerFunc func(s *types.ServiceCtx) func(c *arpc.Context)

func EchoHandler(ctx *types.ServiceCtx) func(c *arpc.Context) {
	return func(c *arpc.Context) {

		var in types.ConfigMeta
		if err := c.Bind(&in); err != nil {
			return
		}
		cfgName := fmt.Sprintf(types.ConfigFormat, in.Name, strings.Join(in.Tags, "#"))
		v, err := ctx.Store.Get([]byte(cfgName))
		if err != nil {
			return
		}

		c.Client.Set("config_meta", &in)
		c.Write(v)
	}
}
