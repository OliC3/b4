package nfq

import (
	"encoding/binary"
	"math/rand"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// sendDisorderFragments - splits and sends in random order without any faking
// DPI expects sequential data; this exploits that assumption
func (w *Worker) sendDisorderFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen < 10 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	payload := packet[payloadStart:]
	seq0 := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	id0 := binary.BigEndian.Uint16(packet[4:6])

	disorder := &cfg.Fragmentation.Disorder

	var splits []int

	// Use middle_sni setting to determine split strategy
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(payload); ok && sniEnd > sniStart {
			sniLen := sniEnd - sniStart
			splits = append(splits, sniStart)
			if sniLen > 6 {
				splits = append(splits, sniStart+sniLen/2)
			}
			splits = append(splits, sniEnd)
		}
	}

	// Fallback if no SNI or middle_sni disabled
	if len(splits) == 0 {
		splits = []int{1, payloadLen / 2, payloadLen * 3 / 4}
	}

	validSplits := []int{0}
	for _, s := range splits {
		if s > 0 && s < payloadLen {
			validSplits = append(validSplits, s)
		}
	}
	validSplits = append(validSplits, payloadLen)

	type segment struct {
		data   []byte
		seqOff uint32
		order  int
	}

	segments := make([]segment, 0, len(validSplits)-1)
	for i := 0; i < len(validSplits)-1; i++ {
		start := validSplits[i]
		end := validSplits[i+1]

		segLen := payloadStart + (end - start)
		seg := make([]byte, segLen)
		copy(seg[:payloadStart], packet[:payloadStart])
		copy(seg[payloadStart:], payload[start:end])

		binary.BigEndian.PutUint32(seg[ipHdrLen+4:ipHdrLen+8], seq0+uint32(start))
		binary.BigEndian.PutUint16(seg[4:6], id0+uint16(i))
		binary.BigEndian.PutUint16(seg[2:4], uint16(segLen))

		if i < len(validSplits)-2 {
			seg[ipHdrLen+13] &^= 0x08 // Clear PSH
		}

		sock.FixIPv4Checksum(seg[:ipHdrLen])
		sock.FixTCPChecksum(seg)

		segments = append(segments, segment{data: seg, seqOff: uint32(start), order: i})
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Apply shuffle mode
	switch disorder.ShuffleMode {
	case "reverse":
		for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
			segments[i], segments[j] = segments[j], segments[i]
		}
	default: // "full"
		if len(segments) > 1 {
			for i := len(segments) - 1; i > 0; i-- {
				j := r.Intn(i + 1)
				segments[i], segments[j] = segments[j], segments[i]
			}
		}
	}

	// Timing settings
	minJitter := disorder.MinJitterUs
	maxJitter := disorder.MaxJitterUs
	if minJitter <= 0 {
		minJitter = 1000
	}
	if maxJitter <= minJitter {
		maxJitter = minJitter + 2000
	}

	seg2d := cfg.TCP.Seg2Delay
	for i, seg := range segments {
		_ = w.sock.SendIPv4(seg.data, dst)
		if i < len(segments)-1 {
			if seg2d > 0 {
				jitter := r.Intn(seg2d/2 + 1)
				time.Sleep(time.Duration(seg2d+jitter) * time.Millisecond)
			} else {
				jitter := minJitter + r.Intn(maxJitter-minJitter+1)
				time.Sleep(time.Duration(jitter) * time.Microsecond)
			}
		}
	}
}
