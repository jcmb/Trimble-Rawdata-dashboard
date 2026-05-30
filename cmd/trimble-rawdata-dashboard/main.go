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
	port := flag.String("port", "", "receiver URI at startup (serial:// or tcp://); optional in hosted mode")
	demo := flag.Bool("demo", false, "start with synthetic RT27/position data")
	allowLocal := flag.Bool("allow-local-hosts", false, "allow web UI connections to loopback and private IP addresses")
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

	if *port != "" && *demo {
		slog.Error("use either -port or -demo, not both")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st := store.New(*port)
	h := hub.New()

	var vlog *verbose.Logger
	if vlevel >= verbose.Info {
		vlog = verbose.New(vlevel)
	}
	mgr := ingest.NewManager(ctx, st, h, ingest.ManagerConfig{
		AllowLocalHosts: *allowLocal,
		Verbose:         vlog,
		VerboseLevel:    vlevel,
	})

	switch {
	case *demo:
		mgr.StartFixedDemo()
	case *port != "":
		if err := mgr.StartFixedPort(*port); err != nil {
			slog.Error("connect", "err", err)
			os.Exit(1)
		}
	default:
		slog.Info("hosted mode: open the web UI to connect to a receiver")
	}

	srv := &server.Server{Addr: *addr, Store: st, Hub: h, Manager: mgr}
	if err := srv.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
