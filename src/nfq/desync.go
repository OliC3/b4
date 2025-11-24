package nfq

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
)

// DesyncAttacker handles TCP desynchronization attacks
type DesyncAttacker struct {
	mode  string
	ttl   uint8
	count int
}

// NewDesyncAttacker creates a new desync attacker
func NewDesyncAttacker(cfg *config.TCPConfig) *DesyncAttacker {
	return &DesyncAttacker{
		mode:  cfg.DesyncMode,
		ttl:   cfg.DesyncTTL,
		count: cfg.DesyncCount,
	}
}

// ExecuteDesyncIPv4 performs desync attack for IPv4
func (w *Worker) ExecuteDesyncIPv4(cfg *config.SetConfig, packet []byte, dst net.IP) {
	if cfg.TCP.DesyncMode == "off" {
		return
	}

	da := NewDesyncAttacker(&cfg.TCP)

	switch da.mode {
	case "rst":
		w.sendDesyncRST(packet, dst, da)
	case "fin":
		w.sendDesyncFIN(packet, dst, da)
	case "ack":
		w.sendDesyncACK(packet, dst, da)
	case "combo":
		w.sendDesyncCombo(packet, dst, da)
	case "full":
		w.sendDesyncFull(packet, dst, da)
	default:
		w.sendDesyncCombo(packet, dst, da)
	}
}

// sendDesyncRST sends fake RST packets with bad checksums
func (w *Worker) sendDesyncRST(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		return
	}

	log.Tracef("Desync: Sending %d fake RST packets", da.count)

	// Get original sequence number only
	origSeq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	// REMOVED: origAck - not needed for RST packets

	for i := 0; i < da.count; i++ {
		fake := make([]byte, ipHdrLen+20) // Just headers
		copy(fake, packet[:ipHdrLen+20])

		// Set RST flag (or RST+ACK for variation)
		if i%2 == 0 {
			fake[ipHdrLen+13] = 0x04 // RST only
		} else {
			fake[ipHdrLen+13] = 0x14 // RST+ACK (some DPI expect this)
			// Keep original ACK number for RST+ACK
		}

		// Modify sequence number slightly
		var seqOffset int32
		switch i {
		case 0:
			seqOffset = -10000 // Past sequence
		case 1:
			seqOffset = 0 // Current sequence
		case 2:
			seqOffset = 10000 // Future sequence
		default:
			seqOffset = int32(i * 5000)
		}

		newSeq := uint32(int32(origSeq) + seqOffset)
		binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], newSeq)

		// Clear ACK number only for pure RST (not RST+ACK)
		if fake[ipHdrLen+13] == 0x04 {
			binary.BigEndian.PutUint32(fake[ipHdrLen+8:ipHdrLen+12], 0)
		}

		// Set low TTL
		fake[8] = da.ttl

		// Update IP length
		binary.BigEndian.PutUint16(fake[2:4], uint16(ipHdrLen+20))

		// Fix IP checksum
		sock.FixIPv4Checksum(fake[:ipHdrLen])

		// Calculate correct TCP checksum first
		sock.FixTCPChecksum(fake)

		// Now corrupt the checksum deliberately
		fake[ipHdrLen+16] ^= 0xFF
		fake[ipHdrLen+17] ^= 0xFF

		// Send the fake RST with bad checksum
		_ = w.sock.SendIPv4(fake, dst)

		// Small delay between packets
		time.Sleep(100 * time.Microsecond)
	}
}

