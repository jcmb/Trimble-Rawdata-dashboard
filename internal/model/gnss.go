package model

import "bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/gnss"

func SystemName(sys byte) string {
	switch sys {
	case gnss.SystemGPS:
		return "GPS"
	case gnss.SystemSBAS:
		return "SBAS"
	case gnss.SystemGLONASS:
		return "GLONASS"
	case gnss.SystemGalileo:
		return "Galileo"
	case gnss.SystemQZSS:
		return "QZSS"
	case gnss.SystemBeidou:
		return "Beidou"
	case gnss.SystemIRNSS:
		return "IRNSS"
	default:
		return "?"
	}
}
