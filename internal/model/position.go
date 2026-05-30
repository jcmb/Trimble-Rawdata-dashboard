package model

import "math"

// RT29 enhanced-position SV flag bits (Trimble ICD record type 29).
const (
	PosSVFlagUnhealthy = 1 << 0
	PosSVFlagUsed      = 1 << 1
	PosSVFlagRAIMFault = 1 << 2
)

// PositionSVFlags decodes the per-SV flag byte from enhanced position.
func PositionSVFlags(flag byte) (used, raimFault, unhealthy bool) {
	return flag&PosSVFlagUsed != 0,
		flag&PosSVFlagRAIMFault != 0,
		flag&PosSVFlagUnhealthy != 0
}

// RTKView holds optional RTK correction block fields from enhanced position.
type RTKView struct {
	Mode   byte    `json:"mode,omitempty"`
	AgeSec float64 `json:"ageSec,omitempty"`
	Flags  byte    `json:"flags,omitempty"`
}

// PositionSVView is one SV entry from the enhanced position record.
type PositionSVView struct {
	System         byte   `json:"system"`
	SystemName     string `json:"systemName"`
	SVID           byte   `json:"svid"`
	Flag           byte   `json:"flag,omitempty"`
	UsedInSolution bool   `json:"usedInSolution,omitempty"`
	RAIMFault      bool   `json:"raimFault,omitempty"`
	Unhealthy      bool   `json:"unhealthy,omitempty"`
}

// HorizontalSigma returns √(σN² + σE²).
func HorizontalSigma(sigmaN, sigmaE float64) float64 {
	return math.Hypot(sigmaN, sigmaE)
}
