package wasmexporter

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
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

func TestS3Exporter(t *testing.T) {
	ctx := t.Context()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Start a Minio container for S3-compatible storage
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

	// Set environment variables for AWS SDK
	t.Setenv("AWS_ACCESS_KEY_ID", "minio")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "minio123")

	// Set up Minio bucket
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

	// Helper function to reset the bucket between tests
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

		runInContainer(ctx, t, container,
			"mc",
			"mb",
			"myminio/testbucket",
		)
	}

	// Create S3 client to verify data was uploaded
	s3Client := createS3Client(t)

	cases := []struct {
		name            string
		dataType        string
		prepareData     func() interface{}
		exportData      func(context.Context, interface{}, *wasmExporterImp) error
		expectedPrefix  string
		validateContent func(t *testing.T, content []byte) bool
	}{
		{
			name:     "metrics",
			dataType: "metrics",
			prepareData: func() interface{} {
				// Create test metrics
				metrics := pmetric.NewMetrics()
				rm := metrics.ResourceMetrics().AppendEmpty()
				rm.Resource().Attributes().PutStr("service.name", "test-service")
				ilm := rm.ScopeMetrics().AppendEmpty()
				ilm.Scope().SetName("test-scope")
				metric := ilm.Metrics().AppendEmpty()
				metric.SetName("test-metric")
				metric.SetEmptyGauge().DataPoints().AppendEmpty().SetIntValue(42)
				return metrics
			},
			exportData: func(ctx context.Context, data interface{}, exporter *wasmExporterImp) error {
				return exporter.pushMetrics(ctx, data.(pmetric.Metrics))
			},
			expectedPrefix: "metrics/",
			validateContent: func(t *testing.T, content []byte) bool {
				// Simple validation: check if the content contains expected data
				var metricsData map[string]interface{}
				if err := json.Unmarshal(content, &metricsData); err != nil {
					t.Logf("Failed to parse metrics JSON: %v", err)
					return false
				}

				// Check for resourceMetrics
				resourceMetrics, ok := metricsData["resourceMetrics"].([]interface{})
				if !ok || len(resourceMetrics) == 0 {
					return false
				}

				// Look for the test-metric name
				found := false
				jsonStr := string(content)
				if strings.Contains(jsonStr, "test-metric") &&
					strings.Contains(jsonStr, "test-service") &&
					strings.Contains(jsonStr, "test-scope") {
					found = true
				}

				return found
			},
		},
		{
			name:     "traces",
			dataType: "traces",
			prepareData: func() interface{} {
				// Create test traces
				traces := ptrace.NewTraces()
				rs := traces.ResourceSpans().AppendEmpty()
				rs.Resource().Attributes().PutStr("service.name", "test-service")
				ss := rs.ScopeSpans().AppendEmpty()
				ss.Scope().SetName("test-scope")
				span := ss.Spans().AppendEmpty()
				span.SetName("test-span")
				span.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
				span.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
				return traces
			},
			exportData: func(ctx context.Context, data interface{}, exporter *wasmExporterImp) error {
				return exporter.pushTraces(ctx, data.(ptrace.Traces))
			},
			expectedPrefix: "traces/",
			validateContent: func(t *testing.T, content []byte) bool {
				// Simple validation: check if the content contains expected data
				var tracesData map[string]interface{}
				if err := json.Unmarshal(content, &tracesData); err != nil {
					t.Logf("Failed to parse traces JSON: %v", err)
					return false
				}

				// Check for resourceSpans
				jsonStr := string(content)
				return strings.Contains(jsonStr, "test-span") &&
					strings.Contains(jsonStr, "test-service") &&
					strings.Contains(jsonStr, "test-scope")
			},
		},
		{
			name:     "logs",
			dataType: "logs",
			prepareData: func() interface{} {
				// Create test logs
				logs := plog.NewLogs()
				rl := logs.ResourceLogs().AppendEmpty()
				rl.Resource().Attributes().PutStr("service.name", "test-service")
				sl := rl.ScopeLogs().AppendEmpty()
				sl.Scope().SetName("test-scope")
				logRecord := sl.LogRecords().AppendEmpty()
				logRecord.SetSeverityText("INFO")
				logRecord.Body().SetStr("test message")
				return logs
			},
			exportData: func(ctx context.Context, data interface{}, exporter *wasmExporterImp) error {
				return exporter.pushLogs(ctx, data.(plog.Logs))
			},
			expectedPrefix: "logs/",
			validateContent: func(t *testing.T, content []byte) bool {
				// Simple validation: check if the content contains expected data
				var logsData map[string]interface{}
				if err := json.Unmarshal(content, &logsData); err != nil {
					t.Logf("Failed to parse logs JSON: %v", err)
					return false
				}

				// Check for specific content
				jsonStr := string(content)
				return strings.Contains(jsonStr, "test message") &&
					strings.Contains(jsonStr, "INFO") &&
					strings.Contains(jsonStr, "test-service") &&
					strings.Contains(jsonStr, "test-scope")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset bucket for each test
			resetBucket()

			// Create exporter configuration
			cfg := createDefaultConfig().(*Config)
			cfg.Path = "testdata/awss3exporter/main.wasm"
			cfg.PluginConfig = map[string]any{
				"s3uploader": map[string]any{
					"region":              "us-east-1",
					"s3_bucket":           "testbucket",
					"s3_partition":        tc.dataType,
					"endpoint":            "http://localhost:9000",
					"disable_ssl":         true,
					"s3_force_path_style": true,
				},
			}

			// Create and start the exporter based on data type
			var wasmExp *wasmExporterImp
			var err error
			settings := exportertest.NewNopSettings(typeStr)
			host := componenttest.NewNopHost(typeStr)

			switch tc.dataType {
			case "metrics":
				me, err := newWasmMetricsExporter(ctx, cfg)
				if err != nil {
					t.Fatalf("failed to create metrics exporter: %v", err)
				}
				wasmExp = me
				err = me.Start(ctx, host)
				if err != nil {
					t.Fatalf("failed to start exporter: %v", err)
				}
				defer me.Shutdown(ctx)
			case "traces":
				te, err := newWasmTracesExporter(ctx, cfg)
				if err != nil {
					t.Fatalf("failed to create traces exporter: %v", err)
				}
				wasmExp = te
				err = te.Start(ctx, host)
				if err != nil {
					t.Fatalf("failed to start exporter: %v", err)
				}
				defer te.Shutdown(ctx)
			case "logs":
				le, err := newWasmLogsExporter(ctx, cfg)
				if err != nil {
					t.Fatalf("failed to create logs exporter: %v", err)
				}
				wasmExp = le
				err = le.Start(ctx, host)
				if err != nil {
					t.Fatalf("failed to start exporter: %v", err)
				}
				defer le.Shutdown(ctx)
			}

			// Export data
			testData := tc.prepareData()
			err = tc.exportData(ctx, testData, wasmExp)
			if err != nil {
				t.Fatalf("failed to export data: %v", err)
			}

			// Allow time for the exporter to process and upload
			time.Sleep(5 * time.Second)

			// Verify data was uploaded to S3
			listOutput, err := s3Client.ListObjectsV2(&s3.ListObjectsV2Input{
				Bucket: aws.String("testbucket"),
				Prefix: aws.String(tc.expectedPrefix),
			})
			if err != nil {
				t.Fatalf("failed to list objects: %v", err)
			}

			if len(listOutput.Contents) == 0 {
				t.Fatalf("no objects found in S3 bucket with prefix %s", tc.expectedPrefix)
			}

			// Log the uploaded files for debugging
			t.Logf("Found %d objects in S3 bucket", len(listOutput.Contents))
			for _, obj := range listOutput.Contents {
				t.Logf("Object: %s, Size: %d", aws.StringValue(obj.Key), aws.Int64Value(obj.Size))

				// Download and validate the content of each object
				getOutput, err := s3Client.GetObject(&s3.GetObjectInput{
					Bucket: aws.String("testbucket"),
					Key:    obj.Key,
				})
				if err != nil {
					t.Fatalf("failed to get object %s: %v", aws.StringValue(obj.Key), err)
				}

				content, err := io.ReadAll(getOutput.Body)
				if err != nil {
					t.Fatalf("failed to read object content: %v", err)
				}
				getOutput.Body.Close()

				if content != nil && len(content) > 0 {
					t.Logf("Content sample: %s", truncateString(string(content), 200))

					if tc.validateContent != nil {
						if !tc.validateContent(t, content) {
							t.Errorf("content validation failed for object %s", aws.StringValue(obj.Key))
						} else {
							t.Logf("Content validation passed for object %s", aws.StringValue(obj.Key))
						}
					}
				} else {
					t.Errorf("empty content for object %s", aws.StringValue(obj.Key))
				}
			}
		})
	}
}

// Helper function to truncate long strings for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// Create an S3 client configured to talk to the local Minio instance
func createS3Client(t *testing.T) *s3.S3 {
	t.Helper()

	s3Config := aws.NewConfig().
		WithRegion("us-east-1").
		WithEndpoint("http://localhost:9000").
		WithCredentials(credentials.NewStaticCredentials("minio", "minio123", "")).
		WithS3ForcePathStyle(true).
		WithDisableSSL(true)

	sess, err := session.NewSession(s3Config)
	if err != nil {
		t.Fatalf("failed to create S3 session: %v", err)
	}

	return s3.New(sess)
}
