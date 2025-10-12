package nfq

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sni"
	"github.com/daniellavrushin/b4/sock"
	"github.com/florianl/go-nfqueue"
)

var sentFake sync.Map

func markFakeOnce(key string, ttl time.Duration) bool {
	_, loaded := sentFake.LoadOrStore(key, struct{}{})
	if loaded {
		return false
	}
	time.AfterFunc(ttl, func() { sentFake.Delete(key) })
	return true
}

func (w *Worker) Start() error {
	s, err := sock.NewSenderWithMark(int(w.cfg.Mark))
	if err != nil {
		return err
	}
	w.sock = s
	w.frag = &sock.Fragmenter{}

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
	go w.gc()

	go func() {
		pid := os.Getpid()
		log.Infof("NFQ bound pid=%d queue=%d", pid, w.qnum)
		_ = q.RegisterWithErrorFunc(w.ctx, func(a nfqueue.Attribute) int {
			if a.PacketID == nil || a.Payload == nil || len(*a.Payload) == 0 {
				return 0
			}
			id := *a.PacketID
			raw := *a.Payload

			v := raw[0] >> 4
			if v != 4 && v != 6 {
				_ = q.SetVerdict(id, nfqueue.NfAccept)
				return 0
			}
			var proto uint8
			var src, dst net.IP
			var ihl int
			if v == 4 {
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
				if len(raw) < 40 {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				ihl = 40
				proto = raw[6]
				src = net.IP(raw[8:24])
				dst = net.IP(raw[24:40])
			}

			if proto == 6 && len(raw) >= ihl+20 {
				tcp := raw[ihl:]
				if len(tcp) < 20 {
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
				if dport == 443 && len(payload) > 0 {
					k := fmt.Sprintf("%s:%d>%s:%d", src.String(), sport, dst.String(), dport)
					host, ok := w.feed(k, payload)
					if ok && w.matcher.Match(host) {
						onlyOnce := markFakeOnce(k, 20*time.Second)
						log.Infof("TCP: %s %s:%d -> %s:%d", host, src.String(), sport, dst.String(), dport)
						w.dropAndInjectTCP(raw, dst, onlyOnce)
						_ = q.SetVerdict(id, nfqueue.NfDrop)
						return 0
					}
				}
			}

			if proto == 17 && len(raw) >= ihl+8 {
				if w.cfg.UDPFilterQUIC == "disabled" {
					_ = q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}
				udp := raw[ihl:]
				if len(udp) >= 8 {
					payload := udp[8:]
					sport := binary.BigEndian.Uint16(udp[0:2])
					dport := binary.BigEndian.Uint16(udp[2:4])

					matchDport := false
					if w.cfg.UDPDPortMin > 0 && w.cfg.UDPDPortMax >= w.cfg.UDPDPortMin {
						if int(dport) >= w.cfg.UDPDPortMin && int(dport) <= w.cfg.UDPDPortMax {
							matchDport = true
						}
					} else {
						if dport == 443 {
							matchDport = true
						}
					}
					if !matchDport {
						_ = q.SetVerdict(id, nfqueue.NfAccept)
						return 0
					}

					handle := false
					host := ""
					switch w.cfg.UDPFilterQUIC {
					case "all":
						handle = true
					case "parse":
						if h, ok := sni.ParseQUICClientHelloSNI(payload); ok && w.matcher.Match(h) {
							host = h
							handle = true
						}
					}

					if handle {
						if host != "" {
							log.Infof("UDP: %s %s:%d -> %s:%d", host, src.String(), sport, dst.String(), dport)
						} else {
							log.Infof("UDP: %s:%d -> %s:%d", src.String(), sport, dst.String(), dport)
						}
						if w.cfg.UDPMode == "drop" {
							_ = q.SetVerdict(id, nfqueue.NfDrop)
							return 0
						}
						if w.cfg.UDPMode == "fake" {
							w.dropAndInjectQUIC(raw, dst)
							_ = q.SetVerdict(id, nfqueue.NfDrop)
							return 0
						}
					}
				}
			}

			_ = q.SetVerdict(id, nfqueue.NfAccept)
			return 0
		}, func(err error) int {
			log.Errorf("nfq: %v", err)
			return 0
		})
	}()

	return nil
}

func (w *Worker) dropAndInjectQUIC(raw []byte, dst net.IP) {
	if w.cfg.UDPMode != "fake" {
		return
	}
	if w.cfg.UDPFakeSeqLength > 0 {
		for i := 0; i < w.cfg.UDPFakeSeqLength; i++ {
			fake, ok := sock.BuildFakeUDPFromOriginal(raw, w.cfg.UDPFakeLen, w.cfg.FakeTTL)
			if ok {
				if w.cfg.UDPFakingStrategy == "checksum" {
					ipHdrLen := int((fake[0] & 0x0F) * 4)
					if len(fake) >= ipHdrLen+8 {
						fake[ipHdrLen+6] ^= 0xFF
						fake[ipHdrLen+7] ^= 0xFF
					}
				}
				_ = w.sock.SendIPv4(fake, dst)
				if w.cfg.Seg2Delay > 0 {
					time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
				} else {
					time.Sleep(1 * time.Millisecond)
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

	if w.cfg.FragSNIReverse {
		_ = w.sock.SendIPv4(frags[0], dst)
		time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
		_ = w.sock.SendIPv4(frags[1], dst)
	} else {
		_ = w.sock.SendIPv4(frags[1], dst)
		time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
		_ = w.sock.SendIPv4(frags[0], dst)
	}
}

func (w *Worker) dropAndInjectTCP(raw []byte, dst net.IP, injectFake bool) {
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

	if injectFake && w.cfg.FakeSNI && w.cfg.FakeSNISeqLength > 0 {
		w.sendFakeSNISequence(raw, dst)
	}

	switch w.cfg.FragmentStrategy {
	case "tcp":
		w.sendTCPFragments(raw, dst)
	case "ip":
		w.sendIPFragments(raw, dst)
	case "none":
		_ = w.sock.SendIPv4(raw, dst)
	default:
		w.sendTCPFragments(raw, dst)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (w *Worker) feed(key string, chunk []byte) (string, bool) {
	w.mu.Lock()
	st := w.flows[key]
	if st == nil {
		st = &flowState{buf: nil, last: time.Now()}
		w.flows[key] = st
	}
	if len(st.buf) < w.limit {
		need := w.limit - len(st.buf)
		if len(chunk) < need {
			st.buf = append(st.buf, chunk...)
		} else {
			st.buf = append(st.buf, chunk[:need]...)
		}
	}
	st.last = time.Now()
	buf := append([]byte(nil), st.buf...)
	w.mu.Unlock()
	host, ok := sni.ParseTLSClientHelloSNI(buf)
	if ok && host != "" {
		w.mu.Lock()
		delete(w.flows, key)
		w.mu.Unlock()
		return host, true
	}
	return "", false
}

func (w *Worker) sendTCPFragments(packet []byte, dst net.IP) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	totalLen := len(packet)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := totalLen - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	splitPos := w.cfg.FragSNIPosition
	if splitPos <= 0 || splitPos >= payloadLen {
		splitPos = 1
	}

	if w.cfg.FragMiddleSNI {
		payload := packet[payloadStart:]
		for i := 0; i < min(len(payload)-20, 100); i++ {
			if i+4 < len(payload) && payload[i] == '.' {
				if (payload[i+1] == 'c' && payload[i+2] == 'o' && payload[i+3] == 'm') ||
					(payload[i+1] == 'o' && payload[i+2] == 'r' && payload[i+3] == 'g') {
					splitPos = i + 2
					break
				}
			}
		}
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

	if w.cfg.FragSNIReverse {
		_ = w.sock.SendIPv4(seg2, dst)
		if w.cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg1, dst)
	} else {
		_ = w.sock.SendIPv4(seg1, dst)
		if w.cfg.Seg2Delay > 0 {
			time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg2, dst)
	}
}

func (w *Worker) sendIPFragments(packet []byte, dst net.IP) {
	splitPos := w.cfg.FragSNIPosition
	if splitPos <= 0 || splitPos >= len(packet) {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	ipHdrLen := int((packet[0] & 0x0F) * 4)
	splitPos = (splitPos + 7) &^ 7
	if splitPos >= len(packet) {
		splitPos = len(packet) - 8
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

	if w.cfg.FragSNIReverse {
		_ = w.sock.SendIPv4(frag2, dst)
		time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
		_ = w.sock.SendIPv4(frag1, dst)
	} else {
		_ = w.sock.SendIPv4(frag1, dst)
		time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
		_ = w.sock.SendIPv4(frag2, dst)
	}
}

func (w *Worker) sendFakeSNISequence(original []byte, dst net.IP) {
	if !w.cfg.FakeSNI || w.cfg.FakeSNISeqLength <= 0 {
		return
	}
	fake := sock.BuildFakeSNIPacket(original, w.cfg)
	for i := 0; i < w.cfg.FakeSNISeqLength; i++ {
		_ = w.sock.SendIPv4(fake, dst)
		if i+1 < w.cfg.FakeSNISeqLength {
			id := binary.BigEndian.Uint16(fake[4:6])
			binary.BigEndian.PutUint16(fake[4:6], id+1)
			if w.cfg.FakeStrategy != "pastseq" && w.cfg.FakeStrategy != "randseq" {
				ipHdrLen := int((fake[0] & 0x0F) * 4)
				tcpHdrLen := int(((fake[ipHdrLen+12] >> 4) & 0xF) * 4)
				plen := len(fake) - (ipHdrLen + tcpHdrLen)
				seq := binary.BigEndian.Uint32(fake[ipHdrLen+4 : ipHdrLen+8])
				binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], seq+uint32(plen))
				sock.FixIPv4Checksum(fake[:ipHdrLen])
				sock.FixTCPChecksum(fake)
			}
			if w.cfg.Seg2Delay > 0 {
				time.Sleep(time.Duration(w.cfg.Seg2Delay) * time.Millisecond)
			}
		}
	}
}

func (w *Worker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
	if w.q != nil {
		_ = w.q.Close()
	}
	if w.sock != nil {
		w.sock.Close()
	}
}

func (w *Worker) gc() {
	defer w.wg.Done()
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case now := <-t.C:
			w.mu.Lock()
			for k, st := range w.flows {
				if now.Sub(st.last) > w.ttl {
					delete(w.flows, k)
				}
			}
			w.mu.Unlock()
		}
	}
}
