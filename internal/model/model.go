package model

import "time"

// Snapshot is the full dashboard state sent to the browser.
type Snapshot struct {
	UpdatedAt   time.Time     `json:"updatedAt"`
	Connected   bool          `json:"connected"`
	Port        string        `json:"port,omitempty"`
	LastError   string        `json:"lastError,omitempty"`
	Position    *PositionView `json:"position,omitempty"`
	RT27        *RT27View     `json:"rt27,omitempty"`
	PacketCount int64         `json:"packetCount"`
	RAWCount    int64         `json:"rawCount"`
}

type PositionView struct {
	Week         uint16  `json:"week"`
	TimeSec      float64 `json:"timeSec"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Altitude     float64 `json:"altitude"`
	Augmentation byte    `json:"augmentation"`
	SVsUsed      byte    `json:"svsUsed"`
	SVsTracked   byte    `json:"svsTracked"`
	HDOP         float64 `json:"hdop"`
	RMS          float64 `json:"rms"`
	SolutionMode byte    `json:"solutionMode"`
}

type RT27View struct {
	Week    uint16       `json:"week"`
	TimeSec float64      `json:"timeSec"`
	NumSVs  byte         `json:"numSVs"`
	Signals []SignalView `json:"signals"`
}

type SignalView struct {
	System     byte    `json:"system"`
	SystemName string  `json:"systemName"`
	SVID       byte    `json:"svid"`
	Azimuth    int16   `json:"azimuth"`
	Elevation  byte    `json:"elevation"`
	Block      int     `json:"block"`
	BlockType  byte    `json:"blockType"`
	SNR        float64 `json:"snr"`
}

// Event is pushed over SSE when state changes.
type Event struct {
	Type     string   `json:"type"` // snapshot | error | status
	Snapshot Snapshot `json:"snapshot,omitempty"`
	Message  string   `json:"message,omitempty"`
}
