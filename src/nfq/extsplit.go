package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// findPreSNIExtensionPoint finds a good split point BEFORE the SNI extension
// This exploits DPI that only parses complete extensions
func findPreSNIExtensionPoint(payload []byte) int {
	// TLS record header
	if len(payload) < 5 || payload[0] != 0x16 {
		return -1
	}

	pos := 5 // After TLS record header

	// Handshake header
	if pos+4 > len(payload) || payload[pos] != 0x01 {
		return -1
	}
	pos += 4

	// Version + Random
	if pos+34 > len(payload) {
		return -1
	}
	pos += 34

	// Session ID
	if pos >= len(payload) {
		return -1
	}
	sidLen := int(payload[pos])
	pos++
	pos += sidLen

	// Cipher suites
	if pos+2 > len(payload) {
		return -1
	}
	csLen := int(binary.BigEndian.Uint16(payload[pos : pos+2]))
	pos += 2 + csLen

	// Compression
	if pos >= len(payload) {
		return -1
	}
	compLen := int(payload[pos])
	pos++
	pos += compLen

	// Extensions length
	if pos+2 > len(payload) {
		return -1
	}
	extLen := int(binary.BigEndian.Uint16(payload[pos : pos+2]))
	pos += 2

	extStart := pos
	extEnd := pos + extLen
	if extEnd > len(payload) {
		extEnd = len(payload)
	}

	// Walk extensions, find a split point just before SNI (type 0)
	lastSafePos := extStart
	for pos+4 <= extEnd {
		extType := binary.BigEndian.Uint16(payload[pos : pos+2])
		extDataLen := int(binary.BigEndian.Uint16(payload[pos+2 : pos+4]))

		if extType == 0 { // SNI extension found
			// Return position just BEFORE this extension's type bytes
			// This causes DPI to see incomplete extension list on first segment
			return lastSafePos
		}

		lastSafePos = pos
		pos += 4 + extDataLen
	}

	return -1
}

func (w *Worker) sendExtSplitFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	if payloadLen < 50 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	payload := packet[payloadStart:]
	splitPos := findPreSNIExtensionPoint(payload)

	if splitPos <= 5 || splitPos >= payloadLen-10 {
		// Fallback - need meaningful split, not at very start
		w.sendTCPFragments(cfg, packet, dst)
		return
	}

	seq0 := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	id0 := binary.BigEndian.Uint16(packet[4:6])

	// Segment 1: everything before SNI extension
	seg1Len := payloadStart + splitPos
	seg1 := make([]byte, seg1Len)
	copy(seg1[:payloadStart], packet[:payloadStart])
	copy(seg1[payloadStart:], payload[:splitPos])

	binary.BigEndian.PutUint16(seg1[2:4], uint16(seg1Len))
	seg1[ipHdrLen+13] &^= 0x08 // Clear PSH
	sock.FixIPv4Checksum(seg1[:ipHdrLen])
	sock.FixTCPChecksum(seg1)

	// Segment 2: SNI extension onwards
	seg2Len := payloadStart + (payloadLen - splitPos)
	seg2 := make([]byte, seg2Len)
	copy(seg2[:payloadStart], packet[:payloadStart])
	copy(seg2[payloadStart:], payload[splitPos:])

	binary.BigEndian.PutUint32(seg2[ipHdrLen+4:ipHdrLen+8], seq0+uint32(splitPos))
	binary.BigEndian.PutUint16(seg2[4:6], id0+1)
	binary.BigEndian.PutUint16(seg2[2:4], uint16(seg2Len))
	sock.FixIPv4Checksum(seg2[:ipHdrLen])
	sock.FixTCPChecksum(seg2)

	delay := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(seg2, dst)
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg1, dst)
	} else {
		_ = w.sock.SendIPv4(seg1, dst)
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(seg2, dst)
	}
}
