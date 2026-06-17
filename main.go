package main

import (
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	// Import the gzip package which auto-registers the gzip gRPC compressor.
	_ "google.golang.org/grpc/encoding/gzip"

	"github.com/danielpacak/opentelemetry-lazybackend/collector/receiver/otlpreceiver/errors"
	"github.com/danielpacak/opentelemetry-lazybackend/collector/statusutil"
	"github.com/danielpacak/opentelemetry-lazybackend/receiver"
	"github.com/danielpacak/opentelemetry-lazybackend/receiver/filesystem"
	"github.com/danielpacak/opentelemetry-lazybackend/receiver/prometheus"
	"github.com/danielpacak/opentelemetry-lazybackend/receiver/stdout"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		os.Exit(1)
	}
}

func run() error {
	address := flag.String("address", fmt.Sprintf("127.0.0.1:%d", 4317), "listen address (host:port)")
	metrics := flag.String("metrics", fmt.Sprintf("127.0.0.1:%d", 2112), "metrics address (host:port)")
	receiverName := flag.String("receiver", "stdout", "profiles receiver to use (stdout, prometheus, filesystem)")
	// Receiver-specific options are namespaced as "<receiver>.<option>".
	filesystemDir := flag.String("filesystem.dir", "profiles", "output directory for the filesystem receiver")
	filesystemContainerID := flag.String("filesystem.container-id", "", "if set, the filesystem receiver only processes profiles with this container.id")
	flag.Parse()

	slog.Info("Starting GRPC server",
		"endpoint", *address, "pid", os.Getpid(),
		"uid", os.Getuid(), "gid", os.Getgid())
	lis, err := net.Listen("tcp", *address)
	if err != nil {
		return err
	}

	var profilesReceiver receiver.Receiver
	switch *receiverName {
	case "stdout":
		profilesReceiver = stdout.NewReceiver(stdout.DefaultConfig())
	case "prometheus":
		profilesReceiver = prometheus.NewReceiver()
	case "filesystem":
		config := filesystem.DefaultConfig()
		config.Dir = *filesystemDir
		config.ContainerID = *filesystemContainerID
		profilesReceiver = filesystem.NewReceiver(config)
	default:
		return fmt.Errorf("unknown receiver %q (supported: stdout, prometheus, filesystem)", *receiverName)
	}
	slog.Info("Using profiles receiver", "receiver", *receiverName)

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)
	pprofileotlp.RegisterGRPCServer(s, newProfilesServer(profilesReceiver))
	pmetricotlp.RegisterGRPCServer(s, newMetricsServer())
	plogotlp.RegisterGRPCServer(s, newLogsServer())
	ptraceotlp.RegisterGRPCServer(s, newTracesServer())

	go func() {
		err = s.Serve(lis)
		//s.GracefulStop()
	}()

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/v1/logs", func(resp http.ResponseWriter, req *http.Request) {
		//handleLogs(resp, req, httpLogsReceiver)
		handleLogs(resp, req)
	})
	httpMux.HandleFunc("/v1/metrics", func(resp http.ResponseWriter, req *http.Request) {
		//handleMetrics(resp, req, httpMetricsReceiver)
		handleMetrics(resp, req)
	})
	httpMux.HandleFunc("/v1development/profiles", func(resp http.ResponseWriter, req *http.Request) {
		//handleProfiles(resp, req, httpProfilesReceiver)
		handleProfiles(resp, req)
	})

	handler := httpContentDecompressor(
		httpMux,
		1000000,                        // config
		nil,                            //serverOpts.ErrHandler,
		defaultCompressionAlgorithms(), //sc.CompressionAlgorithms,
		nil,                            //serverOpts.Decoders,
	)

	serverHTTP := &http.Server{
		Handler: handler, // todo add middleware
		//ReadTimeout:       sc.ReadTimeout,
		//ReadHeaderTimeout: sc.ReadHeaderTimeout,
		//WriteTimeout:      sc.WriteTimeout,
		//IdleTimeout:       sc.IdleTimeout,
		ErrorLog: slog.NewLogLogger(slog.NewTextHandler(os.Stderr, nil), slog.LevelError),
	}
	listener, err := net.Listen("tcp", "127.0.0.1:4318")
	if err != nil {
		return err
	}
	slog.Info("Starting HTTP server", slog.String("endpoint", "127.0.0.1:4318"))
	go func() {
		serverHTTP.Serve(listener)
	}()

	// meeeeetrics
	slog.Info("Starting metrics server", "endpoint", *metrics, "pattern", "/metrics")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(*metrics, nil)

	return err
}

