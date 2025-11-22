package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// dropAndInjectQUIV6 handles QUIC (UDP) packet manipulation for IPv6
func (w *Worker) dropAndInjectQUICV6(cfg *config.SetConfig, raw []byte, dst net.IP) {
	seg2d := cfg.UDP.Seg2Delay
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
				if seg2d > 0 {
					time.Sleep(time.Duration(seg2d) * time.Millisecond)
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
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(frags[1], dst)
	} else {
		_ = w.sock.SendIPv6(frags[1], dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(frags[0], dst)
	}
}

// dropAndInjectTCPv6 handles TCP packet manipulation for IPv6
func (w *Worker) dropAndInjectTCPv6(cfg *config.SetConfig, raw []byte, dst net.IP) {
	if len(raw) < 60 { // IPv6 header (40) + TCP header (20 min)
		_ = w.sock.SendIPv6(raw, dst)
		return
	}

	if cfg.Fragmentation.OOBPosition > 0 {
		if w.sendWithOOBv6(cfg, raw, dst) {
			return
		}
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
	if cfg.Faking.SNI && cfg.Faking.SNISeqLength > 0 {
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

func (w *Worker) sendTCPSegmentsv6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipv6HdrLen := 40
	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	seg2d := cfg.TCP.Seg2Delay

	totalLen := len(packet)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := totalLen - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	payload := packet[payloadStart:]
	p1 := cfg.Fragmentation.SNIPosition
	validP1 := p1 > 0 && p1 < payloadLen

	p2 := -1
	if cfg.Fragmentation.MiddleSNI {
		if s, e, ok := locateSNI(payload); ok && e-s >= 4 {
			sniLen := e - s
			if sniLen > 30 {
				p2 = e - 12
			} else {
				p2 = s + sniLen/2
			}
		}
	}

	validP2 := p2 > 0 && p2 < payloadLen && (!validP1 || p2 != p1)

	if !validP1 && !validP2 {
		p1 = 1
		validP1 = p1 < payloadLen
	}

	if validP1 && validP2 && p2 < p1 {
		p1, p2 = p2, p1
	}

	// Three-segment case when both positions are valid
	if validP1 && validP2 {
		seg1Len := payloadStart + p1
		seg2Len := payloadStart + (p2 - p1)
		seg3Len := payloadStart + (payloadLen - p2)

		seg1 := make([]byte, seg1Len)
		copy(seg1, packet[:seg1Len])

		seg2 := make([]byte, seg2Len)
		copy(seg2[:payloadStart], packet[:payloadStart])
		copy(seg2[payloadStart:], payload[p1:p2])

		seg3 := make([]byte, seg3Len)
		copy(seg3[:payloadStart], packet[:payloadStart])
		copy(seg3[payloadStart:], payload[p2:])

		// Update IPv6 payload length for seg1
		binary.BigEndian.PutUint16(seg1[4:6], uint16(seg1Len-ipv6HdrLen))
		sock.FixTCPChecksumV6(seg1)

		// Update seg2 TCP sequence and IPv6 payload length
		seq0 := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])
		binary.BigEndian.PutUint32(seg2[ipv6HdrLen+4:ipv6HdrLen+8], seq0+uint32(p1))
		binary.BigEndian.PutUint16(seg2[4:6], uint16(seg2Len-ipv6HdrLen))
		sock.FixTCPChecksumV6(seg2)

		// Update seg3 TCP sequence and IPv6 payload length
		binary.BigEndian.PutUint32(seg3[ipv6HdrLen+4:ipv6HdrLen+8], seq0+uint32(p2))
		binary.BigEndian.PutUint16(seg3[4:6], uint16(seg3Len-ipv6HdrLen))
		sock.FixTCPChecksumV6(seg3)

		if cfg.Fragmentation.SNIReverse {
			_ = w.sock.SendIPv6(seg2, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv6(seg1, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv6(seg3, dst)
		} else {
			_ = w.sock.SendIPv6(seg1, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv6(seg2, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv6(seg3, dst)
		}
		return
	}

	// Two-segment case (fallback)
	splitPos := p1
	if !validP1 {
		splitPos = p2
	}

	seg1Len := payloadStart + splitPos
	seg1 := make([]byte, seg1Len)
	copy(seg1, packet[:seg1Len])

	seg2Len := payloadStart + (payloadLen - splitPos)
	seg2 := make([]byte, seg2Len)
	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], packet[payloadStart+splitPos:])

	// Update IPv6 payload length for seg1
	binary.BigEndian.PutUint16(seg1[4:6], uint16(seg1Len-ipv6HdrLen))
	sock.FixTCPChecksumV6(seg1)

	// Update seg2 TCP sequence and IPv6 payload length
	seq := binary.BigEndian.Uint32(seg2[ipv6HdrLen+4 : ipv6HdrLen+8])
	binary.BigEndian.PutUint32(seg2[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(splitPos))
	binary.BigEndian.PutUint16(seg2[4:6], uint16(seg2Len-ipv6HdrLen))
	sock.FixTCPChecksumV6(seg2)

	if cfg.Fragmentation.SNIReverse {
		_ = w.sock.SendIPv6(seg2, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg1, dst)
	} else {
		_ = w.sock.SendIPv6(seg1, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg2, dst)
	}
}

// sendIPFragmentsv6 sends packet as IPv6 fragments (IP-level fragmentation)
func (w *Worker) sendIPFragmentsv6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	splitPos := cfg.Fragmentation.SNIPosition
	seg2d := cfg.TCP.Seg2Delay
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
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(fragments[0], dst)
	} else {
		_ = w.sock.SendIPv6(fragments[0], dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(fragments[1], dst)
	}
}

// sendFakeSNISequencev6 sends a sequence of fake SNI packets for IPv6
func (w *Worker) sendFakeSNISequencev6(cfg *config.SetConfig, original []byte, dst net.IP) {
	faking := cfg.Faking
	if !faking.SNI || faking.SNISeqLength <= 0 {
		return
	}

	fake := sock.BuildFakeSNIPacketV6(original, cfg)
	if fake == nil {
		return
	}

	ipv6HdrLen := 40

	for i := 0; i < faking.SNISeqLength; i++ {
		_ = w.sock.SendIPv6(fake, dst)

		// Update for next iteration
		if i+1 < faking.SNISeqLength {
			// Adjust sequence number for non-past/rand strategies
			if faking.Strategy != "pastseq" && faking.Strategy != "randseq" {
				tcpHdrLen := int((fake[ipv6HdrLen+12] >> 4) * 4)
				payloadLen := len(fake) - (ipv6HdrLen + tcpHdrLen)
				seq := binary.BigEndian.Uint32(fake[ipv6HdrLen+4 : ipv6HdrLen+8])
				binary.BigEndian.PutUint32(fake[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(payloadLen))
				sock.FixTCPChecksumV6(fake)
			}
		}
	}
}
