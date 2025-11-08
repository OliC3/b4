package nfq

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/daniellavrushin/b4/sock"
	"github.com/florianl/go-nfqueue"
)

type Pool struct {
	workers []*Worker
}

type Worker struct {
	packetsProcessed uint64
	cfg              atomic.Value
	qnum             uint16
	ctx              context.Context
	cancel           context.CancelFunc
	q                *nfqueue.Nfqueue
	wg               sync.WaitGroup
	matcher          atomic.Value
	sock             *sock.Sender
}
