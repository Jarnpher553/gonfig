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