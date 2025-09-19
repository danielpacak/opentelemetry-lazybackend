package receiver

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pprofile"
)

type Receiver interface {
	Receive(ctx context.Context, pd pprofile.Profiles) error
}
