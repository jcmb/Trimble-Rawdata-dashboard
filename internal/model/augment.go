package model

import "fmt"

// AugmentationName maps enhanced-position augmentation type to a label (pydcollib GNSS.pos_augment).
func AugmentationName(aug byte) string {
	switch aug {
	case 0:
		return "Autonomous"
	case 1:
		return "DGPS"
	case 2:
		return "SBAS"
	case 3:
		return "RTK Float"
	case 4:
		return "RTK Fixed"
	case 5:
		return "Wide RTK Float"
	case 6:
		return "Wide RTK Fixed"
	case 7:
		return "OmniSTAR"
	case 8:
		return "CDGPS"
	case 9:
		return "Autonomous KF"
	case 10:
		return "DGPS KF"
	case 11:
		return "SBAS KF"
	case 12:
		return "CDGPS KF"
	case 13:
		return "SBAS+ KF"
	case 14:
		return "SBAS+ CDGPS KF"
	case 15:
		return "RTX Std"
	case 16:
		return "XPS HC"
	case 17:
		return "XPS HF"
	case 18:
		return "Autonomous INS"
	case 19:
		return "DGNSS INS"
	case 20:
		return "RTK INS"
	case 21:
		return "SBAS INS"
	case 26:
		return "RTX Code"
	case 27:
		return "RTX Fast"
	case 29:
		return "RTX Lite"
	case 30:
		return "RTX Lite L1"
	case 36:
		return "SLAS"
	case 37:
		return "SLAS KF"
	default:
		return fmt.Sprintf("unknown (%d)", aug)
	}
}
