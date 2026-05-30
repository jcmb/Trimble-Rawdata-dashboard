package store

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/rawdata"
	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
)

// Store holds the latest dashboard snapshot (thread-safe).
type Store struct {
	mu   sync.RWMutex
	snap model.Snapshot
}

func New(port string) *Store {
	return &Store{
		snap: model.Snapshot{
			Port:      port,
			Connected: false,
			UpdatedAt: time.Now(),
		},
	}
}

func (s *Store) Snapshot() model.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snap
}

func (s *Store) SetConnected(on bool) model.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.Connected = on
	s.snap.UpdatedAt = time.Now()
	return s.snap
}

func (s *Store) SetError(err string) model.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.LastError = err
	s.snap.UpdatedAt = time.Now()
	return s.snap
}

func (s *Store) IncPacket() {
	s.mu.Lock()
	s.snap.PacketCount++
	s.snap.UpdatedAt = time.Now()
	s.mu.Unlock()
}

func (s *Store) ApplyRAW(msg rawdata.Message) model.Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snap.RAWCount++
	s.snap.UpdatedAt = time.Now()

	switch m := msg.(type) {
	case *rawdata.RT27Survey:
		s.snap.RT27 = rt27View(m)
		mergePositionSVs(s.snap.RT27, s.snap.Position)
	case *rawdata.EnhancedPosition:
		s.snap.Position = positionView(m)
		mergePositionSVs(s.snap.RT27, s.snap.Position)
	}
	return s.snap
}

func rt27View(m *rawdata.RT27Survey) *model.RT27View {
	v := &model.RT27View{
		Week:     m.Header.WeekNumber,
		TimeSec:  float64(m.Header.ReceiverTimeMS) / 1000,
		NumSVs:   m.Header.NumberSVs,
		Antennas: rt27Antennas(m.Measurements),
	}
	for _, meas := range m.Measurements {
		row := model.SVRowView{
			System:     meas.SVType,
			SystemName: model.SystemName(meas.SVType),
			SVID:       meas.SVID,
			Azimuth:    normalizeAzimuth(meas.Azimuth),
			Elevation:  meas.Elevation,
		}
		for _, blk := range meas.Blocks {
			sig := model.SignalView{
				BlockType:      blk.BlockType,
				SNR:            blk.SNR,
				TrackType:      blk.TrackType,
				TrackName:      model.TrackTypeName(meas.SVType, blk.BlockType, blk.TrackType),
				TrackHint:      model.TrackTypeHint(blk.TrackType),
				CycleSlipNow:   model.MeasCycleSlipNow(blk.MeasFlags),
				CycleSlipCount: blk.CycleSlipCount,
			}
			assignBand(&row, sig, meas.SVType, blk.BlockType, blk.TrackType)
		}
		orderBandSignals(&row)
		v.SVs = append(v.SVs, row)
	}
	return v
}

func assignBand(row *model.SVRowView, sig model.SignalView, system, blockType, trackType byte) {
	switch model.BandSlot(system, blockType, trackType) {
	case model.BandL1:
		model.AppendSignal(&row.L1, sig)
	case model.BandL2:
		model.AppendSignal(&row.L2, sig)
	case model.BandL5:
		model.AppendSignal(&row.L5, sig)
	case model.BandL6:
		model.AppendSignal(&row.L6, sig)
	}
}

// orderBandSignals sorts signals within each band (C/A before P, matching pydcollib display).
func orderBandSignals(row *model.SVRowView) {
	row.L1.Signals = orderSignals(row.System, row.L1.Signals)
	row.L2.Signals = orderSignals(row.System, row.L2.Signals)
	row.L5.Signals = orderSignals(row.System, row.L5.Signals)
	row.L6.Signals = orderSignals(row.System, row.L6.Signals)
}

