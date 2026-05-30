package model_test

import (
	"testing"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
)

func TestPositionSVFlags(t *testing.T) {
	used, raim, unhealthy := model.PositionSVFlags(model.PosSVFlagUsed | model.PosSVFlagRAIMFault)
	if !used || !raim || unhealthy {
		t.Fatalf("got used=%v raim=%v unhealthy=%v", used, raim, unhealthy)
	}
}

func TestHorizontalSigma(t *testing.T) {
	got := model.HorizontalSigma(3, 4)
	if got != 5 {
		t.Fatalf("got %v want 5", got)
	}
}
