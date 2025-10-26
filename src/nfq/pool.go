package nfq

import (
	"context"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/log"
	"github.com/daniellavrushin/b4/sni"
)

func NewWorkerWithQueue(cfg *config.Config, domains *[]string, qnum uint16) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	w := &Worker{
		qnum:   qnum,
		ctx:    ctx,
		cancel: cancel,
		flows:  make(map[string]*flowState),
		ttl:    5 * time.Second,
		limit:  2048,
	}

	w.cfg.Store(cfg)
	w.rebuildMatcher(*domains)

	return w
}

func NewPool(cfg *config.Config, domains *[]string) *Pool {
	threads := cfg.Threads
	start := uint16(cfg.QueueStartNum)
	if threads < 1 {
		threads = 1
	}
	ws := make([]*Worker, 0, threads)
	for i := 0; i < threads; i++ {
		ws = append(ws, NewWorkerWithQueue(cfg, domains, start+uint16(i)))
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
		worker := w // capture loop variable
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

func (w *Worker) UpdateConfig(newCfg *config.Config, allDomains []string) {
	w.cfg.Store(newCfg)
	w.rebuildMatcher(allDomains)
}

func (w *Worker) rebuildMatcher(domains []string) {
	var m *sni.SuffixSet
	if len(domains) > 0 {
		m = sni.NewSuffixSet(domains)
		log.Tracef("Built matcher with %d domains", len(domains))
	} else {
		m = sni.NewSuffixSet([]string{})
		log.Tracef("Built empty matcher")
	}
	w.matcher.Store(m)
}

func (p *Pool) UpdateConfig(newCfg *config.Config, allDomains []string) {
	for _, w := range p.workers {
		w.UpdateConfig(newCfg, allDomains)
	}
	log.Tracef("Updated all %d workers with %d domains for matching", len(p.workers), len(allDomains))
}
