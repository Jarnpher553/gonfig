package cmdflag

import (
	"flag"
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/logger"
	"github.com/Jarnpher553/gonfig/internal/server/types"
	ipAddr "github.com/Jarnpher553/gonfig/internal/util/addr"
	"net"
	"os"
	"strconv"
	"strings"
)

var role = flag.String("role", "master", "server role")
var addr = flag.String("addr", ":9019", "server addr")
var master = flag.String("master", "", "master addr")

func Parse(log *logger.XLogger) *types.ServerCfg {
	c := &types.ServerCfg{}

	flag.Parse()

	serverRole := types.Role(strings.Title(strings.ToLower(*role)))

	if serverRole != types.RoleMaster && serverRole != types.RoleSlave {
		log.Fatal("Role hasn't be master or slave")
	}

	if serverRole != types.RoleMaster && *master == "" {
		log.Fatal("Master address of slave peer can't be empty")
	}

	c.Role = serverRole

	var addrIP string
	var addrPort string
	if !strings.Contains(*addr, ":") {
		log.Fatal("Server address format error")
	} else {
		splitAddr := strings.Split(*addr, ":")
		if splitAddr[0] != "" {
			ip := net.ParseIP(splitAddr[0])
			if ip == nil {
				log.Fatal("Server ip format error")
			}
		}
		addrIP = splitAddr[0]
		port := splitAddr[1]
		if _, err := strconv.Atoi(port); err != nil {
			log.Fatal("Server port format error")
		}
		addrPort = port
	}

	if addrIP == "" {
		envIP := os.Getenv("GONFIG_HOST_IP")
		if envIP != "" {
			addrIP = envIP
		} else {
			a, err := ipAddr.ParseIP(*addr)
			if err != nil {
				log.Fatal("Server address: %s", err)
			}
			addrIP = a
		}
	}
	c.Addr = fmt.Sprintf("%s:%s", addrIP, addrPort)

	if serverRole == types.RoleSlave {
		splitAddr := strings.Split(*master, ":")
		if len(splitAddr) != 2 {
			log.Fatal("Master address format error")
		}
		ip := net.ParseIP(splitAddr[0])
		if ip == nil {
			log.Fatal("Master ip format error")
		}
		port := splitAddr[1]
		if _, err := strconv.Atoi(port); err != nil {
			log.Fatal("Master port format error")
		}
		c.MasterAddr = *master
	}

	return c
}
