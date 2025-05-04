//go:build docker
// +build docker

package wasmreceiver

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func runInContainer(ctx context.Context, t *testing.T, container *testcontainers.DockerContainer, commands ...string) {
	t.Helper()

	execCode, reader, err := container.Exec(ctx, commands)
	if err != nil {
		t.Fatalf("failed to execute command: %v", err)
	}
	io.Copy(os.Stderr, reader)
	if execCode != 0 {
		t.Fatalf("failed to execute command")
	}
}

func TestS3Receiver(t *testing.T) {
	ctx := t.Context()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	container, err := testcontainers.Run(ctx,
		"quay.io/minio/minio:latest",
		testcontainers.WithExposedPorts("9000:9000/tcp", "9001:9001/tcp"),
		testcontainers.WithEnv(map[string]string{
			"MINIO_ROOT_USER":     "minio",
			"MINIO_ROOT_PASSWORD": "minio123",
			"MINIO_ACCESS_KEY":    "minio",
			"MINIO_SECRET_KEY":    "minio123",
		}),
		testcontainers.WithCmd(
			"server", "/data",
			"--address", ":9000",
			"--console-address", ":9001",
		),
		testcontainers.WithWaitStrategy(wait.ForAll(
			wait.ForListeningPort("9000/tcp"),
			wait.ForListeningPort("9001/tcp"),
		)),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Binds = append(hostConfig.Binds, filepath.Join(wd, "testdata")+":/testdata")
		}),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})
	t.Setenv("AWS_ACCESS_KEY_ID", "minio")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "minio123")

	runInContainer(ctx, t, container,
		"mc",
		"config", "host", "add", "myminio",
		"http://127.0.0.1:9000",
		"minio",
		"minio123",
	)

	runInContainer(ctx, t, container,
		"mc",
		"mb",
		"myminio/testbucket",
	)

	resetBucket := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		runInContainer(ctx, t, container,
			"mc",
			"rm",
			"--force",
			"--recursive",
			"myminio/testbucket",
		)
	}

	cases := []struct {
		name         string
		inputDir     string
		expectedPath string
		prepare      func(t *testing.T, sink *sink) (context.Context, *Receiver)
	}{
		{
			name:         "metrics",
			inputDir:     "testdata/awss3receiver/testdata/metrics/input/",
			expectedPath: "testdata/awss3receiver/testdata/metrics/output/metrics.json",
			prepare: func(t *testing.T, sink *sink) (context.Context, *Receiver) {
				cfg := createDefaultConfig().(*Config)
				cfg.PluginConfig = map[string]any{
					"starttime": "2025-01-01 00:00",
					"endtime":   "2025-01-01 01:00",
					"s3downloader": map[string]any{
						"region":              "us-east-1",
						"s3_bucket":           "testbucket",
						"s3_partition":        "hour",
						"endpoint":            "http://127.0.0.1:9000",
						"disable_ssl":         true,
						"s3_force_path_style": true,
					},
				}

				cfg.Path = "testdata/awss3receiver/main.wasm"
				ctx := t.Context()
				settings := receivertest.NewNopSettings(typeStr)

				ctx, wasmProc, err := newMetricsWasmReceiver(ctx, cfg, &sink.metricsSink, settings)
				if err != nil {
					t.Fatalf("failed to create wasm receiver: %v", err)
				}

				return ctx, wasmProc
			},
		},
		{
			name:         "logs",
			inputDir:     "testdata/awss3receiver/testdata/logs/input/",
			expectedPath: "testdata/awss3receiver/testdata/logs/output/logs.json",
			prepare: func(t *testing.T, sink *sink) (context.Context, *Receiver) {
				cfg := createDefaultConfig().(*Config)
				cfg.PluginConfig = map[string]any{
					"starttime": "2025-01-01 00:00",
					"endtime":   "2025-01-01 01:00",
					"s3downloader": map[string]any{
						"region":              "us-east-1",
						"s3_bucket":           "testbucket",
						"s3_partition":        "hour",
						"endpoint":            "http://127.0.0.1:9000",
						"disable_ssl":         true,
						"s3_force_path_style": true,
					},
				}

				cfg.Path = "testdata/awss3receiver/main.wasm"
				ctx := t.Context()
				settings := receivertest.NewNopSettings(typeStr)

				ctx, wasmProc, err := newLogsWasmReceiver(ctx, cfg, &sink.logsSink, settings)
				if err != nil {
					t.Fatalf("failed to create wasm receiver: %v", err)
				}

				return ctx, wasmProc
			},
		},
		{
			name:         "traces",
			inputDir:     "testdata/awss3receiver/testdata/traces/input/",
			expectedPath: "testdata/awss3receiver/testdata/traces/output/traces.json",
			prepare: func(t *testing.T, sink *sink) (context.Context, *Receiver) {
				cfg := createDefaultConfig().(*Config)
				cfg.PluginConfig = map[string]any{
					"starttime": "2025-01-01 00:00",
					"endtime":   "2025-01-01 01:00",
					"s3downloader": map[string]any{
						"region":              "us-east-1",
						"s3_bucket":           "testbucket",
						"s3_partition":        "hour",
						"endpoint":            "http://127.0.0.1:9000",
						"disable_ssl":         true,
						"s3_force_path_style": true,
					},
				}

				cfg.Path = "testdata/awss3receiver/main.wasm"
				ctx := t.Context()
				settings := receivertest.NewNopSettings(typeStr)

				ctx, wasmProc, err := newTracesWasmReceiver(ctx, cfg, &sink.tracesSink, settings)
				if err != nil {
					t.Fatalf("failed to create wasm receiver: %v", err)
				}

				return ctx, wasmProc
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runInContainer(ctx, t, container,
				"mc",
				"cp",
				"--recursive",
				tc.inputDir,
				"myminio/testbucket",
			)

			t.Cleanup(resetBucket)

			sink := &sink{}
			ctx, wasmProc := tc.prepare(t, sink)

			// Start the receiver
			err = wasmProc.Start(ctx, nil)
			if err != nil {
				t.Fatalf("failed to start wasm receiver: %v", err)
			}

			// TODO(tsuzu): Use event-driven approach instead of sleep
			time.Sleep(5 * time.Second)

			// Stop the receiver
			err = wasmProc.Shutdown(ctx)
			if err != nil {
				t.Fatalf("failed to stop wasm receiver: %v", err)
			}

			actual := sink.encodeSink(t)
			t.Log(string(actual))

			expected, err := os.ReadFile(filepath.Join(wd, tc.expectedPath))
			if err != nil {
				t.Fatalf("failed to read expected file: %v", err)
			}
			if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
				t.Fatalf("expected data do not match actual data:\n%s", string(actual))
			}
		})
	}
}

