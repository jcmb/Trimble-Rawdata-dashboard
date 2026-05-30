package model

import "fmt"

// RT27 GNSS system IDs (pydcollib GNSS.py).
const (
	SystemGPS         byte = 0
	SystemSBAS        byte = 1
	SystemGLONASS     byte = 2
	SystemGalileo     byte = 3
	SystemQZSS        byte = 4
	SystemBeidouOld   byte = 5
	SystemOmniStar    byte = 6
	SystemBeidou      byte = 7
	SystemTerralite   byte = 8
	SystemIRNSS       byte = 9
	SystemBeidouB1Geo byte = 10
)

func SystemName(sys byte) string {
	switch sys {
	case SystemGPS:
		return "GPS"
	case SystemSBAS:
		return "SBAS"
	case SystemGLONASS:
		return "GLONASS"
	case SystemGalileo:
		return "Galileo"
	case SystemQZSS:
		return "QZSS"
	case SystemBeidouOld, SystemBeidou, SystemBeidouB1Geo:
		return "Beidou"
	case SystemOmniStar:
		return "OmniSTAR"
	case SystemTerralite:
		return "Terralite"
	case SystemIRNSS:
		return "IRNSS"
	default:
		return fmt.Sprintf("unknown (%d)", sys)
	}
}
