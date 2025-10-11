package sock

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

const PacketMark = 0x8000 // 32768 - prevents NFQUEUE reprocessing

type Sender struct {
	fd4 int // IPv4 raw socket
	fd6 int // IPv6 raw socket
}

func NewSender() (*Sender, error) {
	// Create IPv4 raw socket
	fd4, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		return nil, err
	}

	// Enable IP_HDRINCL to send custom IP headers
	if err := syscall.SetsockoptInt(fd4, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
		syscall.Close(fd4)
		return nil, err
	}

	// Set SO_MARK to prevent NFQUEUE loop (critical!)
	if err := syscall.SetsockoptInt(fd4, syscall.SOL_SOCKET, unix.SO_MARK, PacketMark); err != nil {
		syscall.Close(fd4)
		return nil, err
	}

	// Same for IPv6
	fd6, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_RAW, syscall.IPPROTO_RAW)
	if err != nil {
		syscall.Close(fd4)
		return nil, err
	}

	syscall.SetsockoptInt(fd6, syscall.SOL_SOCKET, unix.SO_MARK, PacketMark)

	return &Sender{fd4: fd4, fd6: fd6}, nil
}

func (s *Sender) SendIPv4(packet []byte, destIP net.IP) error {
	addr := syscall.SockaddrInet4{}
	copy(addr.Addr[:], destIP.To4())
	return syscall.Sendto(s.fd4, packet, 0, &addr)
}

func (s *Sender) SendIPv6(packet []byte, destIP net.IP) error {
	addr := syscall.SockaddrInet6{}
	copy(addr.Addr[:], destIP.To16())
	return syscall.Sendto(s.fd6, packet, 0, &addr)
}
