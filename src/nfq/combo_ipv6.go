package nfq

import (
	"encoding/binary"
	"math/rand"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// sendComboFragmentsV6 - IPv6 version: combines multiple evasion techniques
func (w *Worker) sendComboFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	const ipv6HdrLen = 40

	if len(packet) < ipv6HdrLen+20 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen < 20 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	payload := packet[payloadStart:]
	seq0 := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])

	combo := &cfg.Fragmentation.Combo
	splits := []int{}

	if combo.FirstByteSplit {
		splits = append(splits, 1)
	}

	if combo.ExtensionSplit {
		if extSplit := findPreSNIExtensionPoint(payload); extSplit > 1 && extSplit < payloadLen-5 {
			splits = append(splits, extSplit)
		}
	}

	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(payload); ok && sniEnd > sniStart {
			midSNI := sniStart + (sniEnd-sniStart)/2
			if len(splits) == 0 || midSNI > splits[len(splits)-1]+2 {
				splits = append(splits, midSNI)
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

	for _, splitPos := range splits {
		if splitPos <= prevEnd {
			continue
		}

		segDataLen := splitPos - prevEnd
		segLen := payloadStart + segDataLen
		seg := make([]byte, segLen)
		copy(seg[:payloadStart], packet[:payloadStart])
		copy(seg[payloadStart:], payload[prevEnd:splitPos])

		binary.BigEndian.PutUint32(seg[ipv6HdrLen+4:ipv6HdrLen+8], seq0+uint32(prevEnd))
		binary.BigEndian.PutUint16(seg[4:6], uint16(segLen-ipv6HdrLen))

		seg[ipv6HdrLen+13] &^= 0x08
		sock.FixTCPChecksumV6(seg)

		segments = append(segments, segment{data: seg, seq: seq0 + uint32(prevEnd)})
		prevEnd = splitPos
	}

	if prevEnd < payloadLen {
		segLen := payloadStart + (payloadLen - prevEnd)
		seg := make([]byte, segLen)
		copy(seg[:payloadStart], packet[:payloadStart])
		copy(seg[payloadStart:], payload[prevEnd:])

		binary.BigEndian.PutUint32(seg[ipv6HdrLen+4:ipv6HdrLen+8], seq0+uint32(prevEnd))
		binary.BigEndian.PutUint16(seg[4:6], uint16(segLen-ipv6HdrLen))

		sock.FixTCPChecksumV6(seg)
		segments = append(segments, segment{data: seg, seq: seq0 + uint32(prevEnd)})
	}

	if len(segments) == 0 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch combo.ShuffleMode {
	case "full":
		for i := len(segments) - 1; i > 0; i-- {
			j := r.Intn(i + 1)
			segments[i], segments[j] = segments[j], segments[i]
		}
	case "reverse":
		for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
			segments[i], segments[j] = segments[j], segments[i]
		}
	default:
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

	maxSeqIdx := 0
	for i := range segments {
		segments[i].data[ipv6HdrLen+13] &^= 0x08
		sock.FixTCPChecksumV6(segments[i].data)
		if segments[i].seq > segments[maxSeqIdx].seq {
			maxSeqIdx = i
		}
	}
	segments[maxSeqIdx].data[ipv6HdrLen+13] |= 0x08
	sock.FixTCPChecksumV6(segments[maxSeqIdx].data)

	firstDelayMs := combo.FirstDelayMs
	if firstDelayMs <= 0 {
		firstDelayMs = 100
	}
	jitterMaxUs := combo.JitterMaxUs
	if jitterMaxUs <= 0 {
		jitterMaxUs = 2000
	}

	for i, seg := range segments {
		_ = w.sock.SendIPv6(seg.data, dst)

		if i == 0 {
			jitter := r.Intn(firstDelayMs/3 + 1)
			time.Sleep(time.Duration(firstDelayMs+jitter) * time.Millisecond)
		} else if i < len(segments)-1 {
			time.Sleep(time.Duration(r.Intn(jitterMaxUs)) * time.Microsecond)
		}
	}
}