type profilesServer struct {
	pprofileotlp.UnimplementedGRPCServer
	receiver receiver.Receiver
}

func newProfilesServer(receiver receiver.Receiver) *profilesServer {
	return &profilesServer{
		receiver: receiver,
	}
}

func (f *profilesServer) Export(ctx context.Context, request pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	slog.Debug("GRPC Handling profiles export request...")
	err := f.receiver.Receive(ctx, request.Profiles())
	if err != nil {
		slog.Error("failed to receive profiles")
	}
	return pprofileotlp.NewExportResponse(), nil
}

type metricsServer struct {
	pmetricotlp.UnimplementedGRPCServer
	receiver receiver.Receiver
}

func newMetricsServer() *metricsServer {
	return &metricsServer{}
}

func (m *metricsServer) Export(ctx context.Context, request pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	slog.Info("GRPC Handling metrics export request...")
	return pmetricotlp.NewExportResponse(), nil
}

type logsServer struct {
	plogotlp.UnimplementedGRPCServer
}

func newLogsServer() *logsServer {
	return &logsServer{}
}

func (l *logsServer) Export(ctx context.Context, request plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	slog.Info("GRPC Handling logs export request...")

	logRecordSlice := request.Logs().ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logRecordSlice.Len(); i++ {
		args := []slog.Attr{
			slog.String("body", logRecordSlice.At(i).Body().AsString()),
		}

		logRecordSlice.At(i).Attributes().Range(func(k string, v pcommon.Value) bool {
			args = append(args, slog.String(k, v.AsString()))
			return true
		})

		slog.LogAttrs(ctx, slog.LevelInfo, "Exporting log record", args...)
	}
	return plogotlp.NewExportResponse(), nil
}

type tracesServer struct {
	ptraceotlp.UnimplementedGRPCServer
}

func newTracesServer() *tracesServer {
	return &tracesServer{}
}

func (t *tracesServer) Export(ctx context.Context, request ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	slog.Info("GRPC Handling traces export request...")
	return ptraceotlp.NewExportResponse(), nil
}

// http protocol
// Pre-computed status with code=Internal to be used in case of a marshaling error.
var fallbackMsg = []byte(`{"code": 13, "message": "failed to marshal error message"}`)

const fallbackContentType = "application/json"

const (
	pbContentType   = "application/x-protobuf"
	jsonContentType = "application/json"
)

var (
	pbEncoder = &protoEncoder{}
	jsEncoder = &jsonEncoder{}
)

type protoEncoder struct{}

func (protoEncoder) unmarshalTracesRequest(buf []byte) (ptraceotlp.ExportRequest, error) {
	req := ptraceotlp.NewExportRequest()
	err := req.UnmarshalProto(buf)
	return req, err
}

func (protoEncoder) unmarshalMetricsRequest(buf []byte) (pmetricotlp.ExportRequest, error) {
	req := pmetricotlp.NewExportRequest()
	err := req.UnmarshalProto(buf)
	return req, err
}

func (protoEncoder) unmarshalLogsRequest(buf []byte) (plogotlp.ExportRequest, error) {
	req := plogotlp.NewExportRequest()
	err := req.UnmarshalProto(buf)
	return req, err
}

func (protoEncoder) unmarshalProfilesRequest(buf []byte) (pprofileotlp.ExportRequest, error) {
	req := pprofileotlp.NewExportRequest()
	err := req.UnmarshalProto(buf)
	return req, err
}

func (protoEncoder) marshalTracesResponse(resp ptraceotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalProto()
}

func (protoEncoder) marshalMetricsResponse(resp pmetricotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalProto()
}

func (protoEncoder) marshalLogsResponse(resp plogotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalProto()
}

func (protoEncoder) marshalProfilesResponse(resp pprofileotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalProto()
}

func (protoEncoder) marshalStatus(resp *spb.Status) ([]byte, error) {
	return proto.Marshal(resp)
}

