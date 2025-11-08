package nfq

import (
	"context"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sni"
)

func NewWorkerWithQueue(cfg *config.Config, qnum uint16) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	w := &Worker{
		qnum:   qnum,
		ctx:    ctx,
		cancel: cancel,
	}

	w.cfg.Store(cfg)
	w.rebuildMatcher(cfg)

	return w
}

func NewPool(cfg *config.Config) *Pool {
	threads := cfg.Queue.Threads
	start := uint16(cfg.Queue.StartNum)
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
	var wg sync.WaitGroup
	for _, w := range p.workers {
		wg.Add(1)
		worker := w
		go func() {
			defer wg.Done()
			worker.Stop()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Infof("All NFQueue workers stopped")
	case <-time.After(3 * time.Second):
		log.Errorf("Timeout waiting for NFQueue workers to stop")
	}
}

func (w *Worker) getConfig() *config.Config {
	return w.cfg.Load().(*config.Config)
}

func (w *Worker) getMatcher() *sni.SuffixSet {
	return w.matcher.Load().(*sni.SuffixSet)
}

func (w *Worker) UpdateConfig(newCfg *config.Config) {
	w.cfg.Store(newCfg)
	w.rebuildMatcher(newCfg)
}

func (w *Worker) rebuildMatcher(cfg *config.Config) {
	var m *sni.SuffixSet
	if len(cfg.Sets) > 0 {
		m = sni.NewSuffixSet(cfg.Sets)
		totalDomains := 0
		for _, set := range cfg.Sets {
			totalDomains += len(set.Domains.SNIDomains)
		}
		log.Tracef("Built matcher with %d domains across %d sets", totalDomains, len(cfg.Sets))
	} else {
		m = sni.NewSuffixSet([]*config.SetConfig{})
		log.Tracef("Built empty matcher")
	}
	w.matcher.Store(m)
}

func (p *Pool) UpdateConfig(newCfg *config.Config) {
	for _, w := range p.workers {
		w.UpdateConfig(newCfg)
	}
	totalDomains := 0
	for _, set := range newCfg.Sets {
		totalDomains += len(set.Domains.SNIDomains)
	}
	log.Tracef("Updated all %d workers with %d domains", len(p.workers), totalDomains)
}
