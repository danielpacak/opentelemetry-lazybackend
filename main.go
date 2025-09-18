package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

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
	fmt.Printf("Starting OpenTelemetry Profiles Backend: %v\n", os.Getpid())
	port := 4137
	lis, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)
	pprofileotlp.RegisterGRPCServer(s, &fakeProfilesServer{})

	err = s.Serve(lis)
	//s.GracefulStop()
	return err
}

type fakeProfilesServer struct {
	pprofileotlp.UnimplementedGRPCServer
}

func (f fakeProfilesServer) Export(_ context.Context, request pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	fmt.Printf("%v: Yoooo!! (%d)\n", time.Now(), request.Profiles().SampleCount())
	return pprofileotlp.NewExportResponse(), nil
}
