package main

import (
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/client"
	"github.com/Jarnpher553/gonfig/internal/types"
)

func main() {
	c, err := client.New(&client.Config{
		CfgMeta: &types.ConfigMeta{
			Name: "test",
			Tag: []string{
				"app:lll",
				"ccc:wewf",
			},
		},
		Servers: []string{"127.0.0.1:9020"},
	})
	if err != nil {
		return
	}
	for conf := range c.Watch() {
		fmt.Println(conf)
	}
}
