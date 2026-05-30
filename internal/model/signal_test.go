package model_test

import (
	"testing"

	"github.com/gkirk/trimble-rawdata-dashboard/internal/model"
)

func TestBandSlotBeidou(t *testing.T) {
	if got := model.BandSlot(model.SystemBeidou, model.FreqB1, 26); got != model.BandL1 {
		t.Fatalf("B1: got %d want L1", got)
	}
	if got := model.BandSlot(model.SystemBeidou, model.FreqB3, 29); got != model.BandL6 {
		t.Fatalf("B3: got %d want L6", got)
	}
}

func TestBandSlotSBASL5(t *testing.T) {
	if got := model.BandSlot(model.SystemSBAS, model.FreqL5, 6); got != model.BandL5 {
		t.Fatalf("SBAS L5 block: got %d want L5", got)
	}
	if got := model.BandSlot(model.SystemSBAS, model.FreqL1, 8); got != model.BandL5 {
		t.Fatalf("SBAS L5-IQ on L1 index: got %d want L5", got)
	}
}

func TestBandSlotGalileoE6(t *testing.T) {
	if got := model.BandSlot(model.SystemGalileo, model.FreqE6, 26); got != model.BandL6 {
		t.Fatalf("Galileo E6: got %d want L6", got)
	}
}

func TestBandSlotGLONASSL2(t *testing.T) {
	if got := model.BandSlot(model.SystemGLONASS, model.FreqL2, 0); got != model.BandL2 {
		t.Fatalf("GLONASS L2: got %d want L2", got)
	}
}

func TestTrackType23(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGPS, 0, 23); got != "CBOC" {
		t.Fatalf("track 23: got %q", got)
	}
}

func TestTrackType14(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGalileo, model.FreqE5AB, 14); got != "E5Alt" {
		t.Fatalf("Galileo E5Alt: got %q", got)
	}
	if got := model.TrackTypeName(model.SystemGPS, model.FreqE5AB, 14); got != "AltBoc" {
		t.Fatalf("GPS AltBoc: got %q", got)
	}
}

func TestGlonassL2Names(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGLONASS, model.FreqL2, 1); got != "P" {
		t.Fatalf("G2P: got %q", got)
	}
	if got := model.TrackTypeName(model.SystemGLONASS, model.FreqL2, 0); got != "CA" {
		t.Fatalf("G2C: got %q", got)
	}
}

func TestAppendSignalMulti(t *testing.T) {
	var band model.BandView
	model.AppendSignal(&band, model.SignalView{BlockType: 1, SNR: 39, TrackType: 1, TrackName: "P"})
	model.AppendSignal(&band, model.SignalView{BlockType: 1, SNR: 40, TrackType: 0, TrackName: "CA"})
	if len(band.Signals) != 2 {
		t.Fatalf("want 2 signals, got %d", len(band.Signals))
	}
}

func TestAppendSignalGalileoL5(t *testing.T) {
	var band model.BandView
	model.AppendSignal(&band, model.SignalView{BlockType: model.FreqL5, SNR: 38, TrackType: 11, TrackName: "E5A"})
	model.AppendSignal(&band, model.SignalView{BlockType: model.FreqE5B, SNR: 36, TrackType: 11, TrackName: "E5B"})
	model.AppendSignal(&band, model.SignalView{BlockType: model.FreqE5AB, SNR: 34, TrackType: 14, TrackName: "E5Alt"})
	if len(band.Signals) != 3 {
		t.Fatalf("Galileo L5: want 3 signals, got %d", len(band.Signals))
	}
}

func TestGalileoTrackNames(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGalileo, model.FreqL5, 11); got != "E5A" {
		t.Fatalf("E5A: got %q", got)
	}
	if got := model.TrackTypeName(model.SystemGalileo, model.FreqE5B, 11); got != "E5B" {
		t.Fatalf("E5B: got %q", got)
	}
	if got := model.TrackTypeName(model.SystemGalileo, model.FreqE5AB, 14); got != "E5Alt" {
		t.Fatalf("E5Alt: got %q", got)
	}
}
