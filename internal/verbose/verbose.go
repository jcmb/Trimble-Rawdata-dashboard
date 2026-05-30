// Package verbose controls leveled debug logging for receiver ingest.
package verbose

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/dcol"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/gsof"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/packet"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/packet/genout"
	praw "bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/packet/rawdata"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/rawdata"
	"bitbucket.trimble.tools/gnsstl/geoffrey-kirk-go-dcol/session"
)

// Level controls how much is logged to stderr.
type Level int

const (
	Off Level = iota
	Info
	Debug
	Trace
)

// ParseLevel accepts off, info, debug, trace (or 0–3).
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off", "0":
		return Off, nil
	case "info", "1":
		return Info, nil
	case "debug", "2":
		return Debug, nil
	case "trace", "3":
		return Trace, nil
	default:
		return Off, fmt.Errorf("verbose: unknown level %q (use off, info, debug, trace)", s)
	}
}

func (l Level) String() string {
	switch l {
	case Info:
		return "info"
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	default:
		return "off"
	}
}

// Logger emits leveled diagnostics and tracks link statistics.
type Logger struct {
	Level Level

	bytesRead     atomic.Uint64
	frames        atomic.Uint64
	packets       atomic.Uint64
	rawPages      atomic.Uint64
	rawMessages   atomic.Uint64
	genoutPages   atomic.Uint64
	gsofRecords   atomic.Uint64
	decodeErrors  atomic.Uint64
	rawPageErrors atomic.Uint64
	otherPackets  atomic.Uint64
}

func New(level Level) *Logger {
	return &Logger{Level: level}
}

// Stats is a point-in-time summary for periodic info logging.
type Stats struct {
	BytesRead      uint64
	Frames         uint64
	Packets        uint64
	DCOL0x57Pages  uint64 // RAWDATA (57h) pages
	DCOL0x40Pages  uint64 // GSOF (40h) GENOUT pages
	RAWMessages    uint64 // reassembled RAWDATA subtypes 6/7
	GSOFRecords    uint64
	DecodeErrors   uint64
	RAWPageErrors  uint64
	OtherDCOLTypes uint64
}

func (l *Logger) Snapshot() Stats {
	return Stats{
		BytesRead:      l.bytesRead.Load(),
		Frames:         l.frames.Load(),
		Packets:        l.packets.Load(),
		DCOL0x57Pages:  l.rawPages.Load(),
		DCOL0x40Pages:  l.genoutPages.Load(),
		RAWMessages:    l.rawMessages.Load(),
		GSOFRecords:    l.gsofRecords.Load(),
		DecodeErrors:   l.decodeErrors.Load(),
		RAWPageErrors:  l.rawPageErrors.Load(),
		OtherDCOLTypes: l.otherPackets.Load(),
	}
}

// LogStats writes a summary line at info level or above.
func (l *Logger) LogStats(port string) {
	if l.Level < Info {
		return
	}
	s := l.Snapshot()
	slog.Info("link stats",
		"port", port,
		"bytes", s.BytesRead,
		"frames", s.Frames,
		"dcol_0x57", s.DCOL0x57Pages,
		"dcol_0x40", s.DCOL0x40Pages,
		"raw_rt27_pos", s.RAWMessages,
		"gsof_records", s.GSOFRecords,
		"other_dcol", s.OtherDCOLTypes,
		"decode_errors", s.DecodeErrors,
		"raw_errors", s.RAWPageErrors,
	)
	if s.BytesRead > 0 && s.Frames == 0 {
		slog.Warn("hint: bytes received but no DCOL frames decoded; check baud rate, port, or framing")
	}
	if s.DCOL0x57Pages > 0 && s.RAWMessages == 0 {
		slog.Info("hint: DCOL 0x57 RAWDATA pages arrive but none are subtypes 6 (RT27) or 7 (position); this stream may use other RAWDATA record types (e.g. 12)")
	}
	if s.DCOL0x40Pages > 0 && s.DCOL0x57Pages == 0 {
		slog.Info("hint: stream has DCOL 0x40 (GSOF) but no 0x57 (RAWDATA); dashboard needs RAWDATA subtypes 6/7")
	}
}

