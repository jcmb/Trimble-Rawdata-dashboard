package store

import (
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
	case *rawdata.EnhancedPosition:
		s.snap.Position = positionView(m)
	}
	return s.snap
}

func rt27View(m *rawdata.RT27Survey) *model.RT27View {
	v := &model.RT27View{
		Week:    m.Header.WeekNumber,
		TimeSec: float64(m.Header.ReceiverTimeMS) / 1000,
		NumSVs:  m.Header.NumberSVs,
	}
	for _, e := range m.SNREntries() {
		v.Signals = append(v.Signals, model.SignalView{
			System:     e.System,
			SystemName: model.SystemName(e.System),
			SVID:       e.SVID,
			Azimuth:    e.Azimuth,
			Elevation:  e.Elevation,
			Block:      e.Block,
			BlockType:  e.BlockType,
			SNR:        e.SNR,
		})
	}
	return v
}

func positionView(m *rawdata.EnhancedPosition) *model.PositionView {
	return &model.PositionView{
		Week:         m.Header.WeekNumber,
		TimeSec:      m.Header.ReceiverTimeSec,
		Latitude:     m.Position.Latitude,
		Longitude:    m.Position.Longitude,
		Altitude:     m.Position.Altitude,
		Augmentation: m.Header.AugmentationType,
		SVsUsed:      m.Header.NumberSVsUsed,
		SVsTracked:   m.Header.NumberSVsTracked,
		HDOP:         m.Position.HDOP,
		RMS:          m.Position.RMS,
		SolutionMode: m.Header.SolutionMode,
	}
}
