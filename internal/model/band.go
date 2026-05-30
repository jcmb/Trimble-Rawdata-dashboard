package model

// Frequency block types (pydcollib GNSS.py).
const (
	FreqL1    = 0
	FreqL2    = 1
	FreqL5    = 2
	FreqE5B   = 3
	FreqE5AB  = 4
	FreqE6    = 5
	FreqB1    = 6
	FreqB3    = 7
	FreqE1    = 8
	FreqG3    = 9
	FreqXPS   = 10
	FreqS1    = 11
)

// Band column indices.
const (
	BandL1 = 1
	BandL2 = 2
	BandL5 = 3
	BandL6 = 4
)

// BandSlot maps RT27 measurement to L1/L2/L5/L6 column using pydcollib GetSNRs rules.
func BandSlot(system, blockType, trackType byte) int {
	switch system {
	case SystemGPS, SystemQZSS:
		switch blockType {
		case FreqL1, FreqE1, FreqS1:
			return BandL1
		case FreqL2:
			return BandL2
		default:
			return BandL5
		}
	case SystemGLONASS:
		switch blockType {
		case FreqL1:
			return BandL1
		case FreqL2:
			return BandL2
		default:
			// G3 and any other GLONASS third-band signals (pydcollib GetSNRs → Signals[2] / L5).
			return BandL5
		}
	case SystemGalileo:
		switch blockType {
		case FreqL1, FreqE1:
			return BandL1
		case FreqE6:
			return BandL6
		default:
			return BandL5
		}
	case SystemBeidouOld, SystemBeidou, SystemBeidouB1Geo:
		switch blockType {
		case FreqB1:
			return BandL1
		case FreqB3:
			return BandL6
		default:
			return BandL5
		}
	case SystemSBAS:
		if trackType == 6 || trackType == 7 || trackType == 8 {
			return BandL5
		}
		switch blockType {
		case FreqL1, FreqS1:
			return BandL1
		default:
			return BandL5
		}
	case SystemOmniStar, SystemTerralite:
		switch blockType {
		case FreqL1, FreqS1:
			return BandL1
		default:
			// Terralite XPS and any other MSS third-band signals → L5 (pydcollib GetSNRs).
			return BandL5
		}
	default:
		return bandSlotGeneric(blockType, trackType)
	}
}

func bandSlotGeneric(blockType, trackType byte) int {
	switch trackType {
	case 6, 7, 8:
		return BandL5
	case 29:
		return BandL6
	case 31, 36, 37, 38:
		return BandL6
	case 26:
		if blockType == FreqB1 {
			return BandL1
		}
		return BandL6
	}
	switch blockType {
	case FreqL1, FreqB1, FreqE1, FreqS1:
		return BandL1
	case FreqL2, FreqE5B:
		return BandL2
	case FreqL5, FreqE5AB, FreqXPS:
		return BandL5
	case FreqE6, FreqB3:
		return BandL6
	default:
		return 0
	}
}
