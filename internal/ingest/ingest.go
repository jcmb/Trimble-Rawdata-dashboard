package ingest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/conn"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/gnss"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/packet"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/rawdata"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/session"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/hub"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/store"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/verbose"
)

// Options tune receiver ingest.
type Options struct {
	Port    string
	Verbose *verbose.Logger
}

// Run connects to the receiver and feeds RAWDATA into the store.
func Run(ctx context.Context, opts Options, st *store.Store, h *hub.Hub) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := runOnce(ctx, opts, st, h); err != nil {
			slog.Warn("receiver disconnected", "err", err)
			snap := st.SetError(err.Error())
			snap.Connected = false
			h.PublishSnapshot(snap)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func runOnce(ctx context.Context, opts Options, st *store.Store, h *hub.Hub) error {
	link, err := conn.Open(ctx, opts.Port)
	if err != nil {
		return err
	}
	defer link.Close()

	if opts.Verbose != nil && opts.Verbose.Level >= verbose.Info {
		slog.Info("receiver connected", "port", opts.Port, "verbose", opts.Verbose.Level.String())
	}

	cfg := session.Config{
		OnPacket: func(p packet.Packet) {
			st.IncPacket()
		},
		OnRAWDATA: func(msg rawdata.Message) {
			snap := st.ApplyRAW(msg)
			snap.Connected = true
			snap.LastError = ""
			h.PublishSnapshot(snap)
		},
	}
	if opts.Verbose != nil {
		cfg = opts.Verbose.SessionConfig(cfg)
	}

	client := session.Dial(link, cfg)
	defer client.Close()

	snap := st.SetConnected(true)
	snap.LastError = ""
	h.PublishSnapshot(snap)

	statsCtx, statsCancel := context.WithCancel(ctx)
	defer statsCancel()
	if opts.Verbose != nil && opts.Verbose.Level >= verbose.Info {
		go runStatsLogger(statsCtx, opts.Port, opts.Verbose)
	}

	done := make(chan struct{})
	go func() {
		client.WaitIdle()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return errors.New("connection closed")
	}
}

func runStatsLogger(ctx context.Context, port string, v *verbose.Logger) {
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			v.LogStats(port)
		}
	}
}

// RunDemo emits synthetic RT27/position data for UI development without hardware.
func RunDemo(ctx context.Context, st *store.Store, h *hub.Hub) {
	snap := st.SetConnected(true)
	h.PublishSnapshot(snap)

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	az := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			az = (az + 15) % 360
			rt := demoRT27(az)
			pos := demoPosition()
			snap := st.ApplyRAW(rt)
			snap = st.ApplyRAW(pos)
			snap.Connected = true
			h.PublishSnapshot(snap)
		}
	}
}

func demoRT27(baseAz int) *rawdata.RT27Survey {
	rt := &rawdata.RT27Survey{
		Header: rawdata.EpochHeader{
			WeekNumber:     2200,
			ReceiverTimeMS: int32(time.Now().UnixMilli() % 604800000),
			NumberSVs:      3,
		},
	}
	systems := []byte{gnss.SystemGPS, gnss.SystemGPS, gnss.SystemGLONASS}
	for i, sys := range systems {
		az := int16((baseAz + i*120) % 360)
		if az > 180 {
			az -= 360
		}
		el := byte(20 + i*25)
		rt.Measurements = append(rt.Measurements, rawdata.Measurement{
			SVID:      byte(1 + i*3),
			SVType:    sys,
			Elevation: el,
			Azimuth:   az,
			Blocks: []rawdata.MeasBlock{{
				BlockNum:  0,
				SVType:    sys,
				BlockType: 0,
				SNR:       38 + float64(i*4),
			}},
		})
	}
	return rt
}

func demoPosition() *rawdata.EnhancedPosition {
	return &rawdata.EnhancedPosition{
		Header: rawdata.PositionHeader{
			WeekNumber:       2200,
			ReceiverTimeSec:  float64(time.Now().Unix() % 86400),
			NumberSVsUsed:    8,
			NumberSVsTracked: 12,
			AugmentationType: gnss.PosAugmentAutonomous,
		},
		Position: rawdata.PositionBlock{
			Latitude:  0.654498, // ~37.5°
			Longitude: -2.094395,
			Altitude:  120.5,
			HDOP:      0.9,
			RMS:       0.012,
		},
	}
}
