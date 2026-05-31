package model

import "fmt"

// SignalView is one tracked signal within a band column.
type SignalView struct {
	BlockType      byte    `json:"blockType,omitempty"`
	SNR            float64 `json:"snr"`
	TrackType      byte    `json:"trackType,omitempty"`
	TrackName      string  `json:"trackName"`
	TrackHint      string  `json:"trackHint,omitempty"`
	CycleSlipNow   bool    `json:"cycleSlipNow,omitempty"`
	CycleSlipCount byte    `json:"cycleSlipCount,omitempty"`
}

// BandView aggregates all signals tracked on one band column (e.g. L1 P + C/A).
type BandView struct {
	Present bool         `json:"present"`
	Signals []SignalView `json:"signals,omitempty"`
}

// SVRowView is one satellite row in the RT27 table.
type SVRowView struct {
	System           byte     `json:"system"`
	SystemName       string   `json:"systemName"`
	SVID             byte     `json:"svid"`
	Antenna          byte     `json:"antenna"`
	Azimuth          int16    `json:"azimuth"`
	Elevation        byte     `json:"elevation"`
	UsedInSolution   bool     `json:"usedInSolution,omitempty"`
	RAIMFault        bool     `json:"raimFault,omitempty"`
	Unhealthy        bool     `json:"unhealthy,omitempty"`
	L1               BandView `json:"l1"`
	L2               BandView `json:"l2"`
	L5               BandView `json:"l5"`
	L6               BandView `json:"l6"`
}

// TrackTypeName maps RT27 TrackType to a short label (pydcollib GetSNRs sys_sig_names).
func TrackTypeName(system, blockType, trackType byte) string {
	if name := signalCodeName(system, blockType, trackType); name != "" {
		return name
	}
	if name := galileoTrackName(blockType, trackType); system == SystemGalileo && name != "" {
		return name
	}
	if name := glonassTrackName(blockType, trackType); system == SystemGLONASS && name != "" {
		return name
	}
	if name := sbasTrackName(blockType, trackType); system == SystemSBAS && name != "" {
		return name
	}
	if name := mssTrackName(system, blockType, trackType); name != "" {
		return name
	}

	switch trackType {
	case 0:
		return "C/A"
	case 1:
		return "P"
	case 2:
		if blockType == FreqL2 {
			return "L2E"
		}
		return "E"
	case 3:
		return "L2CM"
	case 4:
		return "L2CL"
	case 5:
		return "L2C"
	case 6:
		return "L5-I"
	case 7:
		return "L5-Q"
	case 8:
		return "L5-IQ"
	case 9:
		return "Y"
	case 10:
		return "M"
	case 11:
		return "BPSK-DP"
	case 12:
		return "BPSK-P"
	case 13:
		return "BPSK-D"
	case 14:
		return "AltBoc"
	case 22:
		return "BOC(1,1) D"
	case 23:
		return "CBOC"
	case 24:
		return "MBOC(1,1) P"
	case 25:
		return "MBOC(1,1) D"
	case 26:
		if blockType == FreqB1 {
			return "B1"
		}
		return "E6"
	case 27:
		return "B1-2"
	case 28:
		return "B2"
	case 29:
		return "B3"
	case 30:
		return "SAIF"
	case 31, 36, 37, 38:
		return "E6"
	default:
		return fmt.Sprintf("unknown (%d)", trackType)
	}
}

// signalCodeName maps RT27 track types that denote signal codes across GNSS systems.
func signalCodeName(system, blockType, trackType byte) string {
	_ = blockType
	switch trackType {
	case 3:
		return "L2CM"
	case 4:
		return "L2CL"
	case 6:
		if isBeidouSystem(system) {
			return "B2B"
		}
	case 8:
		if isBeidouSystem(system) {
			return "B2A"
		}
	case 20:
		switch system {
		case SystemGPS, SystemQZSS:
			return "L1C"
		case SystemBeidouOld, SystemBeidou, SystemBeidouB1Geo:
			return "B1C"
		case SystemGalileo:
			return "E1"
		default:
			return "BOC(PD)"
		}
	}
	return ""
}

func isBeidouSystem(system byte) bool {
	return system == SystemBeidouOld || system == SystemBeidou || system == SystemBeidouB1Geo
}

func galileoTrackName(blockType, trackType byte) string {
	switch blockType {
	case FreqE1:
		return "E1"
	case FreqL5:
		return "E5A"
	case FreqE5B:
		return "E5B"
	case FreqE5AB:
		return "E5Alt"
	case FreqE6:
		return "E6"
	}
	_ = trackType
	return ""
}

func glonassTrackName(blockType, trackType byte) string {
	switch blockType {
	case FreqL1:
		switch trackType {
		case 0:
			return "CA"
		case 1:
			return "P"
		}
	case FreqL2:
		switch trackType {
		case 0:
			return "CA"
		case 1:
			return "P"
		}
	case FreqG3:
		switch trackType {
		case 32:
			return "G3-D+P"
		case 33:
			return "G3-P"
		case 34:
			return "G3-D"
		}
	}
	return ""
}

func sbasTrackName(blockType, trackType byte) string {
	if blockType == FreqL1 && trackType == 0 {
		return "CA"
	}
	if blockType == FreqL5 || trackType == 6 || trackType == 7 || trackType == 8 {
		switch trackType {
		case 6:
			return "L5-I"
		case 7:
			return "L5-Q"
		case 8:
			return "L5-IQ"
		}
	}
	return ""
}

func mssTrackName(system, blockType, trackType byte) string {
	switch system {
	case SystemOmniStar:
		if blockType == FreqL1 || blockType == FreqS1 {
			return "MSS"
		}
	case SystemTerralite:
		if blockType == FreqXPS || trackType == 35 {
			return "XPS"
		}
		if blockType == FreqL1 || blockType == FreqS1 {
			return "MSS"
		}
	}
	return ""
}

// TrackTypeHint returns a longer tooltip for non-obvious signal types.
func TrackTypeHint(trackType byte) string {
	switch trackType {
	case 20:
		return "BOC(1,1) Pilot & Data — Galileo E1 / GPS L1C / BDS-III B1C"
	case 23:
		return "MBOC(1,1) Pilot & Data — Galileo E1 / GPS L1C / BDS-III B1C"
	case 14:
		return "AltBOC-Comp-PD — Galileo Component Mode AltBOC Pilot and Data"
	case 26:
		return "Galileo E6 BPSK(5) — Pilot and Data"
	case 36:
		return "Galileo E6 BPSK(5) — Pilot and Data"
	case 37:
		return "Galileo E6 BPSK(5) — Pilot"
	case 38:
		return "Galileo E6 BPSK(5) — Data"
	case 29:
		return "BeiDou B3 BPSK(10)"
	default:
		return ""
	}
}

// AppendSignal adds a signal to a band column, replacing same block+track if SNR is higher.
func AppendSignal(band *BandView, sig SignalView) {
	band.Present = true
	for i, existing := range band.Signals {
		if existing.BlockType == sig.BlockType && existing.TrackType == sig.TrackType {
			if sig.SNR >= existing.SNR {
				band.Signals[i] = sig
			}
			return
		}
	}
	band.Signals = append(band.Signals, sig)
}

// MaxSNR returns the highest SNR in the band, or -1 if empty.
func (b BandView) MaxSNR() float64 {
	max := -1.0
	for _, s := range b.Signals {
		if s.SNR > max {
			max = s.SNR
		}
	}
	return max
}