func orderSignals(system byte, sigs []model.SignalView) []model.SignalView {
	if len(sigs) < 2 {
		return sigs
	}
	// Stable sort: C/A (track 0) before P (track 1) for GPS/GLONASS-style dual codes.
	out := make([]model.SignalView, len(sigs))
	copy(out, sigs)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if signalOrder(system, out[j]) < signalOrder(system, out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func signalOrder(system byte, sig model.SignalView) int {
	if system == model.SystemGalileo {
		switch sig.BlockType {
		case model.FreqL5:
			return 0
		case model.FreqE5B:
			return 1
		case model.FreqE5AB:
			return 2
		case model.FreqE6:
			return 3
		}
	}
	if system == model.SystemGLONASS || system == model.SystemGPS || system == model.SystemSBAS {
		switch sig.TrackType {
		case 1:
			return 0 // P before C/A
		case 0:
			return 1
		}
	}
	return int(sig.TrackType)
}

func rt27Antennas(measurements []rawdata.Measurement) string {
	if len(measurements) == 0 {
		return ""
	}
	seen := make(map[byte]bool)
	var nums []byte
	for _, m := range measurements {
		if seen[m.AntennaNumber] {
			continue
		}
		seen[m.AntennaNumber] = true
		nums = append(nums, m.AntennaNumber)
	}
	for i := 0; i < len(nums); i++ {
		for j := i + 1; j < len(nums); j++ {
			if nums[j] < nums[i] {
				nums[i], nums[j] = nums[j], nums[i]
			}
		}
	}
	out := make([]string, len(nums))
	for i, n := range nums {
		out[i] = strconv.Itoa(int(n))
	}
	return strings.Join(out, ",")
}

func normalizeAzimuth(az int16) int16 {
	a := int(az) % 360
	if a < 0 {
		a += 360
	}
	return int16(a)
}

func positionView(m *rawdata.EnhancedPosition) *model.PositionView {
	aug := m.Header.AugmentationType
	p := m.Position
	v := &model.PositionView{
		Week:             m.Header.WeekNumber,
		TimeSec:          m.Header.ReceiverTimeSec,
		Latitude:         p.Latitude,
		Longitude:        p.Longitude,
		Altitude:         p.Altitude,
		VelocityN:        p.VelocityN,
		VelocityE:        p.VelocityE,
		VelocityU:        p.VelocityU,
		ClockOffset:      p.ClockOffset,
		ClockDrift:       p.ClockDrift,
		HDOP:             p.HDOP,
		VDOP:             p.VDOP,
		TDOP:             p.TDOP,
		SigmaN:           p.SigmaN,
		SigmaE:           p.SigmaE,
		SigmaU:           p.SigmaU,
		SigmaH:           model.HorizontalSigma(p.SigmaN, p.SigmaE),
		RMS:              p.RMS,
		UnitStdDev:       p.UnitStdDev,
		Augmentation:     aug,
		AugmentationText: model.AugmentationName(aug),
		SVsUsed:          m.Header.NumberSVsUsed,
		SVsTracked:       m.Header.NumberSVsTracked,
		SolutionMode:     m.Header.SolutionMode,
		RAIMInfo:         m.Header.RAIMInfo,
		MotionState:      m.Header.MotionState,
		ProcessingType:   m.Header.ProcessingType,
	}
	if m.RTK != nil {
		v.RTK = &model.RTKView{
			Mode:   m.RTK.Mode,
			AgeSec: m.RTK.Age,
			Flags:  m.RTK.Flags,
		}
	}
	for _, sv := range m.SVs {
		used, raim, unhealthy := model.PositionSVFlags(sv.Flag)
		v.SVs = append(v.SVs, model.PositionSVView{
			System:         sv.SVType,
			SystemName:     model.SystemName(sv.SVType),
			SVID:           sv.SVID,
			Flag:           sv.Flag,
			UsedInSolution: used,
			RAIMFault:      raim,
			Unhealthy:      unhealthy,
		})
	}
	return v
}

func mergePositionSVs(rt27 *model.RT27View, pos *model.PositionView) {
	if rt27 == nil || pos == nil || len(pos.SVs) == 0 {
		return
	}
	lookup := make(map[string]model.PositionSVView, len(pos.SVs))
	for _, sv := range pos.SVs {
		lookup[svKey(sv.System, sv.SVID)] = sv
	}
	for i := range rt27.SVs {
		row := &rt27.SVs[i]
		row.UsedInSolution = false
		row.RAIMFault = false
		row.Unhealthy = false
		if psv, ok := lookup[svKey(row.System, row.SVID)]; ok {
			row.UsedInSolution = psv.UsedInSolution
			row.RAIMFault = psv.RAIMFault
			row.Unhealthy = psv.Unhealthy
		}
	}
}

func svKey(system, svid byte) string {
	return fmt.Sprintf("%d:%d", system, svid)
}
