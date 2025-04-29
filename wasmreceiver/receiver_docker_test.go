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
	"go.opentelemetry.io/collector/pdata/pmetric"
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

func TestS3Metrics(t *testing.T) {
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

	t.Run("metrics", func(t *testing.T) {
		runInContainer(ctx, t, container,
			"mc",
			"cp",
			"--recursive",
			"/testdata/awss3/testdata/metrics/input/",
			"myminio/testbucket",
		)

		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			runInContainer(ctx, t, container,
				"mc",
				"rm",
				"--force",
				"--recursive",
				"myminio/testbucket",
			)
		})

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
		cfg.Path = "testdata/awss3/main.wasm"
		ctx := t.Context()
		settings := receivertest.NewNopSettings(typeStr)

		metricsSink := &consumertest.MetricsSink{}

		ctx, wasmProc, err := newMetricsWasmReceiver(ctx, cfg, metricsSink, settings)
		if err != nil {
			t.Fatalf("failed to create wasm receiver: %v", err)
		}

		// Start the metrics receiver
		err = wasmProc.Start(ctx, nil)
		if err != nil {
			t.Fatalf("failed to start wasm receiver: %v", err)
		}

		// TODO(tsuzu): Use event-driven approach instead of sleep
		time.Sleep(10 * time.Second)

		// Stop the metrics receiver
		err = wasmProc.Shutdown(ctx)
		if err != nil {
			t.Fatalf("failed to stop wasm receiver: %v", err)
		}

		metrics := metricsSink.AllMetrics()
		if len(metrics) == 0 {
			t.Fatal("no metrics received")
		}

		actual := encodeMetricsIntoJSON(t, metrics)

		t.Log(string(actual))

		expected, err := os.ReadFile(filepath.Join(wd, "testdata/awss3/testdata/metrics/output/metrics.json"))
		if err != nil {
			t.Fatalf("failed to read expected metrics file: %v", err)
		}

		if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
			t.Fatalf("expected metrics do not match actual metrics:\n%s", string(actual))
		}
	})
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
