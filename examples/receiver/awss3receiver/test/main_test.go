package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func runInContainer(t *testing.T, container *testcontainers.DockerContainer, commands ...string) {
	t.Helper()
	ctx := t.Context()

	execCode, reader, err := container.Exec(ctx, commands)
	if err != nil {
		t.Fatalf("failed to execute command: %v", err)
	}
	go io.Copy(os.Stderr, reader)
	if execCode != 0 {
		t.Fatalf("failed to execute command: %v", err)
	}
}

func TestMetrics(t *testing.T) {
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
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	runInContainer(t, container,
		"mc",
		"config", "host", "add", "myminio",
		"http://127.0.0.1:9000",
		"minio",
		"minio123",
	)

	runInContainer(t, container,
		"mc",
		"mb",
		"myminio/testbucket",
	)

	runInContainer(t, container,
		"mc",
		"cp",
		"--recursive",
		"/testdata/",
		"myminio/testbucket",
	)

	time.Sleep(3000 * time.Second)
}
