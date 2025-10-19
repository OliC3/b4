package sock

import (
	"net"
	"syscall"

	"github.com/daniellavrushin/b4/log"
	"golang.org/x/sys/unix"
)

const (
	PacketMark    = 0x8000
	AVAILABLE_MTU = 1400
)

type Sender struct {
	fd4 int
	fd6 int
}

func NewSenderWithMark(mark int) (*Sender, error) {
	fd4, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, err
	}
	if err := syscall.SetsockoptInt(fd4, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		syscall.Close(fd4)
		return nil, err
	}
	if err := syscall.SetsockoptInt(fd4, syscall.SOL_SOCKET, unix.SO_MARK, mark); err != nil {
		syscall.Close(fd4)
		return nil, err
	}
	fd6, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		syscall.Close(fd4)
		return nil, err
	}
	_ = syscall.SetsockoptInt(fd6, syscall.SOL_SOCKET, unix.SO_MARK, mark)
	return &Sender{fd4: fd4, fd6: fd6}, nil
}

func NewSender() (*Sender, error) {
	return NewSenderWithMark(PacketMark)
}

func (s *Sender) SendIPv4(packet []byte, destIP net.IP) error {
	// Handle MTU splitting like youtubeUnblock
	if len(packet) > AVAILABLE_MTU {
		log.Tracef("Split packet! len=%d", len(packet))

		// Use tcp_frag equivalent - split at AVAILABLE_MTU-128
		splitPos := AVAILABLE_MTU - 128

		// Need to implement TCP fragmentation
		frag1, frag2, err := tcpFragment(packet, splitPos)
		if err != nil {
			// If fragmentation fails, try sending as-is
			return s.sendIPv4Raw(packet, destIP)
		}

		// Send first fragment
		if err := s.sendIPv4Raw(frag1, destIP); err != nil {
			return err
		}

		// Send second fragment
		return s.sendIPv4Raw(frag2, destIP)
	}

	return s.sendIPv4Raw(packet, destIP)
}

func (s *Sender) sendIPv4Raw(packet []byte, destIP net.IP) error {
	log.Tracef("Sending IPv4 packet to %s, len=%d", destIP.String(), len(packet))
	addr := syscall.SockaddrInet4{}
	copy(addr.Addr[:], destIP.To4())
	return syscall.Sendto(s.fd4, packet, 0, &addr)
}

func (s *Sender) SendIPv6(packet []byte, destIP net.IP) error {
	log.Tracef("Sending IPv6 packet to %s, len=%d", destIP.String(), len(packet))
	addr := syscall.SockaddrInet6{}
	copy(addr.Addr[:], destIP.To16())
	return syscall.Sendto(s.fd6, packet, 0, &addr)
}

func (s *Sender) Close() {
	if s.fd4 != 0 {
		_ = syscall.Close(s.fd4)
	}
	if s.fd6 != 0 {
		_ = syscall.Close(s.fd6)
	}
}