// sendDesyncFIN sends fake FIN packets
func (w *Worker) sendDesyncFIN(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		return
	}

	log.Tracef("Desync: Sending %d fake FIN packets", da.count)

	origSeq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	origAck := binary.BigEndian.Uint32(packet[ipHdrLen+8 : ipHdrLen+12])

	for i := 0; i < da.count; i++ {
		fake := make([]byte, ipHdrLen+20)
		copy(fake, packet[:ipHdrLen+20])

		// Set FIN+ACK flags
		fake[ipHdrLen+13] = 0x11 // FIN | ACK

		// Use past sequence numbers (confuse DPI state)
		seqOffset := uint32(50000 + i*10000) // Way in the past
		if origSeq > seqOffset {
			binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], origSeq-seqOffset)
		} else {
			binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], 1)
		}

		// Keep ACK number
		binary.BigEndian.PutUint32(fake[ipHdrLen+8:ipHdrLen+12], origAck)

		// Set very low TTL
		fake[8] = da.ttl

		// Update lengths
		binary.BigEndian.PutUint16(fake[2:4], uint16(ipHdrLen+20))

		// Fix checksums
		sock.FixIPv4Checksum(fake[:ipHdrLen])
		sock.FixTCPChecksum(fake)

		// Corrupt checksum on even packets
		if i%2 == 0 {
			fake[ipHdrLen+16] ^= 0xAA
		}

		_ = w.sock.SendIPv4(fake, dst)
		time.Sleep(200 * time.Microsecond)
	}
}

// sendDesyncACK sends fake ACK packets with wrong sequence
func (w *Worker) sendDesyncACK(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		return
	}

	log.Tracef("Desync: Sending %d fake ACK packets", da.count)

	origSeq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])
	origAck := binary.BigEndian.Uint32(packet[ipHdrLen+8 : ipHdrLen+12])

	for i := 0; i < da.count; i++ {
		fake := make([]byte, ipHdrLen+20)
		copy(fake, packet[:ipHdrLen+20])

		// ACK only
		fake[ipHdrLen+13] = 0x10

		// Random sequence far in future
		var rb [4]byte
		rand.Read(rb[:])
		futureSeq := origSeq + binary.BigEndian.Uint32(rb[:])
		binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], futureSeq)

		// Random ACK number
		futureAck := origAck + uint32(100000*(i+1))
		binary.BigEndian.PutUint32(fake[ipHdrLen+8:ipHdrLen+12], futureAck)

		// Low TTL
		if uint8(i) >= da.ttl {
			fake[8] = 1
		} else {
			fake[8] = da.ttl - uint8(i)
		}

		if fake[8] < 1 {
			fake[8] = 1
		}

		binary.BigEndian.PutUint16(fake[2:4], uint16(ipHdrLen+20))

		sock.FixIPv4Checksum(fake[:ipHdrLen])
		sock.FixTCPChecksum(fake)

		// Always corrupt ACK checksums
		fake[ipHdrLen+17] = ^fake[ipHdrLen+17]

		_ = w.sock.SendIPv4(fake, dst)
		time.Sleep(50 * time.Microsecond)
	}
}

// sendDesyncCombo sends combination of RST, FIN, ACK
func (w *Worker) sendDesyncCombo(packet []byte, dst net.IP, da *DesyncAttacker) {
	log.Tracef("Desync: Combo attack (RST+FIN+ACK)")

	// First send RST
	w.sendDesyncRST(packet, dst, &DesyncAttacker{ttl: da.ttl, count: 1})
	time.Sleep(500 * time.Microsecond)

	// Then FIN
	w.sendDesyncFIN(packet, dst, &DesyncAttacker{ttl: da.ttl, count: 1})
	time.Sleep(500 * time.Microsecond)

	// Then ACK flood
	w.sendDesyncACK(packet, dst, &DesyncAttacker{ttl: da.ttl, count: 2})
}

