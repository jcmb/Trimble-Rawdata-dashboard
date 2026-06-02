package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/browseropen"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/server"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/sessions"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/verbose"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/version"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	port := flag.String("port", "", "receiver URI at startup (serial:// or tcp://); optional in hosted mode")
	demo := flag.Bool("demo", false, "start with synthetic RT27/position data")
	allowLocal := flag.Bool("allow-local-hosts", false, "allow web UI connections to loopback and private IP addresses")
	verboseFlag := flag.String("verbose", "off", "debug level: off, info, debug, trace")
	devFlag := flag.Bool("dev", false, "show developer options in the web UI (also enabled when -verbose is debug or trace)")
	basePathFlag := flag.String("base-path", "", "URL prefix when served behind a reverse proxy (e.g. /trimble-dashboard)")
	openBrowserFlag := flag.Bool("open-browser", browseropen.HasGUI(), "open the dashboard in the default web browser (GUI sessions only; set false for servers)")
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

	var vlog *verbose.Logger
	if vlevel >= verbose.Info {
		vlog = verbose.New(vlevel)
	}
	sessMgr := sessions.NewManager(ctx, sessions.Config{
		AllowLocalHosts: *allowLocal,
		Verbose:         vlog,
		VerboseLevel:    vlevel,
	})

	switch {
	case *demo:
		sessMgr.StartFixedDemo()
	case *port != "":
		if err := sessMgr.StartFixedPort(*port); err != nil {
			slog.Error("connect", "err", err)
			os.Exit(1)
		}
	default:
		slog.Info("hosted mode: open the web UI to connect to a receiver")
	}

	showDev := *devFlag || vlevel >= verbose.Debug
	srv := &server.Server{
		Addr:        *addr,
		BasePath:    server.NormalizeBasePath(*basePathFlag),
		Sessions:    sessMgr,
		ShowDev:     showDev,
		OpenBrowser: *openBrowserFlag,
	}
	if showDev {
		slog.Info("developer web UI options enabled")
	}
	if err := srv.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
