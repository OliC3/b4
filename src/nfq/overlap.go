package nfq

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// sendOverlapFragments exploits TCP segment overlap behavior
func (w *Worker) sendOverlapFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
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

	sniStart, sniEnd, ok := locateSNI(payload)
	if !ok || sniEnd <= sniStart || sniEnd > payloadLen || sniStart < 0 {
		w.sendTCPFragments(cfg, packet, dst)
		return
	}

	// Segment 1: From before SNI to end, contains REAL SNI (sent FIRST - server keeps this)
	overlapStart := sniStart - 4
	if overlapStart < 0 {
		overlapStart = 0
	}

	seg1Len := payloadStart + (payloadLen - overlapStart)
	seg1 := make([]byte, seg1Len)
	copy(seg1[:payloadStart], packet[:payloadStart])
	copy(seg1[payloadStart:], payload[overlapStart:]) // REAL SNI

	binary.BigEndian.PutUint32(seg1[ipHdrLen+4:ipHdrLen+8], seq0+uint32(overlapStart))
	binary.BigEndian.PutUint16(seg1[2:4], uint16(seg1Len))
	sock.FixIPv4Checksum(seg1[:ipHdrLen])
	sock.FixTCPChecksum(seg1)

	// Segment 2: From start through SNI, with FAKE SNI (sent SECOND - DPI sees, server discards overlap)
	seg2End := sniEnd + 4
	if seg2End > payloadLen {
		seg2End = payloadLen
	}

	seg2Len := payloadStart + seg2End
	seg2 := make([]byte, seg2Len)
	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], payload[:seg2End])

	// Inject fake SNI
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

	binary.BigEndian.PutUint16(seg2[4:6], id0+1)
	binary.BigEndian.PutUint16(seg2[2:4], uint16(seg2Len))
	seg2[ipHdrLen+13] &^= 0x08
	sock.FixIPv4Checksum(seg2[:ipHdrLen])
	sock.FixTCPChecksum(seg2)

	delay := cfg.TCP.Seg2Delay

	// REAL first (server keeps), then FAKE (DPI sees but server discards)
	_ = w.sock.SendIPv4(seg1, dst)
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
	_ = w.sock.SendIPv4(seg2, dst)
}
