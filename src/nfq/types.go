package nfq

import (
	"context"
	"sync"
	"time"

	"github.com/daniellavrushin/b4/config"
	"github.com/daniellavrushin/b4/sni"
	"github.com/daniellavrushin/b4/sock"
	"github.com/florianl/go-nfqueue"
)

type Pool struct {
	workers []*Worker
}

type flowState struct {
	buf      []byte
	last     time.Time
	sniFound bool
	sni      string
}

type Worker struct {
	cfg              *config.Config
	qnum             uint16
	ctx              context.Context
	cancel           context.CancelFunc
	q                *nfqueue.Nfqueue
	wg               sync.WaitGroup
	mu               sync.Mutex
	flows            map[string]*flowState
	ttl              time.Duration
	limit            int
	matcher          *sni.SuffixSet
	sock             *sock.Sender
	frag             *sock.Fragmenter
	packetsProcessed uint64
}
