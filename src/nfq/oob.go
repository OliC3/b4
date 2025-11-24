package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
)

func (w *Worker) sendOOBFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	// Determine split position
	oobPos := cfg.Fragmentation.OOBPosition
	if oobPos <= 0 {
		oobPos = 1
	}

	// Handle middle SNI positioning
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(packet[payloadStart:]); ok && sniEnd > sniStart {
			oobPos = sniStart + (sniEnd-sniStart)/2
			log.Tracef("OOB: SNI at %d-%d, injecting at %d", sniStart, sniEnd, oobPos)
		}
	}

	// Clamp to valid range
	if oobPos >= payloadLen {
		oobPos = payloadLen / 2
	}
	if oobPos <= 0 {
		oobPos = 1
	}

	oobChar := cfg.Fragmentation.OOBChar
	if oobChar == 0 {
		oobChar = 'x' // Default OOB character
	}

	seq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	id := binary.BigEndian.Uint16(packet[4:6])
	payload := packet[payloadStart:]

	log.Tracef("OOB: Injecting 0x%02x at pos %d of %d bytes", oobChar, oobPos, payloadLen)

	// ===== Segment 1: payload[0:oobPos] + OOBChar =====
	seg1DataLen := oobPos + 1 // original data + OOB byte
	seg1Len := payloadStart + seg1DataLen
	seg1 := make([]byte, seg1Len)

	// Copy IP + TCP headers
	copy(seg1[:payloadStart], packet[:payloadStart])
	// Copy first part of payload
	copy(seg1[payloadStart:payloadStart+oobPos], payload[:oobPos])
	// Inject OOB byte at the end
	seg1[payloadStart+oobPos] = oobChar

	// Set URG flag
	seg1[ipHdrLen+13] |= 0x20

	binary.BigEndian.PutUint16(seg1[ipHdrLen+18:ipHdrLen+20], uint16(oobPos+1))

	// Update IP total length
	binary.BigEndian.PutUint16(seg1[2:4], uint16(seg1Len))

	// Fix checksums
	sock.FixIPv4Checksum(seg1[:ipHdrLen])
	sock.FixTCPChecksum(seg1)

	// ===== Segment 2: payload[oobPos:] =====
	seg2DataLen := payloadLen - oobPos
	seg2Len := payloadStart + seg2DataLen
	seg2 := make([]byte, seg2Len)

	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], payload[oobPos:])

	binary.BigEndian.PutUint32(seg2[ipHdrLen+4:ipHdrLen+8], seq+uint32(oobPos)+1)

	binary.BigEndian.PutUint16(seg2[4:6], id+1)

	binary.BigEndian.PutUint16(seg2[2:4], uint16(seg2Len))

	seg2[ipHdrLen+13] &^= 0x20
	binary.BigEndian.PutUint16(seg2[ipHdrLen+18:ipHdrLen+20], 0)

	sock.FixIPv4Checksum(seg2[:ipHdrLen])
	sock.FixTCPChecksum(seg2)

	seg2delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(seg2, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg1, dst)
	} else {
		_ = w.sock.SendIPv4(seg1, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg2, dst)
	}

	log.Tracef("OOB: Sent seg1=%d bytes (with OOB), seg2=%d bytes", seg1Len, seg2Len)
}

// sendOOBFragmentsV6 is the IPv6 version of OOB injection
func (w *Worker) sendOOBFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	const ipv6HdrLen = 40

	if len(packet) < ipv6HdrLen+20 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen <= 0 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	// Determine split position
	oobPos := cfg.Fragmentation.OOBPosition
	if oobPos <= 0 {
		oobPos = 1
	}

	// Handle middle SNI positioning
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(packet[payloadStart:]); ok && sniEnd > sniStart {
			oobPos = sniStart + (sniEnd-sniStart)/2
			log.Tracef("OOB v6: SNI at %d-%d, injecting at %d", sniStart, sniEnd, oobPos)
		}
	}

	// Clamp to valid range
	if oobPos >= payloadLen {
		oobPos = payloadLen / 2
	}
	if oobPos <= 0 {
		oobPos = 1
	}

	oobChar := cfg.Fragmentation.OOBChar
	if oobChar == 0 {
		oobChar = 'x'
	}

	seq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])
	payload := packet[payloadStart:]

	log.Tracef("OOB v6: Injecting 0x%02x at pos %d of %d bytes", oobChar, oobPos, payloadLen)

	// ===== Segment 1: payload[0:oobPos] + OOBChar =====
	seg1DataLen := oobPos + 1
	seg1Len := payloadStart + seg1DataLen
	seg1 := make([]byte, seg1Len)

	copy(seg1[:payloadStart], packet[:payloadStart])
	copy(seg1[payloadStart:payloadStart+oobPos], payload[:oobPos])
	seg1[payloadStart+oobPos] = oobChar

	seg1[ipv6HdrLen+13] |= 0x20

	binary.BigEndian.PutUint16(seg1[ipv6HdrLen+18:ipv6HdrLen+20], uint16(oobPos+1))

	binary.BigEndian.PutUint16(seg1[4:6], uint16(seg1Len-ipv6HdrLen))

	sock.FixTCPChecksumV6(seg1)

	// ===== Segment 2: payload[oobPos:] =====
	seg2DataLen := payloadLen - oobPos
	seg2Len := payloadStart + seg2DataLen
	seg2 := make([]byte, seg2Len)

	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], payload[oobPos:])

	binary.BigEndian.PutUint32(seg2[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(oobPos)+1)

	binary.BigEndian.PutUint16(seg2[4:6], uint16(seg2Len-ipv6HdrLen))

	seg2[ipv6HdrLen+13] &^= 0x20
	binary.BigEndian.PutUint16(seg2[ipv6HdrLen+18:ipv6HdrLen+20], 0)

	sock.FixTCPChecksumV6(seg2)

	seg2delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv6(seg2, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg1, dst)
	} else {
		_ = w.sock.SendIPv6(seg1, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg2, dst)
	}

	log.Tracef("OOB v6: Sent seg1=%d bytes (with OOB), seg2=%d bytes", seg1Len, seg2Len)
}
