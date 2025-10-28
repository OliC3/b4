package sock

import (
	"errors"

	"github.com/daniellavrushin/b4/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type Fragmenter struct {
	SplitPosition int  // Where to split payload (default: 1 byte)
	ReverseOrder  bool // Send second fragment before first
	FakeSNI       bool // Whether to fake SNI
	MiddleSplit   bool // Whether to split in middle of SNI
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
	if f.ReverseOrder {
		fragments = [][]byte{frag2, frag1}
	}
	log.Tracef("Fragmented packet into %d fragments, reverse=%v", len(fragments), f.ReverseOrder)
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

func FindSNIOffset(packet []byte) int {
	if len(packet) < 40 {
		return -1
	}

	ipHdrLen := int((packet[0] & 0x0F) * 4)
	if len(packet) < ipHdrLen+20 {
		return -1
	}

	tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
	payloadStart := ipHdrLen + tcpHdrLen

	if len(packet) <= payloadStart+5 {
		return -1
	}

	payload := packet[payloadStart:]

	// Check for TLS handshake
	if payload[0] != 0x16 || payload[1] != 0x03 {
		return -1
	}

	// Skip to extensions in Client Hello
	if len(payload) < 43 {
		return -1
	}

	// Simple SNI search - look for extension type 0x0000
	for i := 43; i < len(payload)-5; i++ {
		if payload[i] == 0x00 && payload[i+1] == 0x00 {
			// Found SNI extension
			return i
		}
	}

	return 1 // Default split at position 1
}

// Update Fragmenter to support SNI-aware fragmentation
func (f *Fragmenter) FragmentAtSNI(packet []byte, sniPosOverride int) ([][]byte, error) {
	sniOffset := FindSNIOffset(packet)
	log.Tracef("SNI offset found at %d", sniOffset)
	// Use override if provided
	if sniPosOverride > 0 {
		sniOffset = sniPosOverride
	} else if sniOffset <= 0 {
		sniOffset = 1 // Fallback
	}

	// Apply middle split if configured
	if f.MiddleSplit && sniOffset > 0 {
		ipHdrLen := int((packet[0] & 0x0F) * 4)
		tcpHdrLen := int((packet[ipHdrLen+12] >> 4) * 4)
		payloadStart := ipHdrLen + tcpHdrLen

		if len(packet) > payloadStart+sniOffset+10 {
			sniOffset += 5 // Move to middle of SNI value
		}
	}

	pkt := gopacket.NewPacket(packet, layers.LayerTypeIPv4, gopacket.Default)
	log.Tracef("Fragmenting packet at SNI offset %d", sniOffset)
	return f.FragmentPacket(pkt, sniOffset)
}
