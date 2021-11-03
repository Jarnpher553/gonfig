package main

import (
	"fmt"
	"github.com/Jarnpher553/gonfig/component/client"
	"github.com/Jarnpher553/gonfig/component/metadata"
)

func main() {
	c, err := client.New(&client.Config{
		Metadata: &metadata.ConfigMeta{
			Name: "test",
			Tags: []string{
				"app:lll",
				"ccc:wewf",
			},
		},
		Endpoints: []string{"127.0.0.1:9020"},
	})
	if err != nil {
		return
	}
	for conf := range c.Watch() {
		fmt.Println(conf)
	}
}
