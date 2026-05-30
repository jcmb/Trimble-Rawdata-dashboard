package ingest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/conn"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/dcol"
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
			NumberSVs:      7,
		},
	}
	cases := []struct {
		sys    byte
		svid   byte
		blocks []rawdata.MeasBlock
	}{
		{
			sys: gnss.SystemGPS, svid: 7,
			blocks: []rawdata.MeasBlock{
				{BlockNum: 0, BlockType: 0, TrackType: 0, SNR: 42},
				{BlockNum: 1, BlockType: 1, TrackType: 5, SNR: 38},
				{BlockNum: 2, BlockType: 2, TrackType: 8, SNR: 35},
			},
		},
		{
			sys: gnss.SystemSBAS, svid: 133,
			blocks: []rawdata.MeasBlock{
				{BlockNum: 0, BlockType: 0, TrackType: 23, SNR: 44},
				{BlockNum: 1, BlockType: 2, TrackType: 6, SNR: 40},
			},
		},
		{
			sys: gnss.SystemGalileo, svid: 11,
			blocks: []rawdata.MeasBlock{
				{BlockNum: 0, BlockType: 8, TrackType: 23, SNR: 41},
				{BlockNum: 1, BlockType: 2, TrackType: 11, SNR: 38},
				{BlockNum: 2, BlockType: 3, TrackType: 11, SNR: 36},
				{BlockNum: 3, BlockType: 4, TrackType: 14, SNR: 34},
				{BlockNum: 4, BlockType: 5, TrackType: 26, SNR: 37},
			},
		},
		{
			sys: gnss.SystemBeidou, svid: 19,
			blocks: []rawdata.MeasBlock{
				{BlockNum: 0, BlockType: 6, TrackType: 26, SNR: 43},
				{BlockNum: 1, BlockType: 7, TrackType: 29, SNR: 36},
			},
		},
		{
			sys: gnss.SystemGLONASS, svid: 12,
			blocks: []rawdata.MeasBlock{
				{BlockNum: 0, BlockType: 0, TrackType: 0, SNR: 39},
				{BlockNum: 1, BlockType: 1, TrackType: 1, SNR: 37},
				{BlockNum: 2, BlockType: 1, TrackType: 0, SNR: 35},
			},
		},
		{
			sys: gnss.SystemOmniStar, svid: 1,
			blocks: []rawdata.MeasBlock{
				{BlockNum: 0, BlockType: 0, TrackType: 0, SNR: 46},
			},
		},
	}
	for i, c := range cases {
		az := int16((baseAz + i*72) % 360)
		el := byte(15 + i*15)
		blocks := make([]rawdata.MeasBlock, len(c.blocks))
		for j, b := range c.blocks {
			blocks[j] = b
			// Demo: GPS SV7 slips every ~120° rotation (after initial lock slip is ignored).
			if c.svid == 7 && j == 0 && baseAz > 0 && baseAz%90 == 0 {
				blocks[j].MeasFlags = []byte{dcol.MeasFlag1CycleSlip}
				blocks[j].CycleSlipCount = byte(baseAz / 90)
			}
		}
		rt.Measurements = append(rt.Measurements, rawdata.Measurement{
			SVID:          c.svid,
			SVType:        c.sys,
			AntennaNumber: 0,
			Elevation:     el,
			Azimuth:       az,
			Blocks:        blocks,
		})
	}
	// Dual-antenna demo: GPS 7 tracked on both antennas; OmniSTAR on antenna 0 only.
	gps7 := rt.Measurements[0]
	rt.Measurements = append(rt.Measurements, rawdata.Measurement{
		SVID:          gps7.SVID,
		SVType:        gps7.SVType,
		AntennaNumber: 1,
		Elevation:     gps7.Elevation,
		Azimuth:       gps7.Azimuth,
		Blocks:        gps7.Blocks,
	})
	rt.Header.NumberSVs = byte(len(rt.Measurements))
	return rt
}

func demoPosition() *rawdata.EnhancedPosition {
	pos := rawdata.PositionBlock{
		Latitude:    39.897088,
		Longitude:   -105.115120,
		Altitude:    120.5,
		VelocityN:   0.12,
		VelocityE:   -0.05,
		VelocityU:   0.01,
		ClockOffset: 0.0001,
		ClockDrift:  1e-9,
		HDOP:        0.9,
		VDOP:        1.2,
		TDOP:        1.5,
		SigmaN:      0.008,
		SigmaE:      0.009,
		SigmaU:      0.015,
		RMS:         0.012,
		UnitStdDev:  0.010,
	}
	return &rawdata.EnhancedPosition{
		Header: rawdata.PositionHeader{
			WeekNumber:       2200,
			ReceiverTimeSec:  float64(time.Now().Unix() % 86400),
			NumberSVsUsed:    4,
			NumberSVsTracked: 6,
			AugmentationType: gnss.PosAugmentRTKFixed,
		},
		Position: pos,
		RTK: &rawdata.RTKBlock{
			Mode:  1,
			Age:   1.25,
			Flags: 0,
		},
		SVs: []rawdata.SVEntry{
			{SVID: 7, SVType: gnss.SystemGPS, Flag: (1 << 1) | (1 << 2)}, // used + RAIM fault
			{SVID: 133, SVType: gnss.SystemSBAS, Flag: 1 << 1},
			{SVID: 11, SVType: gnss.SystemGalileo, Flag: 1 << 1},
			{SVID: 19, SVType: gnss.SystemBeidou, Flag: 1 << 1},
			{SVID: 12, SVType: gnss.SystemGLONASS, Flag: 0},
			{SVID: 1, SVType: gnss.SystemOmniStar, Flag: 1 << 1},
		},
	}
}
