package receiver

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

type Receiver interface {
	Start(ctx context.Context) error
	Receive(ctx context.Context, pd pprofile.Profiles) error
	Stop(ctx context.Context) error
}

type Chain struct {
	receivers []Receiver
}

func NewChain(receivers ...Receiver) *Chain {
	return &Chain{
		receivers: receivers,
	}
}

func (r *Chain) Start(ctx context.Context) error {
	for _, recv := range r.receivers {
		if err := recv.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Chain) Receive(ctx context.Context, pd pprofile.Profiles) error {
	for _, recv := range r.receivers {
		_ = recv.Receive(ctx, pd)
	}
	return nil
}

func (r *Chain) Stop(ctx context.Context) error {
	for i := len(r.receivers) - 1; i >= 0; i-- {
		if err := r.receivers[i].Stop(ctx); err != nil {
			return err
		}
	}
	return nil
}
