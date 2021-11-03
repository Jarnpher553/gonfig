package client

import (
	"errors"
	"fmt"
	"github.com/Jarnpher553/gonfig/component/metadata"
	"github.com/Jarnpher553/gonfig/internal/logger"
	"github.com/Jarnpher553/gonfig/internal/server/types"
	"github.com/Jarnpher553/gonfig/internal/util/color"
	"github.com/Jarnpher553/gonfig/internal/util/retry"
	"github.com/lesismal/arpc"
	"github.com/lesismal/arpc/extension/pubsub"
	alog "github.com/lesismal/arpc/log"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

type GfClient struct {
	current   int
	client    *pubsub.Client
	endpoints []string
	cfgMeta   *metadata.ConfigMeta
	outbound  chan string
}

type Config struct {
	Metadata  *metadata.ConfigMeta
	Endpoints []string
}

func New(config *Config) (*GfClient, error) {
	if len(config.Endpoints) == 0 {
		return nil, errors.New("server's address is empty")
	}
	if config.Metadata == nil {
		return nil, errors.New("config meta is nil")
	}

	lgx := &logger.XLogger{}
	lgx.SetLevel(alog.LevelInfo)
	alog.SetLogger(lgx)
	arpc.DefaultHandler.SetLogTag("[" + color.Green("Gonfig") + "]")

	cl, err := pubsub.NewClient(func() (net.Conn, error) {
		return net.Dial("tcp", config.Endpoints[0])
	})
	if err != nil {
		return nil, err
	}
	cl.Handler.SetLogTag("[" + color.Green("Gonfig") + "]")

	cl.Password = types.PubSubPassword
	err = cl.Authenticate()
	if err != nil {
		return nil, err
	}

	outbound := make(chan string, 5)
	gfClient := &GfClient{
		client:    cl,
		endpoints: config.Endpoints,
		current:   0,
		cfgMeta:   config.Metadata,
		outbound:  outbound,
	}

	cl.Handler.HandleConnected(func(client *arpc.Client) {
		var rsp string
		err := client.Call("/echo", config.Metadata, &rsp, time.Second*1)
		if err == nil {
			outbound <- rsp
		}
	})
	cl.Handler.HandleDisconnected(gfClient.disconnectedHandler)

	return gfClient, nil
}

func (c *GfClient) Watch() chan string {
	var rsp string
	err := c.client.Call("/echo", c.cfgMeta, &rsp, time.Second*1)
	if err == nil {
		c.outbound <- rsp
	}

	c.client.Subscribe(fmt.Sprintf(types.ConfigFormat, c.cfgMeta.Name, strings.Join(c.cfgMeta.Tags, "#")), c.subHandler, time.Second*30)
	return c.outbound
}

func (c *GfClient) subHandler(topic *pubsub.Topic) {
	c.outbound <- string(topic.Data)
}
func (c *GfClient) disconnectedHandler(client *arpc.Client) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	var n int
	for {
		n = r.Intn(len(c.endpoints))
		if c.current == n {
			continue
		}
		break
	}
	c.current = n

	retry.Retry(math.MaxInt32, func() error {
		cl, err := pubsub.NewClient(func() (net.Conn, error) {
			return net.DialTimeout("tcp", c.endpoints[n], time.Second*30)
		})
		if err != nil {
			return err
		}
		cl.Handler.SetLogTag("[" + color.Green("Gonfig") + "]")

		cl.Password = types.PubSubPassword
		err = cl.Authenticate()
		if err != nil {
			return err
		}

		cl.Handler.HandleConnected(func(client *arpc.Client) {
			var rsp string
			err := client.Call("/echo", c.cfgMeta, &rsp, time.Second*1)
			if err == nil {
				c.outbound <- rsp
			}
		})
		cl.Handler.HandleDisconnected(c.disconnectedHandler)
		c.client = cl

		return nil
	})
}

func (c *GfClient) Close() {
	c.client.Handler.HandleDisconnected(nil)
	c.client.Stop()
}
