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
	pprofileotlp.RegisterGRPCServer(s, &profilesServer{})

	err = s.Serve(lis)
	//s.GracefulStop()
	return err
}

type profilesServer struct {
	pprofileotlp.UnimplementedGRPCServer
}

func (f profilesServer) Export(_ context.Context, request pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	slog.Info("receiving profiles", "count", request.Profiles().SampleCount())
	return pprofileotlp.NewExportResponse(), nil
}
