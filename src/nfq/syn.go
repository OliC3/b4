package nfq

import (
	"encoding/binary"
	"net"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
)

// sendFakeSyn sends a fake SYN packet with payload to confuse DPI systems
func (w *Worker) sendFakeSyn(set *config.SetConfig, raw []byte, ipHdrLen, tcpHdrLen int) {
	// Determine fake payload length
	var fakePayload []byte
	switch set.Faking.SNIType {
	case config.FakePayloadDefault2:
		fakePayload = sock.FakeSNI2
	default:
		fakePayload = sock.FakeSNI1
	}

	fakePayloadLen := 0
	if set.TCP.SynFakeLen > 0 {
		fakePayloadLen = set.TCP.SynFakeLen
		if fakePayloadLen > len(fakePayload) {
			fakePayloadLen = len(fakePayload)
		}
	}
	totalLen := ipHdrLen + tcpHdrLen + fakePayloadLen
	fakePkt := make([]byte, totalLen)

	copy(fakePkt[:ipHdrLen+tcpHdrLen], raw[:ipHdrLen+tcpHdrLen])

	copy(fakePkt[ipHdrLen+tcpHdrLen:], fakePayload[:fakePayloadLen])

	binary.BigEndian.PutUint16(fakePkt[2:4], uint16(totalLen))

	w.applySynFakingStrategy(fakePkt, ipHdrLen, set)

	sock.FixIPv4Checksum(fakePkt[:ipHdrLen])
	sock.FixTCPChecksum(fakePkt)

	if set.Faking.Strategy == "tcp_check" {
		fakePkt[ipHdrLen+16] ^= 0xFF // Flip checksum byte
	}

	dst := net.IP(fakePkt[16:20])
	if err := w.sock.SendIPv4(fakePkt, dst); err != nil {
		log.Errorf("Failed to send fake SYN: %v", err)
	}
}

// sendFakeSynV6 sends a fake SYN packet for IPv6
func (w *Worker) sendFakeSynV6(set *config.SetConfig, raw []byte, ipHdrLen, tcpHdrLen int) {
	var fakePayload []byte
	switch set.Faking.SNIType {
	case config.FakePayloadDefault2:
		fakePayload = sock.FakeSNI2
	default:
		fakePayload = sock.FakeSNI1
	}

	fakePayloadLen := 0
	if set.TCP.SynFakeLen > 0 {
		fakePayloadLen = set.TCP.SynFakeLen
		if fakePayloadLen > len(fakePayload) {
			fakePayloadLen = len(fakePayload)
		}
	}

	totalLen := ipHdrLen + tcpHdrLen + fakePayloadLen
	fakePkt := make([]byte, totalLen)

	copy(fakePkt[:ipHdrLen+tcpHdrLen], raw[:ipHdrLen+tcpHdrLen])

	copy(fakePkt[ipHdrLen+tcpHdrLen:], fakePayload[:fakePayloadLen])

	payloadLen := tcpHdrLen + fakePayloadLen
	binary.BigEndian.PutUint16(fakePkt[4:6], uint16(payloadLen))

	w.applySynFakingStrategyV6(fakePkt, ipHdrLen, set)

	sock.FixTCPChecksumV6(fakePkt)

	if set.Faking.Strategy == "tcp_check" {
		fakePkt[ipHdrLen+16] ^= 0xFF
	}

	dst := net.IP(fakePkt[24:40])

	if err := w.sock.SendIPv6(fakePkt, dst); err != nil {
		log.Errorf("Failed to send fake SYN v6: %v", err)
	}
}

// applySynFakingStrategy modifies the fake SYN packet according to configured strategy
func (w *Worker) applySynFakingStrategy(pkt []byte, ipHdrLen int, set *config.SetConfig) {
	switch set.Faking.Strategy {
	case "ttl":
		pkt[8] = set.Faking.TTL

	case "randseq":
		seq := binary.BigEndian.Uint32(pkt[ipHdrLen+4 : ipHdrLen+8])
		seq += uint32(set.Faking.SeqOffset)
		if set.Faking.SeqOffset == 0 {
			seq += 100000 // Default random offset
		}
		binary.BigEndian.PutUint32(pkt[ipHdrLen+4:ipHdrLen+8], seq)

	case "pastseq":
		seq := binary.BigEndian.Uint32(pkt[ipHdrLen+4 : ipHdrLen+8])
		offset := uint32(set.Faking.SeqOffset)
		if offset == 0 {
			offset = 10000 // Default offset
		}
		if seq > offset {
			seq -= offset
		}
		binary.BigEndian.PutUint32(pkt[ipHdrLen+4:ipHdrLen+8], seq)
	}
}

// applySynFakingStrategyV6 modifies the fake SYN packet for IPv6
func (w *Worker) applySynFakingStrategyV6(pkt []byte, ipHdrLen int, set *config.SetConfig) {
	switch set.Faking.Strategy {
	case "ttl":
		pkt[7] = set.Faking.TTL

	case "randseq":
		seq := binary.BigEndian.Uint32(pkt[ipHdrLen+4 : ipHdrLen+8])
		seq += uint32(set.Faking.SeqOffset)
		if set.Faking.SeqOffset == 0 {
			seq += 100000
		}
		binary.BigEndian.PutUint32(pkt[ipHdrLen+4:ipHdrLen+8], seq)

	case "pastseq":
		seq := binary.BigEndian.Uint32(pkt[ipHdrLen+4 : ipHdrLen+8])
		offset := uint32(set.Faking.SeqOffset)
		if offset == 0 {
			offset = 10000
		}
		if seq > offset {
			seq -= offset
		}
		binary.BigEndian.PutUint32(pkt[ipHdrLen+4:ipHdrLen+8], seq)

	}
}
