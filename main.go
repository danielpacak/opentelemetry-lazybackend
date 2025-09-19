package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"google.golang.org/grpc"

	// Import the gzip package which auto-registers the gzip gRPC compressor.
	_ "google.golang.org/grpc/encoding/gzip"

	"github.com/danielpacak/opentelemetry-profiles-lazybackend/receiver"
	"github.com/danielpacak/opentelemetry-profiles-lazybackend/receiver/stdout"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
}

func run() error {
	slog.Info("starting profiles lazy backend server",
		"pid", os.Getpid(), "uid", os.Getuid(), "gid", os.Getgid())
	port := 4137
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)
	pprofileotlp.RegisterGRPCServer(s, newProfilesServer(stdout.NewReceiver(stdout.DefaultConfig())))

	err = s.Serve(lis)
	//s.GracefulStop()
	return err
}

func newProfilesServer(receiver receiver.Receiver) *profilesServer {
	return &profilesServer{
		receiver: receiver,
	}
}

type profilesServer struct {
	pprofileotlp.UnimplementedGRPCServer
	receiver receiver.Receiver
}

func (f *profilesServer) Export(ctx context.Context, request pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	err := f.receiver.Receive(ctx, request.Profiles())
	if err != nil {
		slog.Error("failed to receive profiles")
	}
	return pprofileotlp.NewExportResponse(), nil
}
