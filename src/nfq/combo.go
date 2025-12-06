package nfq

import (
	"encoding/binary"
	"math/rand"
	"net"
	"sort"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// sendComboFragments combines multiple evasion techniques
// Strategy: split at multiple points + send out of order + optional delay
func (w *Worker) sendComboFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen < 20 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	payload := packet[payloadStart:]
	seq0 := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	id0 := binary.BigEndian.Uint16(packet[4:6])

	combo := &cfg.Fragmentation.Combo
	splits := []int{}

	// First-byte split
	if combo.FirstByteSplit {
		splits = append(splits, 1)
	}

	// Extension split (before SNI extension)
	if combo.ExtensionSplit {
		if extSplit := findPreSNIExtensionPoint(payload); extSplit > 1 && extSplit < payloadLen-5 {
			splits = append(splits, extSplit)
		}
	}

	// SNI split (middle of hostname)
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(payload); ok && sniEnd > sniStart {
			sniLen := sniEnd - sniStart
			if sniStart > 2 {
				splits = append(splits, sniStart-1)
			}
			splits = append(splits, sniStart+sniLen/2)
			if sniLen > 15 {
				splits = append(splits, sniStart+sniLen*3/4)
			}
		}
	}

	splits = uniqueSorted(splits, payloadLen)

	if len(splits) < 1 {
		splits = []int{payloadLen / 2}
	}

	type segment struct {
		data []byte
		seq  uint32
	}

	segments := make([]segment, 0, len(splits)+1)
	prevEnd := 0

	for i, splitPos := range splits {
		if splitPos <= prevEnd {
			continue
		}

		segDataLen := splitPos - prevEnd
		segLen := payloadStart + segDataLen
		seg := make([]byte, segLen)
		copy(seg[:payloadStart], packet[:payloadStart])
		copy(seg[payloadStart:], payload[prevEnd:splitPos])

		binary.BigEndian.PutUint32(seg[ipHdrLen+4:ipHdrLen+8], seq0+uint32(prevEnd))
		binary.BigEndian.PutUint16(seg[4:6], id0+uint16(i))
		binary.BigEndian.PutUint16(seg[2:4], uint16(segLen))

		seg[ipHdrLen+13] &^= 0x08

		sock.FixIPv4Checksum(seg[:ipHdrLen])
		sock.FixTCPChecksum(seg)

		segments = append(segments, segment{data: seg, seq: seq0 + uint32(prevEnd)})
		prevEnd = splitPos
	}

	if prevEnd < payloadLen {
		segLen := payloadStart + (payloadLen - prevEnd)
		seg := make([]byte, segLen)
		copy(seg[:payloadStart], packet[:payloadStart])
		copy(seg[payloadStart:], payload[prevEnd:])

		binary.BigEndian.PutUint32(seg[ipHdrLen+4:ipHdrLen+8], seq0+uint32(prevEnd))
		binary.BigEndian.PutUint16(seg[4:6], id0+uint16(len(segments)))
		binary.BigEndian.PutUint16(seg[2:4], uint16(segLen))

		sock.FixIPv4Checksum(seg[:ipHdrLen])
		sock.FixTCPChecksum(seg)

		segments = append(segments, segment{data: seg, seq: seq0 + uint32(prevEnd)})
	}

	if len(segments) == 0 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	// Thread-safe RNG
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Apply shuffle mode
	switch combo.ShuffleMode {
	case "full":
		// Shuffle all segments
		for i := len(segments) - 1; i > 0; i-- {
			j := r.Intn(i + 1)
			segments[i], segments[j] = segments[j], segments[i]
		}
	case "reverse":
		// Reverse order
		for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
			segments[i], segments[j] = segments[j], segments[i]
		}
	default: // "middle"
		// Shuffle only middle segments, keep first and last in place
		if len(segments) > 3 {
			middle := segments[1 : len(segments)-1]
			for i := len(middle) - 1; i > 0; i-- {
				j := r.Intn(i + 1)
				middle[i], middle[j] = middle[j], middle[i]
			}
		} else if len(segments) > 1 {
			for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
				segments[i], segments[j] = segments[j], segments[i]
			}
		}
	}

	// Set PSH flag on highest-sequence segment
	maxSeqIdx := 0
	for i := range segments {
		segIpHdrLen := int((segments[i].data[0] & 0x0F) * 4)
		segments[i].data[segIpHdrLen+13] &^= 0x08
		sock.FixTCPChecksum(segments[i].data)
		if segments[i].seq > segments[maxSeqIdx].seq {
			maxSeqIdx = i
		}
	}
	segIpHdrLen := int((segments[maxSeqIdx].data[0] & 0x0F) * 4)
	segments[maxSeqIdx].data[segIpHdrLen+13] |= 0x08
	sock.FixTCPChecksum(segments[maxSeqIdx].data)

	// Send with delays
	firstDelayMs := combo.FirstDelayMs
	if firstDelayMs <= 0 {
		firstDelayMs = 100
	}
	jitterMaxUs := combo.JitterMaxUs
	if jitterMaxUs <= 0 {
		jitterMaxUs = 2000
	}

	for i, seg := range segments {
		_ = w.sock.SendIPv4(seg.data, dst)

		if i == 0 {
			jitter := r.Intn(firstDelayMs/3 + 1)
			time.Sleep(time.Duration(firstDelayMs+jitter) * time.Millisecond)
		} else if i < len(segments)-1 {
			time.Sleep(time.Duration(r.Intn(jitterMaxUs)) * time.Microsecond)
		}
	}
}

func uniqueSorted(splits []int, maxVal int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0, len(splits))

	for _, s := range splits {
		if s > 0 && s < maxVal && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	sort.Ints(result)
	return result
}