func (protoEncoder) contentType() string {
	return pbContentType
}

type jsonEncoder struct{}

func (jsonEncoder) unmarshalTracesRequest(buf []byte) (ptraceotlp.ExportRequest, error) {
	req := ptraceotlp.NewExportRequest()
	err := req.UnmarshalJSON(buf)
	return req, err
}

func (jsonEncoder) unmarshalMetricsRequest(buf []byte) (pmetricotlp.ExportRequest, error) {
	req := pmetricotlp.NewExportRequest()
	err := req.UnmarshalJSON(buf)
	return req, err
}

func (jsonEncoder) unmarshalLogsRequest(buf []byte) (plogotlp.ExportRequest, error) {
	req := plogotlp.NewExportRequest()
	err := req.UnmarshalJSON(buf)
	return req, err
}

func (jsonEncoder) unmarshalProfilesRequest(buf []byte) (pprofileotlp.ExportRequest, error) {
	req := pprofileotlp.NewExportRequest()
	err := req.UnmarshalJSON(buf)
	return req, err
}

func (jsonEncoder) marshalTracesResponse(resp ptraceotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalJSON()
}

func (jsonEncoder) marshalMetricsResponse(resp pmetricotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalJSON()
}

func (jsonEncoder) marshalLogsResponse(resp plogotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalJSON()
}

func (jsonEncoder) marshalProfilesResponse(resp pprofileotlp.ExportResponse) ([]byte, error) {
	return resp.MarshalJSON()
}

func (jsonEncoder) marshalStatus(resp *spb.Status) ([]byte, error) {
	return protojson.Marshal(resp)
}

func (jsonEncoder) contentType() string {
	return jsonContentType
}

func handleMetrics(resp http.ResponseWriter, req *http.Request) {
	enc, ok := readContentType(resp, req)
	if !ok {
		return
	}

	body, ok := readAndCloseBody(resp, req, enc)
	if !ok {
		return
	}

	// todo we need request
	//otlpReq, err := enc.unmarshalMetricsRequest(body)
	_, err := enc.unmarshalMetricsRequest(body)
	if err != nil {
		writeError(resp, enc, err, http.StatusBadRequest)
		return
	}
	slog.Info("Handling HTTP metrics >>>>")

	// todo do sth with this data
	//otlpResp, err := metricsReceiver.Export(req.Context(), otlpReq)
	//if err != nil {
	//	writeError(resp, enc, err, http.StatusInternalServerError)
	//	return
	//}
	otlpResp := pmetricotlp.NewExportResponse()

	msg, err := enc.marshalMetricsResponse(otlpResp)
	if err != nil {
		writeError(resp, enc, err, http.StatusInternalServerError)
		return
	}
	writeResponse(resp, enc.contentType(), http.StatusOK, msg)
}

func handleLogs(resp http.ResponseWriter, req *http.Request) {
	enc, ok := readContentType(resp, req)
	if !ok {
		return
	}

	body, ok := readAndCloseBody(resp, req, enc)
	if !ok {
		return
	}

	// TODO we need it
	otlpReq, err := enc.unmarshalLogsRequest(body)
	if err != nil {
		writeError(resp, enc, err, http.StatusBadRequest)
		return
	}

	logRecordSlice := otlpReq.Logs().ResourceLogs().At(0).ScopeLogs().At(0).LogRecords()
	for i := 0; i < logRecordSlice.Len(); i++ {
		args := []slog.Attr{
			slog.String("body", logRecordSlice.At(i).Body().AsString()),
		}

		logRecordSlice.At(i).Attributes().Range(func(k string, v pcommon.Value) bool {
			args = append(args, slog.String(k, v.AsString()))
			return true
		})

		slog.LogAttrs(req.Context(), slog.LevelInfo, "Exporting log record", args...)
	}

	//otlpResp, err := logsReceiver.Export(req.Context(), otlpReq)
	//if err != nil {
	//	writeError(resp, enc, err, http.StatusInternalServerError)
	//	return
	//}
	// TODO ACTUALLY HANDLE IT
	otlpResp := plogotlp.NewExportResponse()

	msg, err := enc.marshalLogsResponse(otlpResp)
	if err != nil {
		writeError(resp, enc, err, http.StatusInternalServerError)
		return
	}
	writeResponse(resp, enc.contentType(), http.StatusOK, msg)
}

