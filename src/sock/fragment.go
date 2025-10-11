package sock

import (
	"errors"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Fragmenter struct {
	splitPosition int  // Where to split payload (default: 1 byte)
	reverseOrder  bool // Send second fragment before first
}

func (f *Fragmenter) FragmentPacket(original gopacket.Packet, splitPos int) ([][]byte, error) {
	ipLayer := original.Layer(layers.LayerTypeIPv4)
	tcpLayer := original.Layer(layers.LayerTypeTCP)

	if ipLayer == nil || tcpLayer == nil {
		return nil, errors.New("not a TCP/IP packet")
	}

	ip := ipLayer.(*layers.IPv4)
	tcp := tcpLayer.(*layers.TCP)
	payload := tcp.Payload

	// Fragment 1: First splitPos bytes of payload
	frag1 := f.buildFragment(ip, tcp, payload[:splitPos], 0)

	// Fragment 2: Remaining payload
	frag2 := f.buildFragment(ip, tcp, payload[splitPos:], uint32(splitPos))

	fragments := [][]byte{frag1, frag2}

	// Reverse order defeats some DPI systems
	if f.reverseOrder {
		fragments = [][]byte{frag2, frag1}
	}

	return fragments, nil
}

func (f *Fragmenter) buildFragment(ip *layers.IPv4, tcp *layers.TCP, data []byte, seqOffset uint32) []byte {
	// Clone IP header
	newIP := &layers.IPv4{
		Version:    ip.Version,
		IHL:        ip.IHL,
		TOS:        ip.TOS,
		Id:         ip.Id,
		Flags:      ip.Flags,
		FragOffset: ip.FragOffset,
		TTL:        ip.TTL,
		Protocol:   ip.Protocol,
		SrcIP:      ip.SrcIP,
		DstIP:      ip.DstIP,
	}

	// Clone TCP header with adjusted sequence number
	newTCP := &layers.TCP{
		SrcPort:    tcp.SrcPort,
		DstPort:    tcp.DstPort,
		Seq:        tcp.Seq + seqOffset, // Critical: advance for second fragment
		Ack:        tcp.Ack,
		DataOffset: tcp.DataOffset,
		FIN:        tcp.FIN && seqOffset > 0,  // Only on last fragment
		SYN:        tcp.SYN && seqOffset == 0, // Only on first fragment
		RST:        tcp.RST,
		PSH:        tcp.PSH && seqOffset > 0, // Push on last fragment
		ACK:        tcp.ACK,
		URG:        tcp.URG,
		ECE:        tcp.ECE,
		CWR:        tcp.CWR,
		NS:         tcp.NS,
		Window:     tcp.Window,
		Urgent:     tcp.Urgent,
	}

	// Set network layer for checksum calculation (critical!)
	newTCP.SetNetworkLayerForChecksum(newIP)

	// Serialize with automatic checksum calculation
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true, // Must recalculate checksums
	}

	gopacket.SerializeLayers(buf, opts,
		newIP,
		newTCP,
		gopacket.Payload(data),
	)

	return buf.Bytes()
}
