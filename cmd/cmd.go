package main

import (
	"github.com/Jarnpher553/gonfig/internal/server"
)

func main() {
	s := server.New()
	s.Serve()
}
