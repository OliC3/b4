package nfq

import (
	"context"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sni"
	"github.com/daniellavrushin/b4/sock"
)

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
		if err := w.Start(); err != nil {
			for _, x := range p.workers {
				x.Stop()
			}
			return err
		}
	}
	return nil
}

func (p *Pool) Stop() {
	// Use goroutines to stop workers in parallel for faster shutdown
	var wg sync.WaitGroup
	for _, w := range p.workers {
		wg.Add(1)
		worker := w // capture loop variable
		go func() {
			defer wg.Done()
			worker.Stop()
		}()
	}

	// Wait for all workers to stop with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers stopped successfully
		log.Infof("All NFQueue workers stopped")
	case <-time.After(3 * time.Second):
		// Timeout - some workers didn't stop cleanly
		log.Errorf("Timeout waiting for NFQueue workers to stop")
	}
}
