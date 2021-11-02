package client

import (
	"errors"
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/logger"
	"github.com/Jarnpher553/gonfig/internal/types"
	"github.com/Jarnpher553/gonfig/internal/utility/color"
	"github.com/Jarnpher553/gonfig/internal/utility/retry"
	"github.com/lesismal/arpc"
	"github.com/lesismal/arpc/extension/pubsub"
	alog "github.com/lesismal/arpc/log"
	"math"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	Password = "U2FsdGVkX18Wi+guhMZL8DvCOqfA6j/MWMdUOv9tOvQ="
	Topic    = "config/%s/#%s"
)

type GfClient struct {
	current int
	client  *pubsub.Client
	servers []string
	cfgMeta *types.ConfigMeta
	ch      chan string
}

type Config struct {
	CfgMeta *types.ConfigMeta
	Servers []string
}

func New(config *Config) (*GfClient, error) {
	if len(config.Servers) == 0 {
		return nil, errors.New("server's address is empty")
	}
	if config.CfgMeta == nil {
		return nil, errors.New("config meta is nil")
	}

	lgx := &logger.XLogger{}
	lgx.SetLevel(alog.LevelInfo)
	alog.SetLogger(lgx)
	arpc.DefaultHandler.SetLogTag("[" + color.Green("Gonfig") + "]")
	cl, err := pubsub.NewClient(func() (net.Conn, error) {
		return net.Dial("tcp", config.Servers[0])
	})
	if err != nil {
		return nil, err
	}
	cl.Handler.SetLogTag("[" + color.Green("Gonfig") + "]")

	cl.Password = Password
	err = cl.Authenticate()
	if err != nil {
		return nil, err
	}

	cm := config.CfgMeta
	var rsp string
	err = cl.Call("/echo", cm, &rsp, time.Second*5)
	if err != nil {
		return nil, err
	}

	gfClient := &GfClient{client: cl, servers: config.Servers, current: 0, cfgMeta: config.CfgMeta, ch: make(chan string, 5)}
	gfClient.client.Handler.HandleDisconnected(gfClient.disconnectedHandler)
	gfClient.ch <- rsp

	cl.Handler.HandleConnected(func(client *arpc.Client) {
		cm := config.CfgMeta
		var rsp string
		err = cl.Call("/echo", cm, &rsp, time.Second*5)
		if err == nil {
			gfClient.ch <- rsp
		}
	})
	return gfClient, nil
}

func (c *GfClient) Watch() chan string {
	c.client.Subscribe(fmt.Sprintf(Topic, c.cfgMeta.Name, strings.Join(c.cfgMeta.Tag, "#")), c.subHandler, time.Second*30)
	return c.ch
}

func (c *GfClient) subHandler(topic *pubsub.Topic) {
	c.ch <- string(topic.Data)
}
func (c *GfClient) disconnectedHandler(client *arpc.Client) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	var n int
	for {
		n = r.Intn(len(c.servers))
		if c.current == n {
			continue
		}
		break
	}
	c.current = n

	retry.Retry(math.MaxInt32, func() error {
		cl, err := pubsub.NewClient(func() (net.Conn, error) {
			return net.DialTimeout("tcp", c.servers[n], time.Second*30)
		})
		if err != nil {
			return err
		}

		cl.Password = Password
		err = cl.Authenticate()
		if err != nil {
			return err
		}

		cl.Handler.HandleDisconnected(c.disconnectedHandler)
		c.client = cl
		return nil
	})
}

func (c *GfClient) Close() {
	c.client.Handler.HandleDisconnected(nil)
	c.client.Stop()
}