type sink struct {
	metricsSink consumertest.MetricsSink
	logsSink    consumertest.LogsSink
	tracesSink  consumertest.TracesSink
}

func (s *sink) encodeSink(t *testing.T) []byte {
	if s.metricsSink.DataPointCount() != 0 {
		metrics := s.metricsSink.AllMetrics()
		if len(metrics) == 0 {
			t.Fatal("no metrics received")
		}
		return encodeMetricsIntoJSON(t, metrics)
	}
	if s.logsSink.LogRecordCount() != 0 {
		logs := s.logsSink.AllLogs()
		if len(logs) == 0 {
			t.Fatal("no logs received")
		}
		return encodeLogsIntoJSON(t, logs)
	}
	if s.tracesSink.SpanCount() != 0 {
		traces := s.tracesSink.AllTraces()
		if len(traces) == 0 {
			t.Fatal("no traces received")
		}
		return encodeTracesIntoJSON(t, traces)
	}
	t.Fatal("no data received")
	return nil
}

func encodeMetricsIntoJSON(t *testing.T, metrics []pmetric.Metrics) []byte {
	t.Helper()

	marshaler := pmetric.JSONMarshaler{}

	var buf bytes.Buffer
	for _, m := range metrics {
		jsonBytes, err := marshaler.MarshalMetrics(m)
		if err != nil {
			t.Fatalf("failed to marshal metrics: %v", err)
		}
		buf.Write(jsonBytes)
	}

	return buf.Bytes()
}

func encodeLogsIntoJSON(t *testing.T, logs []plog.Logs) []byte {
	t.Helper()

	marshaler := plog.JSONMarshaler{}

	var buf bytes.Buffer
	for _, l := range logs {
		jsonBytes, err := marshaler.MarshalLogs(l)
		if err != nil {
			t.Fatalf("failed to marshal logs: %v", err)
		}
		buf.Write(jsonBytes)
	}

	return buf.Bytes()
}

func encodeTracesIntoJSON(t *testing.T, traces []ptrace.Traces) []byte {
	t.Helper()

	marshaler := ptrace.JSONMarshaler{}

	var buf bytes.Buffer
	for _, tr := range traces {
		jsonBytes, err := marshaler.MarshalTraces(tr)
		if err != nil {
			t.Fatalf("failed to marshal traces: %v", err)
		}
		buf.Write(jsonBytes)
	}

	return buf.Bytes()
}
