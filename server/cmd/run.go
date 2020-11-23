package main

import (
	"context"
	"flag"

	"github.com/DanTulovsky/logger"
	"github.com/DanTulovsky/pepper-poker-v2/server/manager"
	"github.com/fatih/color"
)

func main() {

	flag.Parse()
	logg := logger.New("server", color.New(color.FgCyan))

	ctx := context.Background()

	logg.Info("Starting server...")

	m := manager.New()

	if err := m.Run(ctx); err != nil {
		logg.Fatal(err)
	}
}
