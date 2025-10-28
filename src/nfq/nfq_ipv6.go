package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
)

// dropAndInjectQUIV6 handles QUIC (UDP) packet manipulation for IPv6
func (w *Worker) dropAndInjectQUICV6(cfg *config.Config, raw []byte, dst net.IP) {
	if cfg.UDP.Mode != "fake" {
		return
	}

	// Send fake UDP packets
	if cfg.UDP.FakeSeqLength > 0 {
		for i := 0; i < cfg.UDP.FakeSeqLength; i++ {
			fake, ok := sock.BuildFakeUDPFromOriginalV6(raw, cfg.UDP.FakeLen, cfg.Faking.TTL)
			if ok {
				if cfg.UDP.FakingStrategy == "checksum" {
					// Corrupt the UDP checksum
					ipv6HdrLen := 40
					if len(fake) >= ipv6HdrLen+8 {
						fake[ipv6HdrLen+6] ^= 0xFF
						fake[ipv6HdrLen+7] ^= 0xFF
					}
				}
				_ = w.sock.SendIPv6(fake, dst)
				if cfg.Seg2Delay > 0 {
					time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
				} else {
					time.Sleep(1 * time.Millisecond)
				}
			}
		}
	}

	// Fragment and send real packet
	splitPos := 24
	frags, ok := sock.IPv6FragmentUDP(raw, splitPos)
	if !ok {
		_ = w.sock.SendIPv6(raw, dst)
		return
	}

	if cfg.Fragmentation.SNIReverse {
		_ = w.sock.SendIPv6(frags[0], dst)
		if cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(frags[1], dst)
	} else {
		_ = w.sock.SendIPv6(frags[1], dst)
		if cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(frags[0], dst)
	}
}

// dropAndInjectTCPv6 handles TCP packet manipulation for IPv6
func (w *Worker) dropAndInjectTCPv6(cfg *config.Config, raw []byte, dst net.IP, injectFake bool) {
	if len(raw) < 60 { // IPv6 header (40) + TCP header (20 min)
		_ = w.sock.SendIPv6(raw, dst)
		return
	}

	ipv6HdrLen := 40
	tcpHdrLen := int((raw[ipv6HdrLen+12] >> 4) * 4)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := len(raw) - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv6(raw, dst)
		return
	}

	// Inject fake SNI packets if configured
	if injectFake && cfg.Faking.SNI && cfg.Faking.SNISeqLength > 0 {
		w.sendFakeSNISequencev6(cfg, raw, dst)
	}

	// Apply fragmentation strategy
	switch cfg.Fragmentation.Strategy {
	case "tcp":
		w.sendTCPSegmentsv6(cfg, raw, dst)
	case "ip":
		w.sendIPFragmentsv6(cfg, raw, dst)
	case "none":
		_ = w.sock.SendIPv6(raw, dst)
	default:
		w.sendTCPSegmentsv6(cfg, raw, dst)
	}
}

// sendTCPSegmentsv6 sends TCP packet as two segments (TCP-level segmentation for IPv6)
func (w *Worker) sendTCPSegmentsv6(cfg *config.Config, packet []byte, dst net.IP) {
	ipv6HdrLen := 40
	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	totalLen := len(packet)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := totalLen - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	splitPos := cfg.Fragmentation.SNIPosition
	payload := packet[payloadStart:]

	// Find SNI position if middle split is enabled
	if cfg.Fragmentation.MiddleSNI {
		if s, e, ok := locateSNI(payload); ok && e-s >= 4 {
			log.Tracef("SNI found at %d..%d of %d", s, e, payloadLen)
			splitPos = s + (e-s)/2
		} else {
			if splitPos <= 0 || splitPos >= payloadLen {
				splitPos = 1
			}
		}
	}

	// Use IPv6 TCP segmentation
	segments, ok := sock.IPv6SendTCPSegments(packet, splitPos)
	if !ok {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	if cfg.Fragmentation.SNIReverse {
		_ = w.sock.SendIPv6(segments[1], dst)
		if cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(segments[0], dst)
	} else {
		_ = w.sock.SendIPv6(segments[0], dst)
		if cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(segments[1], dst)
	}
}

// sendIPFragmentsv6 sends packet as IPv6 fragments (IP-level fragmentation)
func (w *Worker) sendIPFragmentsv6(cfg *config.Config, packet []byte, dst net.IP) {
	splitPos := cfg.Fragmentation.SNIPosition
	if splitPos <= 0 || splitPos >= len(packet) {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	fragments, ok := sock.IPv6FragmentPacket(packet, splitPos)
	if !ok {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	if cfg.Fragmentation.SNIReverse {
		_ = w.sock.SendIPv6(fragments[1], dst)
		if cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(fragments[0], dst)
	} else {
		_ = w.sock.SendIPv6(fragments[0], dst)
		if cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(fragments[1], dst)
	}
}

// sendFakeSNISequencev6 sends a sequence of fake SNI packets for IPv6
func (w *Worker) sendFakeSNISequencev6(cfg *config.Config, original []byte, dst net.IP) {
	if !cfg.Faking.SNI || cfg.Faking.SNISeqLength <= 0 {
		return
	}

	fake := sock.BuildFakeSNIPacketV6(original, cfg)
	if fake == nil {
		return
	}

	ipv6HdrLen := 40

	for i := 0; i < cfg.Faking.SNISeqLength; i++ {
		_ = w.sock.SendIPv6(fake, dst)

		// Update for next iteration
		if i+1 < cfg.Faking.SNISeqLength {
			// Adjust sequence number for non-past/rand strategies
			if cfg.Faking.Strategy != "pastseq" && cfg.Faking.Strategy != "randseq" {
				tcpHdrLen := int((fake[ipv6HdrLen+12] >> 4) * 4)
				payloadLen := len(fake) - (ipv6HdrLen + tcpHdrLen)
				seq := binary.BigEndian.Uint32(fake[ipv6HdrLen+4 : ipv6HdrLen+8])
				binary.BigEndian.PutUint32(fake[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(payloadLen))
				sock.FixTCPChecksumV6(fake)
			}
		}
	}
}
