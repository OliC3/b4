package nfq

import (
	"context"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sni"
	"github.com/daniellavrushin/b4/sock"
	"github.com/florianl/go-nfqueue"
)

type Pool struct {
	workers  []*Worker
	packetCh chan nfqueue.Attribute
	nfq      *nfqueue.Nfqueue
}

func NewWorkerWithQueue(cfg *config.Config, qnum uint16) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	var m *sni.SuffixSet
	if len(cfg.SNIDomains) > 0 {
		m = sni.NewSuffixSet(cfg.SNIDomains)
	}

	var strategy sock.FakeStrategy
	switch cfg.FakeStrategy {
	case "ttl":
		strategy = sock.FakeStrategyTTL
	case "randseq":
		strategy = sock.FakeStrategyRandSeq
	case "pastseq":
		strategy = sock.FakeStrategyPastSeq
	case "tcp_check":
		strategy = sock.FakeStrategyTCPChecksum
	default:
		strategy = sock.FakeStrategyPastSeq
	}

	fragmenter := &sock.Fragmenter{
		SplitPosition: cfg.FragSNIPosition,
		ReverseOrder:  cfg.FragSNIReverse,
		FakeSNI:       cfg.FakeSNI,
		MiddleSplit:   cfg.FragMiddleSNI,
		FakeStrategy:  strategy,
	}

	return &Worker{
		cfg:     cfg,
		qnum:    qnum,
		ctx:     ctx,
		cancel:  cancel,
		flows:   make(map[string]*flowState),
		ttl:     10 * time.Second,
		limit:   8192,
		matcher: m,
		frag:    fragmenter,
	}
}

func NewPool(start uint16, threads int, cfg *config.Config) *Pool {
	if threads < 1 {
		threads = 1
	}
	ws := make([]*Worker, 0, threads)
	for i := 0; i < threads; i++ {
		ws = append(ws, NewWorkerWithQueue(cfg, start+uint16(i)))
	}
	return &Pool{workers: ws}
}

func (p *Pool) Start() error {
	for _, w := range p.workers {
		if err := w.Start(p.packetCh); err != nil {
			for _, x := range p.workers {
				x.Stop()
			}
			return err
		}
	}
	return nil
}

func (p *Pool) Stop() {
	for _, w := range p.workers {
		w.Stop()
	}
}
