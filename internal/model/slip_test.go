package model_test

import (
	"testing"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
)

func TestMeasCycleSlipNow(t *testing.T) {
	if !model.MeasCycleSlipNow([]byte{0x08}) {
		t.Fatal("expected cycle slip flag")
	}
	if model.MeasCycleSlipNow([]byte{0x00}) {
		t.Fatal("expected no slip")
	}
}
