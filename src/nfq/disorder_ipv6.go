package nfq

import (
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
	"github.com/daniellavrushin/b4/utils"
)

// sendDisorderFragmentsV6 - IPv6 version: splits and sends in random order
func (w *Worker) sendDisorderFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	disorder := &cfg.Fragmentation.Disorder

	pi, ok := ExtractPacketInfoV6(packet)
	if !ok || pi.PayloadLen < 10 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	splits := GetSNISplitPoints(pi.Payload, pi.PayloadLen, cfg.Fragmentation.MiddleSNI, 0)
	if len(splits) == 0 {
		splits = []int{1, pi.PayloadLen / 2, pi.PayloadLen * 3 / 4}
	}

	validSplits := BuildValidSplits(splits, pi.PayloadLen)

	segments := make([]Segment, 0, len(validSplits)-1)
	for i := 0; i < len(validSplits)-1; i++ {
		start, end := validSplits[i], validSplits[i+1]
		seg := BuildSegmentV6(packet, pi, pi.Payload[start:end], uint32(start))
		if i < len(validSplits)-2 {
			ClearPSH(seg, pi.IPHdrLen)
			sock.FixTCPChecksumV6(seg)
		}
		segments = append(segments, Segment{Data: seg, Seq: pi.Seq0 + uint32(start)})
	}

	r := utils.NewRand()
	ShuffleSegments(segments, disorder.ShuffleMode, r)

	minJitter, maxJitter := GetDisorderJitter(disorder)

	seg2d := cfg.TCP.Seg2Delay
	for i, seg := range segments {
		_ = w.sock.SendIPv6(seg.Data, dst)
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
