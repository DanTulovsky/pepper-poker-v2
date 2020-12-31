package main

import (
	"context"
	"flag"

	"github.com/DanTulovsky/logger"
	"github.com/fatih/color"

	"github.com/DanTulovsky/pepper-poker-v2/server/manager"

	_ "net/http/pprof"
)

var ()

const (
	version = "0.1.1"
)

func main() {

	flag.Parse()
	logg := logger.New("server", color.New(color.FgCyan))

	ctx := context.Background()

	logg.Infof("Starting server (version: %v)...", version)

	m := manager.New()

	if err := m.Run(ctx); err != nil {
		logg.Fatal(err)
	}

}
