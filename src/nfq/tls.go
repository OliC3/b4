package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

// splitTLSRecord splits a ClientHello into multiple TLS records
func (w *Worker) sendTLSFragments(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen
	payload := packet[payloadStart:]

	// Validate TLS record
	if len(payload) < 5 || payload[0] != 0x16 {
		_ = w.sock.SendIPv4(packet, dst)
		return
	}

	splitPos := cfg.Fragmentation.TLSRecordPosition
	if splitPos <= 0 {
		splitPos = 1
	}

	// TLS record: [type(1)][version(2)][length(2)][data...]
	recordLen := int(binary.BigEndian.Uint16(payload[3:5]))
	if 5+recordLen > len(payload) {
		recordLen = len(payload) - 5
	}

	if splitPos >= recordLen {
		splitPos = recordLen / 2
	}

	// Build first TLS record (first part of ClientHello)
	rec1Len := 5 + splitPos
	pkt1 := make([]byte, payloadStart+rec1Len)
	copy(pkt1[:payloadStart], packet[:payloadStart])
	pkt1[payloadStart] = 0x16                                                         // TLS Handshake
	copy(pkt1[payloadStart+1:payloadStart+3], payload[1:3])                           // Version
	binary.BigEndian.PutUint16(pkt1[payloadStart+3:payloadStart+5], uint16(splitPos)) // Length
	copy(pkt1[payloadStart+5:], payload[5:5+splitPos])

	// Update IP total length
	binary.BigEndian.PutUint16(pkt1[2:4], uint16(len(pkt1)))
	sock.FixIPv4Checksum(pkt1[:ipHdrLen])
	sock.FixTCPChecksum(pkt1)

	// Build second TLS record (rest of ClientHello)
	rec2DataLen := recordLen - splitPos
	rec2Len := 5 + rec2DataLen
	pkt2 := make([]byte, payloadStart+rec2Len)
	copy(pkt2[:payloadStart], packet[:payloadStart])
	pkt2[payloadStart] = 0x16                               // TLS Handshake
	copy(pkt2[payloadStart+1:payloadStart+3], payload[1:3]) // Version
	binary.BigEndian.PutUint16(pkt2[payloadStart+3:payloadStart+5], uint16(rec2DataLen))
	copy(pkt2[payloadStart+5:], payload[5+splitPos:5+recordLen])

	// Update TCP sequence for second packet
	seq := binary.BigEndian.Uint32(pkt2[ipHdrLen+4 : ipHdrLen+8])
	binary.BigEndian.PutUint32(pkt2[ipHdrLen+4:ipHdrLen+8], seq+uint32(rec1Len))

	// Update IP ID and total length
	id := binary.BigEndian.Uint16(pkt1[4:6])
	binary.BigEndian.PutUint16(pkt2[4:6], id+1)
	binary.BigEndian.PutUint16(pkt2[2:4], uint16(len(pkt2)))
	sock.FixIPv4Checksum(pkt2[:ipHdrLen])
	sock.FixTCPChecksum(pkt2)

	seg2d := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv4(pkt2, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(pkt1, dst)
	} else {
		_ = w.sock.SendIPv4(pkt1, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv4(pkt2, dst)
	}
}

// IPv6 version
func (w *Worker) sendTLSFragmentsV6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	ipv6HdrLen := 40
	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payload := packet[payloadStart:]

	if len(payload) < 5 || payload[0] != 0x16 {
		_ = w.sock.SendIPv6(packet, dst)
		return
	}

	splitPos := cfg.Fragmentation.TLSRecordPosition
	if splitPos <= 0 {
		splitPos = 1
	}

	recordLen := int(binary.BigEndian.Uint16(payload[3:5]))
	if 5+recordLen > len(payload) {
		recordLen = len(payload) - 5
	}

	if splitPos >= recordLen {
		splitPos = recordLen / 2
	}

	// First TLS record
	rec1Len := 5 + splitPos
	pkt1 := make([]byte, payloadStart+rec1Len)
	copy(pkt1[:payloadStart], packet[:payloadStart])
	pkt1[payloadStart] = 0x16
	copy(pkt1[payloadStart+1:payloadStart+3], payload[1:3])
	binary.BigEndian.PutUint16(pkt1[payloadStart+3:payloadStart+5], uint16(splitPos))
	copy(pkt1[payloadStart+5:], payload[5:5+splitPos])

	binary.BigEndian.PutUint16(pkt1[4:6], uint16(len(pkt1)-ipv6HdrLen))
	sock.FixTCPChecksumV6(pkt1)

	// Second TLS record
	rec2DataLen := recordLen - splitPos
	rec2Len := 5 + rec2DataLen
	pkt2 := make([]byte, payloadStart+rec2Len)
	copy(pkt2[:payloadStart], packet[:payloadStart])
	pkt2[payloadStart] = 0x16
	copy(pkt2[payloadStart+1:payloadStart+3], payload[1:3])
	binary.BigEndian.PutUint16(pkt2[payloadStart+3:payloadStart+5], uint16(rec2DataLen))
	copy(pkt2[payloadStart+5:], payload[5+splitPos:5+recordLen])

	seq := binary.BigEndian.Uint32(pkt2[ipv6HdrLen+4 : ipv6HdrLen+8])
	binary.BigEndian.PutUint32(pkt2[ipv6HdrLen+4:ipv6HdrLen+8], seq+uint32(rec1Len))
	binary.BigEndian.PutUint16(pkt2[4:6], uint16(len(pkt2)-ipv6HdrLen))
	sock.FixTCPChecksumV6(pkt2)

	seg2d := cfg.TCP.Seg2Delay

	if cfg.Fragmentation.ReverseOrder {
		_ = w.sock.SendIPv6(pkt2, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(pkt1, dst)
	} else {
		_ = w.sock.SendIPv6(pkt1, dst)
		if seg2d > 0 {
			time.Sleep(time.Duration(seg2d) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(pkt2, dst)
	}
}