// sendDesyncFull sends full sequence of all desync types
func (w *Worker) sendDesyncFull(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		return
	}

	log.Tracef("Desync: Full attack sequence")

	origSeq := binary.BigEndian.Uint32(packet[ipHdrLen+4 : ipHdrLen+8])

	// 1. Send fake SYN with bad checksum (confuse connection start)
	synFake := make([]byte, ipHdrLen+20)
	copy(synFake, packet[:ipHdrLen+20])
	synFake[ipHdrLen+13] = 0x02 // SYN only
	binary.BigEndian.PutUint32(synFake[ipHdrLen+4:ipHdrLen+8], origSeq-100000)
	synFake[8] = 1 // TTL=1
	binary.BigEndian.PutUint16(synFake[2:4], uint16(ipHdrLen+20))
	sock.FixIPv4Checksum(synFake[:ipHdrLen])
	sock.FixTCPChecksum(synFake)
	synFake[ipHdrLen+16] = 0xFF // Corrupt checksum
	_ = w.sock.SendIPv4(synFake, dst)

	time.Sleep(100 * time.Microsecond)

	// 2. Send overlapping RST packets
	for i := 0; i < 3; i++ {
		rstFake := make([]byte, ipHdrLen+20)
		copy(rstFake, packet[:ipHdrLen+20])
		rstFake[ipHdrLen+13] = 0x04 // RST

		// Overlapping sequence numbers
		seq := origSeq + uint32(i*100)
		binary.BigEndian.PutUint32(rstFake[ipHdrLen+4:ipHdrLen+8], seq)

		rstFake[8] = 2 // TTL=2
		binary.BigEndian.PutUint16(rstFake[2:4], uint16(ipHdrLen+20))
		sock.FixIPv4Checksum(rstFake[:ipHdrLen])
		sock.FixTCPChecksum(rstFake)

		// Different corruption patterns
		switch i {
		case 0:
			rstFake[ipHdrLen+16] ^= 0xFF // Flip all bits
		case 1:
			rstFake[ipHdrLen+17] ^= 0xAA // Pattern
		case 2:
			rstFake[ipHdrLen+16] = 0x00 // Zero checksum
			rstFake[ipHdrLen+17] = 0x00
		}

		_ = w.sock.SendIPv4(rstFake, dst)
		time.Sleep(50 * time.Microsecond)
	}

	// 3. Send fake PUSH with no data
	pushFake := make([]byte, ipHdrLen+20)
	copy(pushFake, packet[:ipHdrLen+20])
	pushFake[ipHdrLen+13] = 0x18 // PSH | ACK
	pushFake[8] = 1
	binary.BigEndian.PutUint16(pushFake[2:4], uint16(ipHdrLen+20))
	sock.FixIPv4Checksum(pushFake[:ipHdrLen])
	sock.FixTCPChecksum(pushFake)
	pushFake[ipHdrLen+17] = ^pushFake[ipHdrLen+17]
	_ = w.sock.SendIPv4(pushFake, dst)

	time.Sleep(100 * time.Microsecond)

	// 4. Send FIN|PSH|URG combo (invalid combination)
	urgFake := make([]byte, ipHdrLen+20)
	copy(urgFake, packet[:ipHdrLen+20])
	urgFake[ipHdrLen+13] = 0x39                                          // FIN | PSH | URG | ACK
	binary.BigEndian.PutUint16(urgFake[ipHdrLen+18:ipHdrLen+20], 0xFFFF) // Max urgent pointer
	urgFake[8] = da.ttl
	binary.BigEndian.PutUint16(urgFake[2:4], uint16(ipHdrLen+20))
	sock.FixIPv4Checksum(urgFake[:ipHdrLen])
	sock.FixTCPChecksum(urgFake)
	urgFake[ipHdrLen+16] = 0x12
	urgFake[ipHdrLen+17] = 0x34
	_ = w.sock.SendIPv4(urgFake, dst)
}

// ExecuteDesyncIPv6 performs desync attack for IPv6
func (w *Worker) ExecuteDesyncIPv6(cfg *config.SetConfig, packet []byte, dst net.IP) {
	if cfg.TCP.DesyncMode == "off" {
		return
	}

	da := NewDesyncAttacker(&cfg.TCP)

	switch da.mode {
	case "rst":
		w.sendDesyncRSTv6(packet, dst, da)
	case "fin":
		w.sendDesyncFINv6(packet, dst, da)
	case "ack":
		w.sendDesyncACKv6(packet, dst, da)
	case "combo":
		w.sendDesyncCombov6(packet, dst, da)
	case "full":
		w.sendDesyncFullv6(packet, dst)
	default:
		w.sendDesyncCombov6(packet, dst, da)
	}
}