func handleProfiles(resp http.ResponseWriter, req *http.Request) {
	enc, ok := readContentType(resp, req)
	if !ok {
		return
	}

	body, ok := readAndCloseBody(resp, req, enc)
	if !ok {
		return
	}

	// TODO Actually process this data
	//otlpReq, err := enc.unmarshalProfilesRequest(body)
	_, err := enc.unmarshalProfilesRequest(body)
	if err != nil {
		writeError(resp, enc, err, http.StatusBadRequest)
		return
	}

	//otlpResp, err := profilesReceiver.Export(req.Context(), otlpReq)
	//if err != nil {
	//	writeError(resp, enc, err, http.StatusInternalServerError)
	//	return
	//}
	otlpResp := pprofileotlp.NewExportResponse()

	msg, err := enc.marshalProfilesResponse(otlpResp)
	if err != nil {
		writeError(resp, enc, err, http.StatusInternalServerError)
		return
	}
	writeResponse(resp, enc.contentType(), http.StatusOK, msg)
}

func readContentType(resp http.ResponseWriter, req *http.Request) (encoder, bool) {
	if req.Method != http.MethodPost {
		handleUnmatchedMethod(resp)
		return nil, false
	}

	switch getMimeTypeFromContentType(req.Header.Get("Content-Type")) {
	case pbContentType:
		return pbEncoder, true
	case jsonContentType:
		return jsEncoder, true
	default:
		handleUnmatchedContentType(resp)
		return nil, false
	}
}

func readAndCloseBody(resp http.ResponseWriter, req *http.Request, enc encoder) ([]byte, bool) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		writeError(resp, enc, err, http.StatusBadRequest)
		return nil, false
	}
	if err = req.Body.Close(); err != nil {
		writeError(resp, enc, err, http.StatusBadRequest)
		return nil, false
	}
	return body, true
}

// writeError encodes the HTTP error inside a rpc.Status message as required by the OTLP protocol.
func writeError(w http.ResponseWriter, encoder encoder, err error, statusCode int) {
	s, ok := status.FromError(err)
	if ok {
		statusCode = errors.GetHTTPStatusCodeFromStatus(s)
	} else {
		s = statusutil.NewStatusFromMsgAndHTTPCode(err.Error(), statusCode)
	}
	writeStatusResponse(w, encoder, statusCode, s)
}

func writeStatusResponse(w http.ResponseWriter, enc encoder, statusCode int, st *status.Status) {
	// https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#otlphttp-throttling
	if statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable {
		retryInfo := statusutil.GetRetryInfo(st)
		// Check if server returned throttling information.
		if retryInfo != nil {
			// We are throttled. Wait before retrying as requested by the server.
			// The value of Retry-After field can be either an HTTP-date or a number of
			// seconds to delay after the response is received. See https://datatracker.ietf.org/doc/html/rfc7231#section-7.1.3
			//
			// Retry-After = HTTP-date / delay-seconds
			//
			// Use delay-seconds since is easier to format as well as does not require clock synchronization.
			w.Header().Set("Retry-After", strconv.FormatInt(int64(retryInfo.GetRetryDelay().AsDuration()/time.Second), 10))
		}
	}
	msg, err := enc.marshalStatus(st.Proto())
	if err != nil {
		writeResponse(w, fallbackContentType, http.StatusInternalServerError, fallbackMsg)
		return
	}

	writeResponse(w, enc.contentType(), statusCode, msg)
}

func handleUnmatchedMethod(resp http.ResponseWriter) {
	hst := http.StatusMethodNotAllowed
	writeResponse(resp, "text/plain", hst, fmt.Appendf(nil, "%v method not allowed, supported: [POST]", hst))
}

func handleUnmatchedContentType(resp http.ResponseWriter) {
	hst := http.StatusUnsupportedMediaType
	writeResponse(resp, "text/plain", hst, fmt.Appendf(nil, "%v unsupported media type, supported: [%s, %s]", hst, jsonContentType, pbContentType))
}

func writeResponse(w http.ResponseWriter, contentType string, statusCode int, msg []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	// Nothing we can do with the error if we cannot write to the response.
	_, _ = w.Write(msg)
}

