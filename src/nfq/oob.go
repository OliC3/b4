package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
)

// sendWithOOB sends TCP packet with OOB (urgent) data
// Supports both normal and reverse order based on config.OOBReverse flag
func (w *Worker) sendOOBFragments(cfg *config.SetConfig, packet []byte, dst net.IP) bool {
	if cfg.Fragmentation.OOBPosition <= 0 {
		return false
	}

	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		return false
	}

	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen <= 0 {
		return false
	}

	oobPos := cfg.Fragmentation.OOBPosition

	// Handle +s modifier (add SNI offset) if needed
	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(packet[payloadStart:]); ok {
			oobPos += sniStart + (sniEnd-sniStart)/2
		}
	}

	// Validate position
	if oobPos <= 0 || oobPos >= payloadLen {
		oobPos = 1 // Default to 1 byte
	}

	log.Tracef("OOB: Splitting at position %d (reverse=%v)", oobPos, cfg.Fragmentation.ReverseOrder)

	// Get OOB character
	oobChar := cfg.Fragmentation.OOBChar
	if oobChar == 0 {
		oobChar = 'x' // default
	}

	// Build segment with OOB byte (first oobPos bytes + OOB char)
	oobSegLen := payloadStart + oobPos + 1 // +1 for the OOB byte
	oobSeg := make([]byte, oobSegLen)

	// Copy headers and first oobPos bytes of payload
	copy(oobSeg[:payloadStart], packet[:payloadStart])
	copy(oobSeg[payloadStart:payloadStart+oobPos], packet[payloadStart:payloadStart+oobPos])

	oobSeg[payloadStart+oobPos] = oobChar

	oobSeg[ipHdrLen+13] |= 0x20

	binary.BigEndian.PutUint16(oobSeg[ipHdrLen+18:ipHdrLen+20], uint16(oobPos+1))

	binary.BigEndian.PutUint16(oobSeg[2:4], uint16(oobSegLen))

	regularSegLen := payloadStart + (payloadLen - oobPos)
	regularSeg := make([]byte, regularSegLen)

	// Copy headers
	copy(regularSeg[:payloadStart], packet[:payloadStart])
	copy(regularSeg[payloadStart:], packet[payloadStart+oobPos:])

	seq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	id := binary.BigEndian.Uint16(packet[4:6])

	if cfg.Fragmentation.ReverseOrder {
		binary.BigEndian.PutUint32(oobSeg[ipHdrLen+4:ipHdrLen+8], seq)
		binary.BigEndian.PutUint16(oobSeg[4:6], id+1)
		binary.BigEndian.PutUint32(regularSeg[ipHdrLen+4:ipHdrLen+8], seq+uint32(oobPos))
	} else {
		binary.BigEndian.PutUint32(regularSeg[ipHdrLen+4:ipHdrLen+8], seq+uint32(oobPos))
		binary.BigEndian.PutUint16(regularSeg[4:6], id+1)
	}

	binary.BigEndian.PutUint16(regularSeg[2:4], uint16(regularSegLen))

	// Fix checksums AFTER all modifications
	sock.FixIPv4Checksum(oobSeg[:ipHdrLen])
	sock.FixTCPChecksum(oobSeg)
	sock.FixIPv4Checksum(regularSeg[:ipHdrLen])
	sock.FixTCPChecksum(regularSeg)

	seg2delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(regularSeg, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(oobSeg, dst)
		log.Tracef("OOB: Sent %d + %d bytes (reversed)", len(regularSeg), len(oobSeg))
	} else {
		_ = w.sock.SendIPv4(oobSeg, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(regularSeg, dst)
		log.Tracef("OOB: Sent %d + %d bytes (normal)", len(oobSeg), len(regularSeg))
	}

	return true
}

func (w *Worker) sendOOBFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) bool {
	if cfg.Fragmentation.OOBPosition <= 0 {
		return false
	}

	ipv6HdrLen := 40
	if len(packet) < ipv6HdrLen+20 {
		return false
	}

	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen <= 0 {
		return false
	}

	oobPos := cfg.Fragmentation.OOBPosition

	if cfg.Fragmentation.MiddleSNI {
		if sniStart, sniEnd, ok := locateSNI(packet[payloadStart:]); ok {
			oobPos += sniStart + (sniEnd-sniStart)/2
		}
	}

	if oobPos <= 0 || oobPos >= payloadLen {
		oobPos = 1
	}

	log.Tracef("OOB v6: Splitting at position %d (reverse=%v)", oobPos, cfg.Fragmentation.ReverseOrder)

	oobChar := cfg.Fragmentation.OOBChar
	if oobChar == 0 {
		oobChar = 'x'
	}

	// Build OOB segment
	oobSegLen := payloadStart + oobPos + 1
	oobSeg := make([]byte, oobSegLen)
	copy(oobSeg[:payloadStart], packet[:payloadStart])
	copy(oobSeg[payloadStart:payloadStart+oobPos], packet[payloadStart:payloadStart+oobPos])
	oobSeg[payloadStart+oobPos] = oobChar
	oobSeg[ipv6HdrLen+13] |= 0x20
	binary.BigEndian.PutUint16(oobSeg[ipv6HdrLen+18:ipv6HdrLen+20], uint16(oobPos+1))

	// Build regular segment
	regularSegLen := payloadStart + (payloadLen - oobPos)
	regularSeg := make([]byte, regularSegLen)
	copy(regularSeg[:payloadStart], packet[:payloadStart])
	copy(regularSeg[payloadStart:], packet[payloadStart+oobPos:])

	// Handle sequence numbers
	seq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])

	if cfg.Fragmentation.ReverseOrder {
		binary.BigEndian.PutUint32(oobSeg[ipv6HdrLen+4:ipv6HdrLen+8], seq)
		binary.BigEndian.PutUint32(regularSeg[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(oobPos))
	} else {
		binary.BigEndian.PutUint32(regularSeg[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(oobPos))
	}

	// Update lengths and fix checksums AFTER all modifications
	binary.BigEndian.PutUint16(oobSeg[4:6], uint16(oobSegLen-ipv6HdrLen))
	binary.BigEndian.PutUint16(regularSeg[4:6], uint16(regularSegLen-ipv6HdrLen))
	sock.FixTCPChecksumV6(oobSeg)
	sock.FixTCPChecksumV6(regularSeg)

	seg2delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv6(regularSeg, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(oobSeg, dst)
		log.Tracef("OOB v6: Sent %d + %d bytes (reversed)", len(regularSeg), len(oobSeg))
	} else {
		_ = w.sock.SendIPv6(oobSeg, dst)
		if seg2delay > 0 {
			time.Sleep(time.Duration(seg2delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(regularSeg, dst)
		log.Tracef("OOB v6: Sent %d + %d bytes (normal)", len(oobSeg), len(regularSeg))
	}

	return true
}
