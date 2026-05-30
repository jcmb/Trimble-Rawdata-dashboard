package model

import "bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/dcol"

// MeasCycleSlipNow reports RT27_MEAS_FLAG1_CycleSlip on the first meas flag byte.
func MeasCycleSlipNow(flags []byte) bool {
	return len(flags) > 0 && flags[0]&dcol.MeasFlag1CycleSlip != 0
}
