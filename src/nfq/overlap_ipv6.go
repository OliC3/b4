package nfq

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// sendOverlapFragmentsV6 - IPv6 version: exploits TCP segment overlap behavior
func (w *Worker) sendOverlapFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) {
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

	sniStart, sniEnd, ok := locateSNI(payload)
	if !ok || sniEnd <= sniStart || sniEnd > payloadLen || sniStart < 0 {
		w.sendTCPSegmentsv6(cfg, packet, dst)
		return
	}

	// Segment 1: Contains REAL SNI (sent FIRST - server keeps)
	overlapStart := sniStart - 4
	if overlapStart < 0 {
		overlapStart = 0
	}

	seg1Len := payloadStart + (payloadLen - overlapStart)
	seg1 := make([]byte, seg1Len)
	copy(seg1[:payloadStart], packet[:payloadStart])
	copy(seg1[payloadStart:], payload[overlapStart:])

	binary.BigEndian.PutUint32(seg1[ipv6HdrLen+4:ipv6HdrLen+8], seq0+uint32(overlapStart))
	binary.BigEndian.PutUint16(seg1[4:6], uint16(seg1Len-ipv6HdrLen))
	sock.FixTCPChecksumV6(seg1)

	// Segment 2: With FAKE SNI (sent SECOND - DPI sees, server discards overlap)
	seg2End := sniEnd + 4
	if seg2End > payloadLen {
		seg2End = payloadLen
	}

	seg2Len := payloadStart + seg2End
	seg2 := make([]byte, seg2Len)
	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], payload[:seg2End])

	sniLen := sniEnd - sniStart
	fakeDomains := cfg.Fragmentation.Overlap.FakeSNIs
	if len(fakeDomains) == 0 {
		fakeDomains = config.DefaultSetConfig.Fragmentation.Overlap.FakeSNIs
	}
	fakeSNI := []byte(fakeDomains[int(seq0)%len(fakeDomains)])
	if len(fakeSNI) < sniLen {
		fakeSNI = append(fakeSNI, bytes.Repeat([]byte{'.'}, sniLen-len(fakeSNI))...)
	}
	copy(seg2[payloadStart+sniStart:payloadStart+sniEnd], fakeSNI[:sniLen])

	binary.BigEndian.PutUint16(seg2[4:6], uint16(seg2Len-ipv6HdrLen))
	seg2[ipv6HdrLen+13] &^= 0x08
	sock.FixTCPChecksumV6(seg2)

	delay := cfg.TCP.Seg2Delay

	_ = w.sock.SendIPv6(seg1, dst)
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	_ = w.sock.SendIPv6(seg2, dst)
}