// sendDesyncRSTv6 for IPv6
func (w *Worker) sendDesyncRSTv6(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipv6HdrLen := 40
	if len(packet) < ipv6HdrLen+20 {
		return
	}

	origSeq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])
	// REMOVED: origAck - not needed

	for i := 0; i < da.count; i++ {
		fake := make([]byte, ipv6HdrLen+20)
		copy(fake, packet[:ipv6HdrLen+20])

		// Alternate between RST and RST+ACK
		if i%2 == 0 {
			fake[ipv6HdrLen+13] = 0x04 // RST only
		} else {
			fake[ipv6HdrLen+13] = 0x14 // RST+ACK
		}

		var seqOffset int32
		switch i {
		case 0:
			seqOffset = -10000
		case 1:
			seqOffset = 0
		case 2:
			seqOffset = 10000
		default:
			seqOffset = int32(i * 5000)
		}

		newSeq := uint32(int32(origSeq) + seqOffset)
		binary.BigEndian.PutUint32(fake[ipv6HdrLen+4:ipv6HdrLen+8], newSeq)

		// Clear ACK only for pure RST
		if fake[ipv6HdrLen+13] == 0x04 {
			binary.BigEndian.PutUint32(fake[ipv6HdrLen+8:ipv6HdrLen+12], 0)
		}

		fake[7] = da.ttl                          // Hop limit
		binary.BigEndian.PutUint16(fake[4:6], 20) // Payload length

		sock.FixTCPChecksumV6(fake)

		// Corrupt checksum
		fake[ipv6HdrLen+16] ^= 0xFF
		fake[ipv6HdrLen+17] ^= 0xFF

		_ = w.sock.SendIPv6(fake, dst)
		time.Sleep(100 * time.Microsecond)
	}
}

// sendDesyncFINv6 for IPv6
func (w *Worker) sendDesyncFINv6(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipv6HdrLen := 40
	if len(packet) < ipv6HdrLen+20 {
		return
	}

	origSeq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])
	origAck := binary.BigEndian.Uint32(packet[ipv6HdrLen+8 : ipv6HdrLen+12])

	for i := 0; i < da.count; i++ {
		fake := make([]byte, ipv6HdrLen+20)
		copy(fake, packet[:ipv6HdrLen+20])

		fake[ipv6HdrLen+13] = 0x11 // FIN | ACK

		seqOffset := uint32(50000 + i*10000)
		if origSeq > seqOffset {
			binary.BigEndian.PutUint32(fake[ipv6HdrLen+4:ipv6HdrLen+8], origSeq-seqOffset)
		} else {
			binary.BigEndian.PutUint32(fake[ipv6HdrLen+4:ipv6HdrLen+8], 1)
		}

		binary.BigEndian.PutUint32(fake[ipv6HdrLen+8:ipv6HdrLen+12], origAck)

		fake[7] = da.ttl
		binary.BigEndian.PutUint16(fake[4:6], 20)

		sock.FixTCPChecksumV6(fake)

		if i%2 == 0 {
			fake[ipv6HdrLen+16] ^= 0xAA
		}

		_ = w.sock.SendIPv6(fake, dst)
		time.Sleep(200 * time.Microsecond)
	}
}

