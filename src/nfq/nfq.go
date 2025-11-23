package nfq

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/daniellavrushin/b4/capture"
	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/metrics"
	"github.com/daniellavrushin/b4/quic"
	"github.com/daniellavrushin/b4/sni"
	"github.com/daniellavrushin/b4/sock"
	"github.com/daniellavrushin/b4/stun"
	"github.com/florianl/go-nfqueue"
)

func (w *Worker) Start() error {
	cfg := w.getConfig()
	mark := cfg.Queue.Mark
	s, err := sock.NewSenderWithMark(int(mark))
	if err != nil {
		return err
	}
	w.sock = s

	c := nfqueue.Config{
		NfQueue:      w.qnum,
		MaxPacketLen: 0xffff,
		MaxQueueLen:  4096,
		Copymode:     nfqueue.NfQnlCopyPacket,
	}
	q, err := nfqueue.Open(&c)
	if err != nil {
		return err
	}
	w.q = q

	w.wg.Add(1)
	go w.gc(cfg)

	w.wg.Add(1)

	go func() {
		pid := os.Getpid()
		log.Tracef("NFQ bound pid=%d queue=%d", pid, w.qnum)
		defer w.wg.Done()
		_ = q.RegisterWithErrorFunc(w.ctx, func(a nfqueue.Attribute) int {
			cfg := w.getConfig()
			set := cfg.MainSet

			matcher := w.getMatcher()
			id := *a.PacketID

			if a.Mark != nil && *a.Mark == uint32(mark) {
				_ = q.SetVerdict(id, nfqueue.NfAccept)
				return 0
			}

			select {
			case <-w.ctx.Done():
				return 0
			default:
			}

			atomic.AddUint64(&w.packetsProcessed, 1)

			if a.PacketID == nil || a.Payload == nil || len(*a.Payload) == 0 {
				return 0
			}
			raw := *a.Payload

			v := raw[0] >> 4
			if v != IPv4 && v != IPv6 {
				_ = q.SetVerdict(id, nfqueue.NfAccept)
				return 0
			}
			var proto uint8
			var src, dst net.IP
			var ihl int
			if v == IPv4 {
				if len(raw) < 20 {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				ihl = int(raw[0]&0x0f) * 4
				if len(raw) < ihl {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				proto = raw[9]
				src = net.IP(raw[12:16])
				dst = net.IP(raw[16:20])
			} else {
				if len(raw) < IPv6HeaderLen {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				ihl = IPv6HeaderLen
				nextHeader := raw[6]
				offset := 40

				// Skip extension headers
				for {
					switch nextHeader {
					case 0, 43, 44, 60: // Hop-by-Hop, Routing, Fragment, Destination Options
						if len(raw) < offset+2 {
							_ = q.SetVerdict(id, nfqueue.NfAccept)
							return 0
						}
						nextHeader = raw[offset]
						hdrLen := int(raw[offset+1])*8 + 8
						offset += hdrLen
					default:
						goto done
					}
				}
			done:
				proto = nextHeader
				ihl = offset
				src = net.IP(raw[8:24])
				dst = net.IP(raw[24:40])
			}

			if src.IsLoopback() || dst.IsLoopback() {
				_ = q.SetVerdict(id, nfqueue.NfAccept)
				return 0
			}
			srcStr := src.String()
			dstStr := dst.String()

			matched, st := matcher.MatchIP(dst)
			if matched {
				set = st
			}

			// TCP processing
			if proto == 6 && len(raw) >= ihl+TCPHeaderMinLen {
				tcp := raw[ihl:]
				if len(tcp) < TCPHeaderMinLen {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				datOff := int((tcp[12]>>4)&0x0f) * 4
				if len(tcp) < datOff {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				payload := tcp[datOff:]
				sport := binary.BigEndian.Uint16(tcp[0:2])
				dport := binary.BigEndian.Uint16(tcp[2:4])

				tcpFlags := tcp[13]
				isSyn := (tcpFlags & 0x02) != 0 // SYN flag
				isAck := (tcpFlags & 0x10) != 0 // ACK flag

				if set.TCP.SynFake && isSyn && !isAck && dport == HTTPSPort {

					if matched {
						log.Tracef("TCP SYN to %s:%d - sending fake SYN (set: %s)", dstStr, dport, set.Name)

						metrics := metrics.GetMetricsCollector()
						metrics.RecordConnection("TCP-SYN", "", srcStr, dstStr, true)

						if v == IPv4 {
							w.sendFakeSyn(set, raw, ihl, datOff)
						} else {
							w.sendFakeSynV6(set, raw, ihl, datOff)
						}

						_ = q.SetVerdict(id, nfqueue.NfDrop)
						return 0
					}

					log.Tracef("TCP SYN to %s:%d - passing through", dstStr, dport)
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}

				host := ""
				matchedIP := matched
				matchedSNI := false
				ipTarget := ""
				sniTarget := ""

				if dport == HTTPSPort && len(payload) > 0 {
					connKey := fmt.Sprintf("%s:%d->%s:%d", srcStr, sport, dstStr, dport)

					host, _ = sni.ParseTLSClientHelloSNI(payload)

					if captureManager := capture.GetManager(cfg); captureManager != nil {
						captureManager.CapturePayload(connKey, host, "tls", payload)
					}

					if host != "" {
						if mSNI, stSNI := matcher.MatchSNI(host); mSNI {
							matchedSNI = true
							matched = true
							set = stSNI
						}
					}
				}

				if matchedIP {
					ipTarget = st.Name
				}
				if matchedSNI {
					sniTarget = set.Name
				}

				log.Infof(",TCP,%s,%s,%s:%d,%s,%s:%d", sniTarget, host, srcStr, sport, ipTarget, dstStr, dport)

				if matched {
					metrics := metrics.GetMetricsCollector()
					metrics.RecordConnection("TCP", host, srcStr, dstStr, true)
					metrics.RecordPacket(uint64(len(raw)))

					packetCopy := make([]byte, len(raw))
					copy(packetCopy, raw)

					if set.TCP.DropSACK {
						if v == 4 {
							packetCopy = sock.StripSACKFromTCP(packetCopy)
						} else {
							packetCopy = sock.StripSACKFromTCPv6(packetCopy)
						}
					}

					dstCopy := make(net.IP, len(dst))
					copy(dstCopy, dst)
					setCopy := set

					_ = q.SetVerdict(id, nfqueue.NfDrop)

					go func(s *config.SetConfig, pkt []byte, d net.IP) {
						if v == 4 {
							w.dropAndInjectTCP(s, pkt, d)
						} else {
							w.dropAndInjectTCPv6(s, pkt, d)
						}
					}(setCopy, packetCopy, dstCopy)
					return 0
				}

				_ = q.SetVerdict(id, nfqueue.NfAccept)
				return 0
			}

			// UDP processing
			if proto == 17 && len(raw) >= ihl+8 {
				udp := raw[ihl:]
				if len(udp) < 8 {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}

				payload := udp[8:]
				sport := binary.BigEndian.Uint16(udp[0:2])
				dport := binary.BigEndian.Uint16(udp[2:4])
				connKey := fmt.Sprintf("%s:%d->%s:%d", srcStr, sport, dstStr, dport)

				// Early exit for DNS
				if dport == 53 || sport == 53 {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}

				if set == nil {
					set = cfg.MainSet
				}

				matchedIP := matched
				matchedPort := false
				matchedQUIC := false
				isSTUN := false
				host := ""
				ipTarget := ""
				sniTarget := ""

				if matchedIP {
					ipTarget = st.Name
				}

				if mport, portSet := matcher.MatchUDPPort(dport); mport {
					matchedPort = true
					set = portSet
					ipTarget = portSet.Name
				}

				isSTUN = stun.IsSTUNMessage(payload)

				switch set.UDP.FilterQUIC {
				case "disabled":

				case "all":
					if quic.IsInitial(payload) {
						matchedQUIC = true
						if h, ok := sni.ParseQUICClientHelloSNI(payload); ok {
							host = h
						}
					}

				case "parse":
					if h, ok := sni.ParseQUICClientHelloSNI(payload); ok {
						host = h
						if mSNI, sniSet := matcher.MatchSNI(host); mSNI {
							matchedQUIC = true
							set = sniSet
							sniTarget = sniSet.Name
						}
					}
				}

				if captureManager := capture.GetManager(cfg); captureManager != nil {
					captureManager.CapturePayload(connKey, host, "quic", payload)
				}

				shouldHandle := (matchedPort || matchedIP || matchedQUIC) && !(isSTUN && set.UDP.FilterSTUN)

				matched = shouldHandle

				// Log ALL UDP packets (this runs before verdict)
				log.Infof(",UDP,%s,%s,%s:%d,%s,%s:%d", sniTarget, host, srcStr, sport, ipTarget, dstStr, dport)

				// Early exit for STUN
				if isSTUN && set.UDP.FilterSTUN {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}

				// Accept if no match
				if !shouldHandle {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}

				metrics := metrics.GetMetricsCollector()
				metrics.RecordConnection("UDP", host, srcStr, dstStr, matched)
				metrics.RecordPacket(uint64(len(raw)))

				// Apply configured UDP mode
				switch set.UDP.Mode {
				case "drop":
					_ = q.SetVerdict(id, nfqueue.NfDrop)
					return 0

				case "fake":
					packetCopy := make([]byte, len(raw))
					copy(packetCopy, raw)
					dstCopy := make(net.IP, len(dst))
					copy(dstCopy, dst)
					setCopy := set

					_ = q.SetVerdict(id, nfqueue.NfDrop)

					go func(s *config.SetConfig, pkt []byte, d net.IP) {
						if v == IPv4 {
							w.dropAndInjectQUIC(s, pkt, d)
						} else {
							w.dropAndInjectQUICV6(s, pkt, d)
						}
					}(setCopy, packetCopy, dstCopy)
					return 0

				default:
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
			}

			_ = q.SetVerdict(id, nfqueue.NfAccept)
			return 0
		}, func(e error) int {
			if w.ctx.Err() != nil {
				return 0
			}
			if errors.Is(e, os.ErrClosed) || errors.Is(e, net.ErrClosed) || errors.Is(e, syscall.EBADF) {
				return 0
			}
			if ne, ok := e.(net.Error); ok && ne.Timeout() {
				return 0
			}
			msg := e.Error()
			if strings.Contains(msg, "use of closed file") || strings.Contains(msg, "file descriptor") {
				return 0
			}
			log.Errorf("nfq: %v", e)
			return 0
		})
	}()

	return nil
}

func (w *Worker) dropAndInjectQUIC(cfg *config.SetConfig, raw []byte, dst net.IP) {
	udpCfg := &cfg.UDP
	seg2d := udpCfg.Seg2Delay
	if udpCfg.Mode != "fake" {
		return
	}
	if udpCfg.FakeSeqLength > 0 {
		for i := 0; i < udpCfg.FakeSeqLength; i++ {
			fake, ok := sock.BuildFakeUDPFromOriginalV4(raw, udpCfg.FakeLen, cfg.Faking.TTL)
			if ok {
				if udpCfg.FakingStrategy == "checksum" {
					ipHdrLen := int((fake[0] & 0x0F) * 4)
					if len(fake) >= ipHdrLen+8 {
						fake[ipHdrLen+6] ^= 0xFF
						fake[ipHdrLen+7] ^= 0xFF
					}
				}
				_ = w.sock.SendIPv4(fake, dst)
				if seg2d > 0 {
					time.Sleep(time.Duration(seg2d) * time.Millisecond)
				}
			}
		}
	}

	splitPos := 24
	frags, ok := sock.IPv4FragmentUDP(raw, splitPos)
	if !ok {
		_ = w.sock.SendIPv4(raw, dst)
		return
	}

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(frags[0], dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(frags[1], dst)
	} else {
		_ = w.sock.SendIPv4(frags[1], dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(frags[0], dst)
	}
}

func (w *Worker) dropAndInjectTCP(cfg *config.SetConfig, raw []byte, dst net.IP) {

	if len(raw) < 40 {
		_ = w.sock.SendIPv4(raw, dst)
		return
	}

	ipHdrLen := int((raw[0] & 0x0F) * 4)
	tcpHdrLen := int((raw[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := len(raw) - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv4(raw, dst)
		return
	}

	if cfg.Faking.SNI && cfg.Faking.SNISeqLength > 0 {
		w.sendFakeSNISequence(cfg, raw, dst)
	}

	switch cfg.Fragmentation.Strategy {
	case "tcp":
		w.sendTCPFragments(cfg, raw, dst)
	case "ip":
		w.sendIPFragments(cfg, raw, dst)
	case "oob":
		w.sendOOBFragments(cfg, raw, dst)
	case "tls":
		w.sendTLSFragments(cfg, raw, dst)
	case "none":
		_ = w.sock.SendIPv4(raw, dst)
	default:
		w.sendTCPFragments(cfg, raw, dst)
	}
}

func (w *Worker) sendTCPFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {

	seg2d := cfg.TCP.Seg2Delay
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	totalLen := len(packet)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := totalLen - payloadStart
	if payloadLen <= 0 {
		_ = w.sock.SendIPv4(packet, dst)
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

		binary.BigEndian.PutUint16(seg1[2:4], uint16(seg1Len))
		sock.FixIPv4Checksum(seg1[:ipHdrLen])
		sock.FixTCPChecksum(seg1)

		seq0 := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
		id0 := binary.BigEndian.Uint16(packet[4:6])

		binary.BigEndian.PutUint32(seg2[ipHdrLen+4:ipHdrLen+8], seq0+uint32(p1))
		binary.BigEndian.PutUint16(seg2[4:6], id0+1)
		binary.BigEndian.PutUint16(seg2[2:4], uint16(seg2Len))
		sock.FixIPv4Checksum(seg2[:ipHdrLen])
		sock.FixTCPChecksum(seg2)

		binary.BigEndian.PutUint32(seg3[ipHdrLen+4:ipHdrLen+8], seq0+uint32(p2))
		binary.BigEndian.PutUint16(seg3[4:6], id0+2)
		binary.BigEndian.PutUint16(seg3[2:4], uint16(seg3Len))
		sock.FixIPv4Checksum(seg3[:ipHdrLen])
		sock.FixTCPChecksum(seg3)

		if cfg.Fragmentation.ReverseOrder {
			_ = w.sock.SendIPv4(seg2, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv4(seg1, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv4(seg3, dst)
		} else {
			_ = w.sock.SendIPv4(seg1, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv4(seg2, dst)
			if seg2d > 0 {
				time.Sleep(time.Duration(seg2d) * time.Millisecond)
			}
			_ = w.sock.SendIPv4(seg3, dst)
		}
		return
	}

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

	binary.BigEndian.PutUint16(seg1[2:4], uint16(seg1Len))
	sock.FixIPv4Checksum(seg1[:ipHdrLen])
	sock.FixTCPChecksum(seg1)

	seq := binary.BigEndian.Uint32(seg2[ipHdrLen+4 : ipHdrLen+8])
	binary.BigEndian.PutUint32(seg2[ipHdrLen+4:ipHdrLen+8], seq+uint32(splitPos))
	id := binary.BigEndian.Uint16(seg1[4:6])
	binary.BigEndian.PutUint16(seg2[4:6], id+1)
	binary.BigEndian.PutUint16(seg2[2:4], uint16(seg2Len))
	sock.FixIPv4Checksum(seg2[:ipHdrLen])
	sock.FixTCPChecksum(seg2)

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(seg2, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg1, dst)
	} else {
		_ = w.sock.SendIPv4(seg1, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg2, dst)
	}
}

func (w *Worker) sendIPFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {

	splitPos := cfg.Fragmentation.SNIPosition
	seg2d := cfg.TCP.Seg2Delay
	if splitPos <= 0 || splitPos >= len(packet) {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	ipHdrLen := int((packet[0] & 0x0F) * 4)

	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen

	splitPos = payloadStart + splitPos

	splitPos = (splitPos + 7) &^ 7

	minSplitPos := ipHdrLen + 8
	if splitPos < minSplitPos {
		splitPos = minSplitPos
	}

	if splitPos >= len(packet) {
		splitPos = len(packet) - 8
		if splitPos < minSplitPos {
			_ = w.sock.SendIPv4(packet, dst)
			return
		}
	}

	frag1 := make([]byte, splitPos)
	copy(frag1, packet[:splitPos])
	frag1[6] |= 0x20
	binary.BigEndian.PutUint16(frag1[2:4], uint16(splitPos))
	sock.FixIPv4Checksum(frag1[:ipHdrLen])

	frag2Len := ipHdrLen + len(packet) - splitPos
	frag2 := make([]byte, frag2Len)
	copy(frag2, packet[:ipHdrLen])
	copy(frag2[ipHdrLen:], packet[splitPos:])
	fragOff := uint16(splitPos-ipHdrLen) / 8
	binary.BigEndian.PutUint16(frag2[6:8], fragOff)
	binary.BigEndian.PutUint16(frag2[2:4], uint16(frag2Len))
	sock.FixIPv4Checksum(frag2[:ipHdrLen])

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(frag2, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(frag1, dst)
	} else {
		_ = w.sock.SendIPv4(frag1, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(frag2, dst)
	}
}

func (w *Worker) sendFakeSNISequence(cfg *config.SetConfig, original []byte, dst net.IP) {
	fk := &cfg.Faking
	if !fk.SNI || fk.SNISeqLength <= 0 {
		return
	}

	fake := sock.BuildFakeSNIPacketV4(original, cfg)
	ipHdrLen := int((fake[0] & 0x0F) * 4)
	tcpHdrLen := int((fake[ipHdrLen+12] >> 4) * 4)

	for i := 0; i < fk.SNISeqLength; i++ {
		_ = w.sock.SendIPv4(fake, dst)

		// Update for next iteration
		if i+1 < fk.SNISeqLength {
			// Increment IP ID
			id := binary.BigEndian.Uint16(fake[4:6])
			binary.BigEndian.PutUint16(fake[4:6], id+1)

			// Adjust sequence number for non-past/rand strategies
			if fk.Strategy != "pastseq" && fk.Strategy != "randseq" {
				payloadLen := len(fake) - (ipHdrLen + tcpHdrLen)
				seq := binary.BigEndian.Uint32(fake[ipHdrLen+4 : ipHdrLen+8])
				binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], seq+uint32(payloadLen))
				sock.FixIPv4Checksum(fake[:ipHdrLen])
				sock.FixTCPChecksum(fake)
			}
		}
	}
}

// locateSNI returns start and end (relative to payload start) of the SNI hostname bytes.
func locateSNI(payload []byte) (start, end int, ok bool) {
	// TLS record header: ContentType(1)=0x16, Version(2), Length(2)
	if len(payload) < 5 || payload[0] != TLSHandshakeType {
		return 0, 0, false
	}
	recLen := int(binary.BigEndian.Uint16(payload[3:5]))
	// Be tolerant if the full record isn't present yet
	if 5+recLen > len(payload) {
		recLen = len(payload) - 5
	}
	p := 5 // handshake starts right after record header

	// Handshake header: HandshakeType(1)=client_hello(1), Length(3)
	if p+4 > len(payload) || payload[p] != TLSClientHello {
		return 0, 0, false
	}
	hsLen := int(payload[p+1])<<16 | int(payload[p+2])<<8 | int(payload[p+3])
	p += 4
	if p+hsLen > len(payload) {
		hsLen = len(payload) - p
	}

	if p+2+32 > len(payload) {
		return 0, 0, false
	}
	p += 2 + 32

	// session_id
	if p >= len(payload) {
		return 0, 0, false
	}
	sidLen := int(payload[p])
	p++
	if p+sidLen > len(payload) {
		return 0, 0, false
	}
	p += sidLen

	// cipher_suites
	if p+2 > len(payload) {
		return 0, 0, false
	}
	csLen := int(binary.BigEndian.Uint16(payload[p : p+2]))
	p += 2
	if p+csLen > len(payload) {
		return 0, 0, false
	}
	p += csLen

	// compression_methods
	if p >= len(payload) {
		return 0, 0, false
	}
	cmLen := int(payload[p])
	p++
	if p+cmLen > len(payload) {
		return 0, 0, false
	}
	p += cmLen

	// extensions
	if p+2 > len(payload) {
		return 0, 0, false
	}
	extLen := int(binary.BigEndian.Uint16(payload[p : p+2]))
	p += 2
	if p+extLen > len(payload) {
		extLen = len(payload) - p
	}
	e := p
	ee := p + extLen

	// Walk extensions to find server_name (type=0)
	for e+4 <= ee {
		extType := binary.BigEndian.Uint16(payload[e : e+2])
		extDataLen := int(binary.BigEndian.Uint16(payload[e+2 : e+4]))
		e += 4
		if e+extDataLen > ee {
			break
		}

		if extType == 0 && extDataLen >= 5 {
			q := e
			// server_name_list length (2)
			if q+2 > e+extDataLen {
				break
			}
			listLen := int(binary.BigEndian.Uint16(payload[q : q+2]))
			q += 2
			if q+listLen > e+extDataLen {
				break
			}
			// First item: name_type(1) == 0 (host_name)
			if q+3 > e+extDataLen {
				break
			}
			nameType := payload[q]
			q++
			if nameType != 0 {
				break
			}
			nameLen := int(binary.BigEndian.Uint16(payload[q : q+2]))
			q += 2
			if nameLen == 0 || q+nameLen > e+extDataLen {
				break
			}
			// q is absolute offset into payload
			return q, q + nameLen, true
		}

		e += extDataLen
	}
	return 0, 0, false
}

func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	if w.q != nil {
		_ = w.q.Close()
	}
	done := make(chan struct{})
	go func() { w.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
	if w.sock != nil {
		w.sock.Close()
	}
}

func (w *Worker) gc(cfg *config.Config) {
	defer w.wg.Done()
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-t.C:

			if cfg.System.WebServer.IsEnabled {
				mtcs := metrics.GetMetricsCollector()
				workerID := int(w.qnum - uint16(cfg.Queue.StartNum))
				processed := atomic.LoadUint64(&w.packetsProcessed)
				mtcs.UpdateSingleWorker(workerID, "active", processed)
			}
		}
	}
}

func (w *Worker) GetStats() (uint64, string) {
	return atomic.LoadUint64(&w.packetsProcessed), "active"
}
