package receiver

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

type Receiver interface {
	ReceiveProfiles(ctx context.Context, pd pprofile.Profiles) error
	ReceiveLogs(ctx context.Context, ld plog.Logs) error
}

type Chain struct {
	receivers []Receiver
}

func NewChain(receivers ...Receiver) *Chain {
	return &Chain{
		receivers: receivers,
	}
}

func (r *Chain) ReceiveProfiles(ctx context.Context, pd pprofile.Profiles) error {
	for _, receiver := range r.receivers {
		_ = receiver.ReceiveProfiles(ctx, pd)
	}
	return nil
}

func (r *Chain) ReceiveLogs(ctx context.Context, ld plog.Logs) error {
	for _, receiver := range r.receivers {
		_ = receiver.ReceiveLogs(ctx, ld)
	}
	return nil
}
