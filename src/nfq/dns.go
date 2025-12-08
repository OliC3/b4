package nfq

import (
	"net"

	"github.com/daniellavrushin/b4/dns"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sock"
	"github.com/florianl/go-nfqueue"
)

func (w *Worker) processDnsPacket(ipVersion byte, sport uint16, dport uint16, payload []byte, raw []byte, ihl int, id uint32) int {

	if dport == 53 {
		domain, ok := dns.ParseQueryDomain(payload)
		if ok {
			matcher := w.getMatcher()
			if matchedSet, set := matcher.MatchSNI(domain); matchedSet && set.DNS.Enabled && set.DNS.TargetDNS != "" {

				targetIP := net.ParseIP(set.DNS.TargetDNS)
				if targetIP == nil {
					_ = w.q.SetVerdict(id, nfqueue.NfAccept)
					return 0
				}

				if ipVersion == IPv4 {
					targetDNS := targetIP.To4()
					if targetDNS == nil {
						// Target is IPv6 but packet is IPv4 - can't redirect
						_ = w.q.SetVerdict(id, nfqueue.NfAccept)
						return 0
					}

					originalDst := make(net.IP, 4)
					copy(originalDst, raw[16:20])

					dns.DnsNATSet(net.IP(raw[12:16]), sport, originalDst)

					copy(raw[16:20], targetDNS)
					sock.FixIPv4Checksum(raw[:ihl])
					sock.FixUDPChecksum(raw, ihl)
					_ = w.sock.SendIPv4(raw, targetDNS)
					_ = w.q.SetVerdict(id, nfqueue.NfDrop)
					log.Infof("DNS redirect: %s -> %s (set: %s)", domain, set.DNS.TargetDNS, set.Name)
					return 0

				} else { // IPv6
					cfg := w.getConfig()
					if !cfg.Queue.IPv6Enabled {
						_ = w.q.SetVerdict(id, nfqueue.NfAccept)
						return 0
					}

					targetDNS := targetIP.To16()
					if targetDNS == nil {
						_ = w.q.SetVerdict(id, nfqueue.NfAccept)
						return 0
					}

					// For IPv6: src at [8:24], dst at [24:40]
					originalDst := make(net.IP, 16)
					copy(originalDst, raw[24:40])

					dns.DnsNATSet(net.IP(raw[8:24]), sport, originalDst)

					copy(raw[24:40], targetDNS)
					sock.FixUDPChecksumV6(raw)
					_ = w.sock.SendIPv6(raw, targetDNS)
					_ = w.q.SetVerdict(id, nfqueue.NfDrop)
					log.Infof("DNS redirect (IPv6): %s -> %s (set: %s)", domain, set.DNS.TargetDNS, set.Name)
					return 0
				}
			}
		}
	}

	if sport == 53 {
		if ipVersion == IPv4 {
			if originalDst, ok := dns.DnsNATGet(net.IP(raw[16:20]), dport); ok {
				copy(raw[12:16], originalDst.To4())
				sock.FixIPv4Checksum(raw[:ihl])
				sock.FixUDPChecksum(raw, ihl)
				dns.DnsNATDelete(net.IP(raw[16:20]), dport)
				_ = w.sock.SendIPv4(raw, net.IP(raw[16:20]))
				_ = w.q.SetVerdict(id, nfqueue.NfDrop)
				return 0
			}
		} else { // IPv6
			cfg := w.getConfig()
			if cfg.Queue.IPv6Enabled {
				if originalDst, ok := dns.DnsNATGet(net.IP(raw[24:40]), dport); ok {
					copy(raw[8:24], originalDst.To16())
					sock.FixUDPChecksumV6(raw)
					dns.DnsNATDelete(net.IP(raw[24:40]), dport)
					_ = w.sock.SendIPv6(raw, net.IP(raw[24:40]))
					_ = w.q.SetVerdict(id, nfqueue.NfDrop)
					return 0
				}
			}
		}
	}

	_ = w.q.SetVerdict(id, nfqueue.NfAccept)
	return 0
}
