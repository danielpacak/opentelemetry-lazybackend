package receiver

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

type Receiver interface {
	Receive(ctx context.Context, pd pprofile.Profiles) error
}

type Chain struct {
	receivers []Receiver
}

func NewChain(receivers ...Receiver) *Chain {
	return &Chain{
		receivers: receivers,
	}
}

func (r *Chain) Receive(ctx context.Context, pd pprofile.Profiles) error {
	for _, receiver := range r.receivers {
		_ = receiver.Receive(ctx, pd)
	}
	return nil
}