// sendDesyncACKv6 for IPv6
func (w *Worker) sendDesyncACKv6(packet []byte, dst net.IP, da *DesyncAttacker) {
	ipv6HdrLen := 40
	if len(packet) < ipv6HdrLen+20 {
		return
	}

	origSeq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])
	origAck := binary.BigEndian.Uint32(packet[ipv6HdrLen+8 : ipv6HdrLen+12])

	for i := 0; i < da.count; i++ {
		fake := make([]byte, ipv6HdrLen+20)
		copy(fake, packet[:ipv6HdrLen+20])

		fake[ipv6HdrLen+13] = 0x10 // ACK

		var rb [4]byte
		rand.Read(rb[:])
		futureSeq := origSeq + binary.BigEndian.Uint32(rb[:])
		binary.BigEndian.PutUint32(fake[ipv6HdrLen+4:ipv6HdrLen+8], futureSeq)

		futureAck := origAck + uint32(100000*(i+1))
		binary.BigEndian.PutUint32(fake[ipv6HdrLen+8:ipv6HdrLen+12], futureAck)

		if uint8(i) >= da.ttl {
			fake[7] = 1
		} else {
			fake[7] = da.ttl - uint8(i)
		}

		binary.BigEndian.PutUint16(fake[4:6], 20)

		sock.FixTCPChecksumV6(fake)
		fake[ipv6HdrLen+17] = ^fake[ipv6HdrLen+17]

		_ = w.sock.SendIPv6(fake, dst)
		time.Sleep(50 * time.Microsecond)
	}
}

// sendDesyncCombov6 for IPv6
func (w *Worker) sendDesyncCombov6(packet []byte, dst net.IP, da *DesyncAttacker) {
	w.sendDesyncRSTv6(packet, dst, &DesyncAttacker{ttl: da.ttl, count: 1})
	time.Sleep(500 * time.Microsecond)

	w.sendDesyncFINv6(packet, dst, &DesyncAttacker{ttl: da.ttl, count: 1})
	time.Sleep(500 * time.Microsecond)

	w.sendDesyncACKv6(packet, dst, &DesyncAttacker{ttl: da.ttl, count: 2})
}

// sendDesyncFullv6 for IPv6
func (w *Worker) sendDesyncFullv6(packet []byte, dst net.IP) {
	ipv6HdrLen := 40
	if len(packet) < ipv6HdrLen+20 {
		return
	}

	origSeq := binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8])

	// Similar to IPv4 but adapted for IPv6
	synFake := make([]byte, ipv6HdrLen+20)
	copy(synFake, packet[:ipv6HdrLen+20])
	synFake[ipv6HdrLen+13] = 0x02 // SYN
	binary.BigEndian.PutUint32(synFake[ipv6HdrLen+4:ipv6HdrLen+8], origSeq-100000)
	synFake[7] = 1
	binary.BigEndian.PutUint16(synFake[4:6], 20)
	sock.FixTCPChecksumV6(synFake)
	synFake[ipv6HdrLen+16] = 0xFF
	_ = w.sock.SendIPv6(synFake, dst)

	// Continue with other attacks...
	time.Sleep(100 * time.Microsecond)

	// Send overlapping RSTs
	for i := 0; i < 3; i++ {
		rstFake := make([]byte, ipv6HdrLen+20)
		copy(rstFake, packet[:ipv6HdrLen+20])
		rstFake[ipv6HdrLen+13] = 0x04

		seq := origSeq + uint32(i*100)
		binary.BigEndian.PutUint32(rstFake[ipv6HdrLen+4:ipv6HdrLen+8], seq)

		rstFake[7] = 2
		binary.BigEndian.PutUint16(rstFake[4:6], 20)
		sock.FixTCPChecksumV6(rstFake)

		switch i {
		case 0:
			rstFake[ipv6HdrLen+16] ^= 0xFF
		case 1:
			rstFake[ipv6HdrLen+17] ^= 0xAA
		case 2:
			rstFake[ipv6HdrLen+16] = 0x00
			rstFake[ipv6HdrLen+17] = 0x00
		}

		_ = w.sock.SendIPv6(rstFake, dst)
		time.Sleep(50 * time.Microsecond)
	}
}
