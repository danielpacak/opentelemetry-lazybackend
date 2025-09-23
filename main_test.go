package main

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/danielpacak/opentelemetry-lazybackend/libpf"
)

type Config struct {
	// Name defines the name of the agent.
	Name string

	// Version defines the version of the agent.
	Version string

	// CollAgentAddr defines the destination of the backend connection.
	CollAgentAddr string

	// MaxRPCMsgSize defines the maximum size of a gRPC message.
	MaxRPCMsgSize int

	// Disable secure communication with Collection Agent.
	DisableTLS bool
	// ExecutablesCacheElements defines item capacity of the executables cache.
	ExecutablesCacheElements uint32
	// samplesPerSecond defines the number of samples per second.
	SamplesPerSecond int

	// Number of connection attempts to the collector after which we give up retrying.
	MaxGRPCRetries uint32

	GRPCOperationTimeout   time.Duration
	GRPCStartupBackoffTime time.Duration
	GRPCConnectionTimeout  time.Duration
	ReportInterval         time.Duration

	// gRPCInterceptor is the client gRPC interceptor, e.g., for sending gRPC metadata.
	GRPCClientInterceptor grpc.UnaryClientInterceptor

	// ExtraSampleAttrProd is an optional hook point for adding custom
	// attributes to samples.
	//ExtraSampleAttrProd samples.SampleAttrProducer

	// GRPCDialOptions allows passing additional gRPC dial options when establishing
	// the connection to the collector. These options are appended after the default options.
	GRPCDialOptions []grpc.DialOption
}

func TestMyProfiler(t *testing.T) {
	cfg := &Config{
		CollAgentAddr:          "localhost:4137",
		DisableTLS:             true,
		GRPCStartupBackoffTime: 1 * time.Minute,
		MaxGRPCRetries:         5,
		GRPCConnectionTimeout:  3 * time.Second,
		MaxRPCMsgSize:          32 << 20, // 32 MiB
	}

	var client pprofileotlp.GRPCClient

	otlpGrpcConn, err := waitGrpcEndpoint(t.Context(), cfg)
	require.NoError(t, err)

	client = pprofileotlp.NewGRPCClient(otlpGrpcConn)

	_, err = client.Export(context.Background(), generateProfilesRequest())
	require.NoError(t, err)
	t.Logf("xxxs")
	// st, okSt := status.FromError(err)
	// require.True(t, okSt)
	// assert.Equal(t, "my error", st.Message())
	// assert.Equal(t, codes.Unknown, st.Code())
	// assert.Equal(t, pprofileotlp.ExportResponse{}, resp)
}

// waitGrpcEndpoint waits until the gRPC connection is established.
func waitGrpcEndpoint(ctx context.Context, cfg *Config) (*grpc.ClientConn, error) {
	// Sleep with a fixed backoff time added of +/- 20% jitter
	tick := time.NewTicker(libpf.AddJitter(cfg.GRPCStartupBackoffTime, 0.2))
	defer tick.Stop()

	var retries uint32
	for {
		if collAgentConn, err := setupGrpcConnection(ctx, cfg); err != nil {
			if retries >= cfg.MaxGRPCRetries {
				return nil, err
			}
			retries++

			log.Warnf(
				"Failed to setup gRPC connection (try %d of %d): %v",
				retries,
				cfg.MaxGRPCRetries,
				err,
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-tick.C:
				continue
			}
		} else {
			return collAgentConn, nil
		}
	}
}

func setupGrpcConnection(parent context.Context, cfg *Config) (*grpc.ClientConn, error) {
	//nolint:staticcheck
	opts := []grpc.DialOption{grpc.WithBlock(),
		grpc.WithUnaryInterceptor(cfg.GRPCClientInterceptor),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.MaxRPCMsgSize),
			grpc.MaxCallSendMsgSize(cfg.MaxRPCMsgSize)),
		grpc.WithReturnConnectionError(),
	}

	if cfg.DisableTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				// Support only TLS1.3+ with valid CA certificates
				MinVersion:         tls.VersionTLS13,
				InsecureSkipVerify: false,
			})))
	}

	opts = append(opts, cfg.GRPCDialOptions...)

	ctx, cancel := context.WithTimeout(parent, cfg.GRPCConnectionTimeout)
	defer cancel()
	//nolint:staticcheck
	return grpc.DialContext(ctx, cfg.CollAgentAddr, opts...)
}

// This test is the basic setup as in tutorials
// https://grpc.io/docs/languages/go/basics/
func TestMe(t *testing.T) {
	cc, err := grpc.NewClient("localhost:4137",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	logClient := pprofileotlp.NewGRPCClient(cc)

	_, err = logClient.Export(context.Background(), generateProfilesRequest())
	require.NoError(t, err)
	// st, okSt := status.FromError(err)
	// require.True(t, okSt)
	// assert.Equal(t, "my error", st.Message())
	// assert.Equal(t, codes.Unknown, st.Code())
	// assert.Equal(t, pprofileotlp.ExportResponse{}, resp)
}

func generateProfilesRequest() pprofileotlp.ExportRequest {
	td := pprofile.NewProfiles()
	td.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	return pprofileotlp.NewExportRequestFromProfiles(td)
}
