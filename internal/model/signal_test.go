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

func TestBandSlotGLONASSG3(t *testing.T) {
	if got := model.BandSlot(model.SystemGLONASS, model.FreqG3, 32); got != model.BandL5 {
		t.Fatalf("GLONASS G3: got %d want L5", got)
	}
}

func TestTrackType20(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGPS, 0, 20); got != "L1C" {
		t.Fatalf("GPS track 20: got %q want L1C", got)
	}
	if got := model.TrackTypeName(model.SystemBeidou, model.FreqB1, 20); got != "B1C" {
		t.Fatalf("BeiDou track 20: got %q want B1C", got)
	}
	if got := model.TrackTypeName(model.SystemGalileo, model.FreqE1, 20); got != "E1" {
		t.Fatalf("Galileo E1 track 20: got %q want E1", got)
	}
	if got := model.TrackTypeHint(20); got == "" {
		t.Fatal("track 20 hint expected")
	}
}

func TestBandSlotTrackType20AlwaysL1(t *testing.T) {
	// Block types that would otherwise land in L5/L6 must still use L1 for mode 20.
	cases := []struct {
		name      string
		system    byte
		blockType byte
	}{
		{"GPS wrong block", model.SystemGPS, model.FreqL5},
		{"Galileo wrong block", model.SystemGalileo, model.FreqL5},
		{"BeiDou B3 block", model.SystemBeidou, model.FreqB3},
		{"QZSS", model.SystemQZSS, model.FreqL2},
	}
	for _, tc := range cases {
		if got := model.BandSlot(tc.system, tc.blockType, 20); got != model.BandL1 {
			t.Fatalf("%s: BandSlot got %d want L1", tc.name, got)
		}
	}
}

func TestTrackTypeL2C(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGPS, model.FreqL2, 3); got != "L2CM" {
		t.Fatalf("track 3: got %q want L2CM", got)
	}
	if got := model.TrackTypeName(model.SystemGPS, model.FreqL2, 4); got != "L2CL" {
		t.Fatalf("track 4: got %q want L2CL", got)
	}
}

func TestTrackTypeBeidouB2(t *testing.T) {
	if got := model.TrackTypeName(model.SystemBeidou, model.FreqB1, 6); got != "B2B" {
		t.Fatalf("BDS track 6: got %q want B2B", got)
	}
	if got := model.TrackTypeName(model.SystemBeidou, model.FreqB1, 8); got != "B2A" {
		t.Fatalf("BDS track 8: got %q want B2A", got)
	}
	if got := model.BandSlot(model.SystemBeidou, model.FreqB1, 6); got != model.BandL5 {
		t.Fatalf("BDS track 6 band: got %d want L5", got)
	}
	if got := model.BandSlot(model.SystemBeidou, model.FreqB3, 8); got != model.BandL5 {
		t.Fatalf("BDS track 8 band: got %d want L5", got)
	}
}

func TestTrackTypeGPSL5Unchanged(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGPS, model.FreqL5, 6); got != "L5-I" {
		t.Fatalf("GPS track 6: got %q want L5-I", got)
	}
	if got := model.TrackTypeName(model.SystemGPS, model.FreqL5, 8); got != "L5-IQ" {
		t.Fatalf("GPS track 8: got %q want L5-IQ", got)
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

func TestGlonassG3Track32(t *testing.T) {
	if got := model.TrackTypeName(model.SystemGLONASS, model.FreqG3, 32); got != "G3-D+P" {
		t.Fatalf("G3 track 32: got %q want G3-D+P", got)
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

func TestSystemNameMSS(t *testing.T) {
	if got := model.SystemName(model.SystemOmniStar); got != "MSS" {
		t.Fatalf("OmniStar: got %q", got)
	}
	if got := model.SystemName(model.SystemTerralite); got != "MSS" {
		t.Fatalf("Terralite: got %q", got)
	}
}

func TestMSSBandAndTrack(t *testing.T) {
	if got := model.BandSlot(model.SystemOmniStar, model.FreqL1, 0); got != model.BandL1 {
		t.Fatalf("OmniStar L1 band: got %d", got)
	}
	if got := model.TrackTypeName(model.SystemOmniStar, model.FreqL1, 0); got != "MSS" {
		t.Fatalf("OmniStar track: got %q", got)
	}
	if got := model.BandSlot(model.SystemTerralite, model.FreqXPS, 35); got != model.BandL5 {
		t.Fatalf("Terralite XPS band: got %d", got)
	}
	if got := model.TrackTypeName(model.SystemTerralite, model.FreqXPS, 35); got != "XPS" {
		t.Fatalf("Terralite XPS track: got %q", got)
	}
}
