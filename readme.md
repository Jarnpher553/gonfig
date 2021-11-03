# gonfig

gonfig is a lightweight config center

# usage

```shell
# download and install gonfig
go install github.com/Jarnpher553/gonfig/cmd
# start master http server listen on port 9019 default
# and  rpc server listen on port 9020 (http port plus 1)
gonfig -role master 
# start slave server listen on port 8888
# and  rpc server listen on port 8889
# at the same time, register self to master endpoint
gonfig -role slave -master 127.0.0.1:9019 -addr 8888
```

- master

```shell
  __  __                 _
 |  \/  |   __ _   ___  | |_    ___   _ __
 | |\/| |  / _` | / __| | __|  / _ \ | '__|
 | |  | | | (_| | \__ \ | |_  |  __/ | |
 |_|  |_|  \__,_| |___/  \__|  \___| |_|
2021/11/03 16:44:42 [INF] [HttpServer] HttpRouter [0] Method [POST] Path [/register]
2021/11/03 16:44:42 [INF] [HttpServer] HttpRouter [1] Method [POST] Path [/unregister]
2021/11/03 16:44:42 [INF] [HttpServer] HttpRouter [2] Method [POST] Path [/push]
2021/11/03 16:44:42 [INF] [HttpServer] HttpRouter [3] Method [POST] Path [/pull]
2021/11/03 16:44:42 [INF] [RpcServer] RpcRouter [0] Method [/echo]
2021/11/03 16:44:42 [INF] [HttpServer] [Master]/[0589d590-3a22-44c9-947a-0e75cde5c813] listening on [:9019]
2021/11/03 16:44:42 [INF] [RpcServer] Running On: "[::]:9020"
2021/11/03 16:45:06 [INF] [HttpServer] Slave id:[11aced0f-cc8a-41ea-867e-2c106121f589] addr:[127.0.0.1:8888] online
```

- slave

```shell
  ____    _
 / ___|  | |   __ _  __   __   ___
 \___ \  | |  / _` | \ \ / /  / _ \
  ___) | | | | (_| |  \ V /  |  __/
 |____/  |_|  \__,_|   \_/    \___|
2021/11/03 16:45:23 [INF] [HttpServer] HttpRouter [0] Method [POST] Path [/sync]
2021/11/03 16:45:23 [INF] [HttpServer] HttpRouter [1] Method [GET] Path [/health]
2021/11/03 16:45:23 [INF] [HttpServer] HttpRouter [2] Method [POST] Path [/pull]
2021/11/03 16:45:23 [INF] [RpcServer] RpcRouter [0] Method [/echo]
2021/11/03 16:45:23 [INF] [HttpServer] [Slave]/[0f8c69f4-dcec-46f9-b208-e6b9826975f2] listening on [:8888]
2021/11/03 16:45:23 [INF] [RpcServer] Running On: "[::]:8889"
```

# example

## push config

* use postman or other http client
* url: ***http://127.0.0.1:9019/push***
* method: ***POST***
* content:

```json
{
  "Tag": [
    "demo",
    "test",
    "local"
  ],
  "Name": "demo_app",
  "Body": "version: 1.0\nappName: demo_app ..."
}
```

## client

```go
package main

import (
	"fmt"
	"github.com/Jarnpher553/gonfig/component/client"
	"github.com/Jarnpher553/gonfig/component/metadata"
)

func main() {
	c, err := client.New(&client.Config{
		Metadata: &metadata.ConfigMeta{
			Name: "demo_app",
			Tags: []string{ // config tag
				"demo",
				"test",
				"local",
			},
		},
		Endpoints: []string{"127.0.0.1:9020"}, // rpc server list
	})
	if err != nil {
		return
	}
	for conf := range c.Watch() { // c.Watch() will return a channel
		fmt.Println(conf) // "version: 1.0\nappName: demo_app ..."
	}
}
```