// SessionConfig returns session callbacks wired to this logger.
func (l *Logger) SessionConfig(base session.Config) session.Config {
	if l.Level >= Info {
		base.OnBytesRead = l.onBytes
		base.OnFrame = l.onFrame
	}
	if l.Level >= Trace {
		base.DecoderLog = l.decoderLog
	}
	if l.Level >= Debug {
		base.RAWLog = l.rawLog
		base.OnDecodeError = l.onDecodeError
		base.OnRAWPageError = l.onRAWPageError
		base.OnGSOF = l.onGSOF
	}
	if l.Level >= Info {
		userPacket := base.OnPacket
		userRAW := base.OnRAWDATA
		base.OnPacket = func(p packet.Packet) {
			l.onPacketCount(p)
			if userPacket != nil {
				userPacket(p)
			}
		}
		base.OnRAWDATA = func(msg rawdata.Message) {
			l.rawMessages.Add(1)
			if l.Level >= Debug {
				slog.Debug("RAWDATA assembled", "msg", msg)
			}
			if userRAW != nil {
				userRAW(msg)
			}
		}
	} else {
		// Off: pass through unchanged except trace hooks stay nil.
	}
	return base
}

func (l *Logger) onBytes(data []byte) {
	l.bytesRead.Add(uint64(len(data)))
	if l.Level >= Trace {
		const max = 64
		shown := data
		if len(shown) > max {
			shown = shown[:max]
		}
		slog.Debug("RX bytes", "n", len(data), "hex", hex.EncodeToString(shown), "truncated", len(data) > max)
	}
}

func (l *Logger) decoderLog(msg string) {
	if l.Level >= Trace {
		slog.Debug("DCOL decoder", "msg", msg)
	}
}

func (l *Logger) onFrame(frame []byte) {
	l.frames.Add(1)
	if l.Level >= Trace {
		const max = 128
		shown := frame
		if len(shown) > max {
			shown = shown[:max]
		}
		slog.Debug("DCOL frame", "n", len(frame), "hex", hex.EncodeToString(shown), "truncated", len(frame) > max)
	}
}

func (l *Logger) onPacketCount(p packet.Packet) {
	l.packets.Add(1)
	switch pkt := p.(type) {
	case *praw.Page:
		l.rawPages.Add(1)
		if l.Level >= Debug {
			slog.Debug("DCOL packet", "msg", "0x57 RAWDATA", "detail", pkt.String())
			if !praw.SupportedRecordType(pkt.RecType) {
				slog.Debug("DCOL 0x57 record type not displayed", "rec_type", pkt.RecType, "supported", "6=RT27, 7=position")
			}
		}
	case *genout.Packet:
		l.genoutPages.Add(1)
		if l.Level >= Debug {
			slog.Debug("DCOL packet", "msg", "0x40 GSOF", "detail", pkt.String())
		}
	default:
		l.otherPackets.Add(1)
		if l.Level >= Debug {
			key := p.Key()
			slog.Debug("DCOL packet", "msg", fmt.Sprintf("0x%02X", key.ID), "detail", p)
		}
	}
}

func (l *Logger) onGSOF(rec gsof.Record) {
	l.gsofRecords.Add(1)
	if l.Level >= Debug {
		slog.Debug("GSOF record", "type", rec.Type, "len", len(rec.Data))
	}
}

func (l *Logger) rawLog(msg string) {
	if l.Level >= Debug {
		slog.Debug("RAWDATA expander", "msg", msg)
	}
}

func (l *Logger) onDecodeError(frame []byte, err error) {
	l.decodeErrors.Add(1)
	if l.Level >= Debug {
		slog.Warn("frame decode failed", "err", err, "len", len(frame))
	}
}

func (l *Logger) onRAWPageError(page *praw.Page, err error) {
	l.rawPageErrors.Add(1)
	if l.Level >= Debug {
		slog.Warn("RAWDATA page error", "page", page.String(), "err", err)
	}
}

// PacketTypeName returns a short label for a DCOL wire type byte.
func PacketTypeName(id byte) string {
	switch id {
	case dcol.RAWDATA:
		return "0x57 RAWDATA"
	case dcol.GSOF:
		return "0x40 GSOF"
	case dcol.RETSVDATA:
		return "0x55 RETSVDATA"
	default:
		return fmt.Sprintf("0x%02X", id)
	}
}
