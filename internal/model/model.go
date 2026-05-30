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
	Week             uint16  `json:"week"`
	TimeSec          float64 `json:"timeSec"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Altitude         float64 `json:"altitude"`
	VelocityN        float64 `json:"velocityN"`
	VelocityE        float64 `json:"velocityE"`
	VelocityU        float64 `json:"velocityU"`
	ClockOffset      float64 `json:"clockOffset"`
	ClockDrift       float64 `json:"clockDrift"`
	HDOP             float64 `json:"hdop"`
	VDOP             float64 `json:"vdop"`
	TDOP             float64 `json:"tdop"`
	SigmaN           float64 `json:"sigmaN"`
	SigmaE           float64 `json:"sigmaE"`
	SigmaU           float64 `json:"sigmaU"`
	SigmaH           float64 `json:"sigmaH"`
	RMS              float64 `json:"rms"`
	UnitStdDev       float64 `json:"unitStdDev"`
	Augmentation     byte    `json:"augmentation"`
	AugmentationText string  `json:"augmentationText"`
	SVsUsed          byte    `json:"svsUsed"`
	SVsTracked       byte    `json:"svsTracked"`
	SolutionMode     byte    `json:"solutionMode"`
	RAIMInfo         byte    `json:"raimInfo,omitempty"`
	MotionState      byte    `json:"motionState,omitempty"`
	ProcessingType   byte    `json:"processingType,omitempty"`
	RTK              *RTKView         `json:"rtk,omitempty"`
	SVs              []PositionSVView `json:"svs,omitempty"`
}

type RT27View struct {
	Week     uint16      `json:"week"`
	TimeSec  float64     `json:"timeSec"`
	NumSVs   byte        `json:"numSVs"`
	Antennas string      `json:"antennas,omitempty"`
	SVs      []SVRowView `json:"svs"`
}

// Event is pushed over SSE when state changes.
type Event struct {
	Type     string   `json:"type"` // snapshot | error | status
	Snapshot Snapshot `json:"snapshot,omitempty"`
	Message  string   `json:"message,omitempty"`
}
