package nfq

import (
	"encoding/binary"
	"net"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sock"
)

func ExtractPacketInfoV6(packet []byte) (PacketInfo, bool) {
	const ipv6HdrLen = 40
	if len(packet) < ipv6HdrLen+20 {
		return PacketInfo{}, false
	}
	tcpHdrLen := int((packet[ipv6HdrLen+12] >> 4) * 4)
	payloadStart := ipv6HdrLen + tcpHdrLen
	payloadLen := len(packet) - payloadStart

	return PacketInfo{
		IPHdrLen:     ipv6HdrLen,
		TCPHdrLen:    tcpHdrLen,
		PayloadStart: payloadStart,
		PayloadLen:   payloadLen,
		Payload:      packet[payloadStart:],
		Seq0:         binary.BigEndian.Uint32(packet[ipv6HdrLen+4 : ipv6HdrLen+8]),
		IsIPv6:       true,
	}, true
}

func (w *Worker) SendSegmentsV6(segs [][]byte, dst net.IP, cfg *config.SetConfig) {
	delay := cfg.TCP.Seg2Delay
	if cfg.Fragmentation.ReverseOrder {
		for i := len(segs) - 1; i >= 0; i-- {
			_ = w.sock.SendIPv6(segs[i], dst)
			if i > 0 && delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	} else {
		for i, seg := range segs {
			_ = w.sock.SendIPv6(seg, dst)
			if i < len(segs)-1 && delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}
}

func BuildSegmentV6(packet []byte, pi PacketInfo, payloadSlice []byte, seqOffset uint32) []byte {
	segLen := pi.PayloadStart + len(payloadSlice)
	seg := make([]byte, segLen)
	copy(seg[:pi.PayloadStart], packet[:pi.PayloadStart])
	copy(seg[pi.PayloadStart:], payloadSlice)

	binary.BigEndian.PutUint32(seg[pi.IPHdrLen+4:pi.IPHdrLen+8], pi.Seq0+seqOffset)
	binary.BigEndian.PutUint16(seg[4:6], uint16(segLen-pi.IPHdrLen))

	sock.FixTCPChecksumV6(seg)
	return seg
}

func (w *Worker) SendTwoSegmentsV6(seg1, seg2 []byte, dst net.IP, delay int, reverse bool) {
	if reverse {
		_ = w.sock.SendIPv6(seg2, dst)
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg1, dst)
	} else {
		_ = w.sock.SendIPv6(seg1, dst)
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		_ = w.sock.SendIPv6(seg2, dst)
	}
}
