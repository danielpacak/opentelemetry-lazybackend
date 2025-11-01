package main_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	logsdk "go.opentelemetry.io/otel/sdk/log"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

func TestThreatDetection(t *testing.T) {

	t.Run("Should export process exec events", func(t *testing.T) {
		grpcLogExporter, err := otlploggrpc.New(t.Context(),
			otlploggrpc.WithInsecure(),
			otlploggrpc.WithCompressor(""),
			otlploggrpc.WithEndpoint("localhost:4317"),
		)
		require.NoError(t, err)
		loggerProvider := logsdk.NewLoggerProvider(
			logsdk.WithProcessor(logsdk.NewBatchProcessor(grpcLogExporter,
				logsdk.WithMaxQueueSize(2048),
				logsdk.WithExportInterval(1*time.Second),
				logsdk.WithExportTimeout(30*time.Second),
				logsdk.WithExportMaxBatchSize(512),
				logsdk.WithExportBufferSize(1),
			)),
		)
		global.SetLoggerProvider(loggerProvider)

		logger := otelslog.NewLogger("xthis is my test app")

		// send some log records
		logger.Info("exporting dynamic-exec",
			"type", "dynamic_exec",
			"resourceName", "dynamic-exec",
			"resourceData", "_",
			"containerID", "ba348639d542059482a036379549878e5e2b0aeaa1351ed965515bdc9e343863",
			"pid", 3606,
			"ppid", 3579,
			"binary", "/usr/bin/calico-node",
		)

		err = loggerProvider.ForceFlush(t.Context())
		require.NoError(t, err)
		err = loggerProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

}

func TestLazyBackendExportersGRPC(t *testing.T) {

	t.Run("Should export logs with no compression", func(t *testing.T) {
		grpcLogExporter, err := otlploggrpc.New(t.Context(),
			otlploggrpc.WithInsecure(),
			otlploggrpc.WithCompressor(""),
			otlploggrpc.WithEndpoint("localhost:4317"),
		)
		require.NoError(t, err)
		loggerProvider := logsdk.NewLoggerProvider(
			logsdk.WithProcessor(logsdk.NewBatchProcessor(grpcLogExporter,
				logsdk.WithMaxQueueSize(2048),
				logsdk.WithExportInterval(1*time.Second),
				logsdk.WithExportTimeout(30*time.Second),
				logsdk.WithExportMaxBatchSize(512),
				logsdk.WithExportBufferSize(1),
			)),
		)
		global.SetLoggerProvider(loggerProvider)
		// TODO Fix this name to something more in real world
		logger := otelslog.NewLogger("this is my test app")

		// send some log records
		logger.Info("Log message 1 GRPC")
		logger.Info("Log message 2 GRPC")
		logger.Info("Log message 3 GRPC", "foo", "bar", "da", "niel")

		err = loggerProvider.ForceFlush(t.Context())
		require.NoError(t, err)
		err = loggerProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("Should export logs with gzip compression", func(t *testing.T) {
		grpcLogExporter, err := otlploggrpc.New(t.Context(),
			otlploggrpc.WithInsecure(),
			otlploggrpc.WithCompressor("gzip"),
			otlploggrpc.WithEndpoint("localhost:4317"),
		)
		require.NoError(t, err)
		loggerProvider := logsdk.NewLoggerProvider(
			logsdk.WithProcessor(logsdk.NewBatchProcessor(grpcLogExporter,
				logsdk.WithMaxQueueSize(2048),
				logsdk.WithExportInterval(1*time.Second),
				logsdk.WithExportTimeout(30*time.Second),
				logsdk.WithExportMaxBatchSize(512),
				logsdk.WithExportBufferSize(1),
			)),
		)
		global.SetLoggerProvider(loggerProvider)
		// TODO Fix this name to something more in real world
		logger := otelslog.NewLogger("this is my test app")

		// send some log records
		logger.Info("Log message 1 GRPC GZIP")
		logger.Info("Log message 2 GRPC GZIP")
		logger.Info("Log message 3 GRPC GZIP", "foo", "bar", "da", "niel")

		err = loggerProvider.ForceFlush(t.Context())
		require.NoError(t, err)
		err = loggerProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("Should export metrics", func(t *testing.T) {
		grpcMetricExporter, err := otlpmetricgrpc.New(t.Context(),
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithEndpoint("localhost:4317"),
		)
		require.NoError(t, err)

		meterProvider := metricsdk.NewMeterProvider(
			metricsdk.WithReader(metricsdk.NewPeriodicReader(grpcMetricExporter,
				// Default is 1m. Set to 3s for demonstrative purposes.
				metricsdk.WithInterval(3*time.Second))),
		)
		otel.SetMeterProvider(meterProvider)

		meter := otel.Meter("this is my test app")
		rollCnt, err := meter.Int64Counter("dice.rolls",
			metric.WithDescription("The number of rolls by roll value"),
			metric.WithUnit("{roll}"))
		require.NoError(t, err)
		rollValueAttr := attribute.Int("roll.value", 3)
		rollCnt.Add(t.Context(), 1, metric.WithAttributes(rollValueAttr))

		err = meterProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("Should export profiles", func(t *testing.T) {
		// TODO - > I can also test it with opentelemetry-ebpf-profiler
	})

	t.Run("Should export traces", func(t *testing.T) {
		// TODO test with Beyla or use Go SDK
	})

}

func TestLazyBackendExportersHTTP(t *testing.T) {

	t.Run("Should export logs with no compression", func(t *testing.T) {
		httpLogExporter, err := otlploghttp.New(t.Context(),
			otlploghttp.WithInsecure(),
			otlploghttp.WithCompression(otlploghttp.NoCompression),
			otlploghttp.WithEndpoint("localhost:4318"),
		)
		require.NoError(t, err)
		loggerProvider := logsdk.NewLoggerProvider(
			logsdk.WithProcessor(logsdk.NewBatchProcessor(httpLogExporter,
				logsdk.WithMaxQueueSize(2048),
				logsdk.WithExportInterval(1*time.Second),
				logsdk.WithExportTimeout(30*time.Second),
				logsdk.WithExportMaxBatchSize(512),
				logsdk.WithExportBufferSize(1),
			)))
		global.SetLoggerProvider(loggerProvider)

		logger := otelslog.NewLogger("this is my test app")

		// send some log records
		logger.Info("Log message 1 HTTP")
		logger.Info("Log message 2 HTTP")
		logger.Info("Log message 3 HTTP", "foo", "bar", "da", "niel")

		err = loggerProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("Should export logs with gzip compression", func(t *testing.T) {
		httpLogExporter, err := otlploghttp.New(t.Context(),
			otlploghttp.WithInsecure(),
			otlploghttp.WithCompression(otlploghttp.GzipCompression),
			otlploghttp.WithEndpoint("localhost:4318"),
		)
		require.NoError(t, err)
		loggerProvider := logsdk.NewLoggerProvider(
			logsdk.WithProcessor(logsdk.NewBatchProcessor(httpLogExporter,
				logsdk.WithMaxQueueSize(2048),
				logsdk.WithExportInterval(1*time.Second),
				logsdk.WithExportTimeout(30*time.Second),
				logsdk.WithExportMaxBatchSize(512),
				logsdk.WithExportBufferSize(1),
			)))
		global.SetLoggerProvider(loggerProvider)

		logger := otelslog.NewLogger("this is my test app")

		// send some log records
		logger.Info("Log message 1 HTTP GZIP")
		logger.Info("Log message 2 HTTP GZIP")
		logger.Info("Log message 3 HTTP GZIP", "foo", "bar", "da", "niel")

		err = loggerProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("Should export metrics with no compression", func(t *testing.T) {
		httpMetricExporter, err := otlpmetrichttp.New(t.Context(),
			otlpmetrichttp.WithInsecure(),
			otlpmetrichttp.WithCompression(otlpmetrichttp.NoCompression),
			otlpmetrichttp.WithEndpoint("localhost:4318"),
		)
		require.NoError(t, err)

		meterProvider := metricsdk.NewMeterProvider(
			metricsdk.WithReader(metricsdk.NewPeriodicReader(httpMetricExporter,
				metricsdk.WithInterval(3*time.Second))),
		)
		otel.SetMeterProvider(meterProvider)

		meter := otel.Meter("this is my test app")
		rollCnt, err := meter.Int64Counter("dice.rolls",
			metric.WithDescription("The number of rolls by roll value"),
			metric.WithUnit("{roll}"))
		require.NoError(t, err)
		rollValueAttr := attribute.Int("roll.value", 3)
		rollCnt.Add(t.Context(), 1, metric.WithAttributes(rollValueAttr))

		err = meterProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

	t.Run("Should export metrics with gzip compression", func(t *testing.T) {
		httpMetricExporter, err := otlpmetrichttp.New(t.Context(),
			otlpmetrichttp.WithInsecure(),
			otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
			otlpmetrichttp.WithEndpoint("localhost:4318"),
		)
		require.NoError(t, err)

		meterProvider := metricsdk.NewMeterProvider(
			metricsdk.WithReader(metricsdk.NewPeriodicReader(httpMetricExporter,
				metricsdk.WithInterval(3*time.Second))),
		)
		otel.SetMeterProvider(meterProvider)

		meter := otel.Meter("this is my test app")
		rollCnt, err := meter.Int64Counter("dice.rolls",
			metric.WithDescription("The number of rolls by roll value"),
			metric.WithUnit("{roll}"))
		require.NoError(t, err)
		rollValueAttr := attribute.Int("roll.value", 3)
		rollCnt.Add(t.Context(), 1, metric.WithAttributes(rollValueAttr))

		err = meterProvider.Shutdown(t.Context())
		require.NoError(t, err)
	})

}
