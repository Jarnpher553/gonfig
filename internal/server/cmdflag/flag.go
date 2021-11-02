package cmdflag

import (
	"flag"
	"fmt"
	"github.com/Jarnpher553/gonfig/internal/server/vars"
	ipAddr "github.com/Jarnpher553/gonfig/internal/utility/addr"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

var role = flag.String("role", "master", "server role")
var addr = flag.String("addr", ":9019", "server addr")
var master = flag.String("master", "", "master addr")

func Parse() *vars.ServerCfg {
	c := &vars.ServerCfg{}

	flag.Parse()

	serverRole := vars.Role(strings.Title(strings.ToLower(*role)))

	if serverRole != vars.RoleMaster && serverRole != vars.RoleSlave {
		log.Fatalln("Role hasn't be master or slave")
	}

	if serverRole != vars.RoleMaster && *master == "" {
		log.Fatalln("Master address of slave peer can't be empty")
	}

	c.Role = serverRole

	var addrIP string
	var addrPort string
	if !strings.Contains(*addr, ":") {
		log.Fatalln("Server address format error")
	} else {
		splitAddr := strings.Split(*addr, ":")
		if splitAddr[0] != "" {
			ip := net.ParseIP(splitAddr[0])
			if ip == nil {
				log.Fatalln("Server ip format error")
			}
		}
		addrIP = splitAddr[0]
		port := splitAddr[1]
		if _, err := strconv.Atoi(port); err != nil {
			log.Fatalln("Server port format error")
		}
		addrPort = port
	}

	if addrIP == "" {
		envIP := os.Getenv("GONFIG_HOST_IP")
		if envIP != "" {
			addrIP = envIP
		} else {
			a, err := ipAddr.Extract(*addr)
			if err != nil {
				log.Fatalln("Server address:", err)
			}
			addrIP = a
		}
	}
	c.Addr = fmt.Sprintf("%s:%s", addrIP, addrPort)

	if serverRole == vars.RoleSlave {
		splitAddr := strings.Split(*master, ":")
		if len(splitAddr) != 2 {
			log.Fatalln("Master address format error")
		}
		ip := net.ParseIP(splitAddr[0])
		if ip == nil {
			log.Fatalln("Master ip format error")
		}
		port := splitAddr[1]
		if _, err := strconv.Atoi(port); err != nil {
			log.Fatalln("Master port format error")
		}
		c.MasterAddr = *master
	}

	return c
}