func getMimeTypeFromContentType(contentType string) string {
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return mediatype
}

type encoder interface {
	unmarshalTracesRequest(buf []byte) (ptraceotlp.ExportRequest, error)
	unmarshalMetricsRequest(buf []byte) (pmetricotlp.ExportRequest, error)
	unmarshalLogsRequest(buf []byte) (plogotlp.ExportRequest, error)
	unmarshalProfilesRequest(buf []byte) (pprofileotlp.ExportRequest, error)

	marshalTracesResponse(ptraceotlp.ExportResponse) ([]byte, error)
	marshalMetricsResponse(pmetricotlp.ExportResponse) ([]byte, error)
	marshalLogsResponse(plogotlp.ExportResponse) ([]byte, error)
	marshalProfilesResponse(pprofileotlp.ExportResponse) ([]byte, error)

	marshalStatus(rsp *spb.Status) ([]byte, error)

	contentType() string
}

// compression

// httpContentDecompressor offloads the task of handling compressed HTTP requests
// by identifying the compression format in the "Content-Encoding" header and re-writing
// request body so that the handlers further in the chain can work on decompressed data.
func httpContentDecompressor(h http.Handler, maxRequestBodySize int64, eh func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int), enableDecoders []string, decoders map[string]func(body io.ReadCloser) (io.ReadCloser, error)) http.Handler {
	errHandler := defaultErrorHandler
	if eh != nil {
		errHandler = eh
	}

	enabled := map[string]func(body io.ReadCloser) (io.ReadCloser, error){}
	for _, dec := range enableDecoders {
		enabled[dec] = availableDecoders[dec]
	}

	d := &decompressor{
		maxRequestBodySize: maxRequestBodySize,
		errHandler:         errHandler,
		base:               h,
		decoders:           enabled,
	}

	maps.Copy(d.decoders, decoders)

	return d
}

// defaultErrorHandler writes the error message in plain text.
func defaultErrorHandler(w http.ResponseWriter, _ *http.Request, errMsg string, statusCode int) {
	http.Error(w, errMsg, statusCode)
}

var availableDecoders = map[string]func(body io.ReadCloser) (io.ReadCloser, error){
	"": func(io.ReadCloser) (io.ReadCloser, error) {
		// Not a compressed payload. Nothing to do.
		return nil, nil
	},
	"gzip": func(body io.ReadCloser) (io.ReadCloser, error) {
		gr, err := gzip.NewReader(body)
		if err != nil {
			return nil, err
		}
		return gr, nil
	},
}

type decompressor struct {
	errHandler         func(w http.ResponseWriter, r *http.Request, errorMsg string, statusCode int)
	base               http.Handler
	decoders           map[string]func(body io.ReadCloser) (io.ReadCloser, error)
	maxRequestBodySize int64
}

func (d *decompressor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	newBody, err := d.newBodyReader(r)
	if err != nil {
		d.errHandler(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	if newBody != nil {
		defer newBody.Close()
		// "Content-Encoding" header is removed to avoid decompressing twice
		// in case the next handler(s) have implemented a similar mechanism.
		r.Header.Del("Content-Encoding")
		// "Content-Length" is set to -1 as the size of the decompressed body is unknown.
		r.Header.Del("Content-Length")
		r.ContentLength = -1
		r.Body = http.MaxBytesReader(w, newBody, d.maxRequestBodySize)
	}
	d.base.ServeHTTP(w, r)
}

const (
	headerContentEncoding = "Content-Encoding"
)

func (d *decompressor) newBodyReader(r *http.Request) (io.ReadCloser, error) {
	encoding := r.Header.Get(headerContentEncoding)
	decoder, ok := d.decoders[encoding]
	if !ok {
		return nil, fmt.Errorf("unsupported %s: %s", headerContentEncoding, encoding)
	}
	return decoder(r.Body)
}

func defaultCompressionAlgorithms() []string {
	//if enableFramedSnappy.IsEnabled() {
	//	return []string{"", "gzip", "zstd", "zlib", "snappy", "deflate", "lz4", "x-snappy-framed"}
	//}
	//return []string{"", "gzip", "zstd", "zlib", "snappy", "deflate", "lz4"}
	return []string{"", "gzip"}
}
