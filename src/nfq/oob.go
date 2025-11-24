package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
)

// sendOOBFragments sends TCP packet with OOB (urgent) data - byedpi style
// Splits at oobPos, marks first part as urgent WITHOUT adding extra bytes
func (w *Worker) sendOOBFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
	if cfg.Fragmentation.OOBPosition <= 0 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

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

	// Calculate OOB split position
	oobPos := cfg.Fragmentation.OOBPosition

	// Handle middle SNI positioning if configured
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(packet[payloadStart:]); ok {
			// Split in middle of SNI
			oobPos = sniStart + (sniEnd-sniStart)/2
			log.Tracef("OOB: SNI detected at %d-%d, splitting at %d", sniStart, sniEnd, oobPos)
		}
	}

	// Validate position
	if oobPos <= 0 || oobPos >= payloadLen {
		oobPos = 1 // Default to 1 byte like byedpi
	}

	log.Tracef("OOB: Splitting at position %d of %d bytes (reverse=%v)",
		oobPos, payloadLen, cfg.Fragmentation.ReverseOrder)

	// Get original sequence and ID
	seq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	id := binary.BigEndian.Uint16(packet[4:6])

	// ===== First segment: data up to oobPos with URG flag =====
	seg1Len := payloadStart + oobPos
	seg1 := make([]byte, seg1Len)
	copy(seg1, packet[:seg1Len])

	// Set URG flag (bit 5 of TCP flags)
	seg1[ipHdrLen+13] |= 0x20

	// Set urgent pointer to the END of urgent data
	// byedpi uses oobPos here (points to last urgent byte + 1)
	binary.BigEndian.PutUint16(seg1[ipHdrLen+18:ipHdrLen+20], uint16(oobPos))

	// Update IP total length
	binary.BigEndian.PutUint16(seg1[2:4], uint16(seg1Len))

	// Keep original sequence and ID for first segment
	binary.BigEndian.PutUint32(seg1[ipHdrLen+4:ipHdrLen+8], seq)
	binary.BigEndian.PutUint16(seg1[4:6], id)

	// ===== Second segment: remaining data =====
	seg2Len := payloadStart + (payloadLen - oobPos)
	seg2 := make([]byte, seg2Len)

	// Copy headers
	copy(seg2[:payloadStart], packet[:payloadStart])
	// Copy remaining payload
	copy(seg2[payloadStart:], packet[payloadStart+oobPos:])

	// Update sequence number (advance by oobPos bytes)
	binary.BigEndian.PutUint32(seg2[ipHdrLen+4:ipHdrLen+8], seq+uint32(oobPos))

	// Increment IP ID
	binary.BigEndian.PutUint16(seg2[4:6], id+1)

	// Update IP total length
	binary.BigEndian.PutUint16(seg2[2:4], uint16(seg2Len))

	// Fix checksums for both segments
	sock.FixIPv4Checksum(seg1[:ipHdrLen])
	sock.FixTCPChecksum(seg1)
	sock.FixIPv4Checksum(seg2[:ipHdrLen])
	sock.FixTCPChecksum(seg2)

	// Send segments
	seg2delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		// Send second segment first (byedpi style)
		_ = w.sock.SendIPv4(seg2, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg1, dst)
		log.Tracef("OOB: Sent %d + %d bytes (reversed)", len(seg2), len(seg1))
	} else {
		// Normal order
		_ = w.sock.SendIPv4(seg1, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg2, dst)
		log.Tracef("OOB: Sent %d + %d bytes (normal)", len(seg1), len(seg2))
	}
}

// sendOOBFragmentsV6 - IPv6 version matching byedpi logic
func (w *Worker) sendOOBFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	if cfg.Fragmentation.OOBPosition <= 0 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	ipv6HdrLen := 40
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

	// Calculate OOB split position
	oobPos := cfg.Fragmentation.OOBPosition

	// Handle middle SNI positioning
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(packet[payloadStart:]); ok {
			oobPos = sniStart + (sniEnd-sniStart)/2
			log.Tracef("OOB v6: SNI detected at %d-%d, splitting at %d", sniStart, sniEnd, oobPos)
		}
	}

	// Validate position
	if oobPos <= 0 || oobPos >= payloadLen {
		oobPos = 1
	}

	log.Tracef("OOB v6: Splitting at position %d of %d bytes (reverse=%v)",
		oobPos, payloadLen, cfg.Fragmentation.ReverseOrder)

	// Get original sequence
	seq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])

	// ===== First segment with URG flag =====
	seg1Len := payloadStart + oobPos
	seg1 := make([]byte, seg1Len)
	copy(seg1, packet[:seg1Len])

	// Set URG flag
	seg1[ipv6HdrLen+13] |= 0x20

	// Set urgent pointer
	binary.BigEndian.PutUint16(seg1[ipv6HdrLen+18:ipv6HdrLen+20], uint16(oobPos))

	// Update IPv6 payload length
	binary.BigEndian.PutUint16(seg1[4:6], uint16(seg1Len-ipv6HdrLen))

	// Keep original sequence
	binary.BigEndian.PutUint32(seg1[ipv6HdrLen+4:ipv6HdrLen+8], seq)

	// ===== Second segment =====
	seg2Len := payloadStart + (payloadLen - oobPos)
	seg2 := make([]byte, seg2Len)
	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], packet[payloadStart+oobPos:])

	// Update sequence
	binary.BigEndian.PutUint32(seg2[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(oobPos))

	// Update IPv6 payload length
	binary.BigEndian.PutUint16(seg2[4:6], uint16(seg2Len-ipv6HdrLen))

	// Fix checksums
	sock.FixTCPChecksumV6(seg1)
	sock.FixTCPChecksumV6(seg2)

	// Send segments
	seg2delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		// Send second segment first
		_ = w.sock.SendIPv6(seg2, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg1, dst)
		log.Tracef("OOB v6: Sent %d + %d bytes (reversed)", len(seg2), len(seg1))
	} else {
		// Normal order
		_ = w.sock.SendIPv6(seg1, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg2, dst)
		log.Tracef("OOB v6: Sent %d + %d bytes (normal)", len(seg1), len(seg2))
	}
}
