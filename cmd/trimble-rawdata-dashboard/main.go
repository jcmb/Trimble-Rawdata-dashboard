package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/hub"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/ingest"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/server"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/store"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/verbose"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/version"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	port := flag.String("port", "", "receiver URI (serial:// or tcp://)")
	demo := flag.Bool("demo", false, "synthetic RT27/position data (no hardware)")
	verboseFlag := flag.String("verbose", "off", "debug level: off, info, debug, trace")
	flag.Parse()

	vlevel, err := verbose.ParseLevel(*verboseFlag)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	if vlevel >= verbose.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	slog.Info("trimble-rawdata-dashboard", "version", version.String())

	if *port == "" && !*demo {
		slog.Error("provide -port or -demo")
		flag.Usage()
		os.Exit(1)
	}
	if *port != "" && *demo {
		slog.Error("use either -port or -demo, not both")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st := store.New(*port)
	h := hub.New()

	if *demo {
		go ingest.RunDemo(ctx, st, h)
	} else {
		vlog := verbose.New(vlevel)
		go ingest.Run(ctx, ingest.Options{Port: *port, Verbose: vlog}, st, h)
	}

	srv := &server.Server{Addr: *addr, Store: st, Hub: h}
	if err := srv.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
