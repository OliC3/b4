package sock

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/daniellavrushin/b4/config"
)

func BuildFakeSNIPacketV4(original []byte, cfg *config.SetConfig) []byte {
	if len(original) < 40 || original[0]>>4 != 4 {
		return nil
	}

	ipHdrLen := int((original[0] & 0x0F) * 4)
	tcpHdrLen := int((original[ipHdrLen+12] >> 4) * 4)

	var fakePayload []byte
	switch cfg.Faking.SNIType {
	case config.FakePayloadRandom:
		fakePayload = make([]byte, 1200)
		rand.Read(fakePayload)
	case config.FakePayloadCustom:
		fakePayload = []byte(cfg.Faking.CustomPayload)
	case config.FakePayloadDefault1:
		fakePayload = FakeSNI1
	case config.FakePayloadDefault2:
		fakePayload = FakeSNI2
	default:
		fakePayload = FakeSNI1
	}

	fakeLen := ipHdrLen + tcpHdrLen + len(fakePayload)
	fake := make([]byte, fakeLen)
	copy(fake[:ipHdrLen+tcpHdrLen], original[:ipHdrLen+tcpHdrLen])
	copy(fake[ipHdrLen+tcpHdrLen:], fakePayload)

	binary.BigEndian.PutUint16(fake[2:4], uint16(fakeLen))

	off := cfg.Faking.SeqOffset
	if off <= 0 {
		off = 10000
	}

	switch cfg.Faking.Strategy {
	case "ttl":
		fake[8] = cfg.Faking.TTL
	case "pastseq":
		off := uint32(cfg.Faking.SeqOffset)
		if off == 0 {
			off = 8192
		}
		seq := binary.BigEndian.Uint32(fake[ipHdrLen+4 : ipHdrLen+8])
		binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], seq-off)
	case "randseq":
		dlen := len(original) - ipHdrLen - tcpHdrLen
		if cfg.Faking.SeqOffset == 0 {
			var r [4]byte
			rand.Read(r[:])
			binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], binary.BigEndian.Uint32(r[:]))
		} else {
			seq := binary.BigEndian.Uint32(fake[ipHdrLen+4 : ipHdrLen+8])
			off := uint32(cfg.Faking.SeqOffset) + uint32(dlen)
			binary.BigEndian.PutUint32(fake[ipHdrLen+4:ipHdrLen+8], seq-off)
		}
	case "tcp_check":
	default:
	}

	FixIPv4Checksum(fake[:ipHdrLen])
	FixTCPChecksum(fake)

	if cfg.Faking.Strategy == "tcp_check" {
		fake[ipHdrLen+16] ^= 0xFF
	}

	return fake
}

func FixIPv4Checksum(ip []byte) {
	if len(ip) < 20 {
		return
	}

	ip[10], ip[11] = 0, 0
	ihl := int((ip[0] & 0x0F) * 4)
	var sum uint32

	for i := 0; i < ihl; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(ip[i : i+2]))
	}

	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	binary.BigEndian.PutUint16(ip[10:12], ^uint16(sum))
}

func FixTCPChecksum(packet []byte) {
	ipHdrLen := int((packet[0] & 0x0F) * 4)
	tcpOffset := ipHdrLen
	packet[tcpOffset+16] = 0
	packet[tcpOffset+17] = 0
	totalLen := binary.BigEndian.Uint16(packet[2:4])
	tcpLen := int(totalLen) - ipHdrLen
	pseudo := make([]byte, 12)
	copy(pseudo[0:4], packet[12:16])
	copy(pseudo[4:8], packet[16:20])
	pseudo[9] = 6
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(tcpLen))
	var sum uint32
	for i := 0; i < len(pseudo); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(pseudo[i : i+2]))
	}
	tcp := packet[tcpOffset : tcpOffset+tcpLen]
	for i := 0; i+1 < len(tcp); i += 2 {
		sum += uint32(binary.BigEndian.Uint16(tcp[i : i+2]))
	}
	if len(tcp)%2 == 1 {
		sum += uint32(tcp[len(tcp)-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	checksum := ^uint16(sum)
	binary.BigEndian.PutUint16(packet[tcpOffset+16:tcpOffset+18], checksum)
}
