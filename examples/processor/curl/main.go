package main

import (
	"io"
	"net/http"
	"time"

	"github.com/otelwasm/otelwasm/guest/api"
	"github.com/otelwasm/otelwasm/guest/plugin" // register tracesprocessor
	"github.com/stealthrocket/net/wasip1"       // for wasip1 dialer
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func init() {
	plugin.Set(&CurlProcessor{})
}
func main() {}

var (
	_ api.TracesProcessor  = (*CurlProcessor)(nil)
	_ api.MetricsProcessor = (*CurlProcessor)(nil)
	_ api.LogsProcessor    = (*CurlProcessor)(nil)
)

// Create a custom http client that uses stealthrocket's net implementation
var httpClient = &http.Client{
	Transport: &http.Transport{
		// Use wasip1's DialContext for WASI socket compatibility
		DialContext: wasip1.DialContext,
		// Add reasonable timeouts
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	},
	Timeout: 60 * time.Second,
}

type CurlProcessor struct{}

// ProcessTraces implements api.TracesProcessor.
func (n *CurlProcessor) ProcessTraces(traces ptrace.Traces) (ptrace.Traces, *api.Status) {
	// Make a GET request to example.com using http.Client
	resp, err := httpClient.Get("http://example.com")
	if err != nil {
		return traces, api.StatusError(err.Error())
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return traces, api.StatusError(err.Error())
	}

	// Output results
	println("Response received from example.com:")
	println("Status:", resp.Status)
	println("Body:", string(body[:min(len(body), 200)]), "...") // Print first 200 chars of body

	return traces, nil
}

// ProcessMetrics implements api.MetricsProcessor.
func (n *CurlProcessor) ProcessMetrics(metrics pmetric.Metrics) (pmetric.Metrics, *api.Status) {
	// Make a GET request to example.com using http.Client
	resp, err := httpClient.Get("http://example.com")
	if err != nil {
		return metrics, api.StatusError(err.Error())
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return metrics, api.StatusError(err.Error())
	}

	// Output results
	println("Response received from example.com:")
	println("Status:", resp.Status)
	println("Body:", string(body[:min(len(body), 200)]), "...") // Print first 200 chars of body

	return metrics, nil
}

// ProcessLogs implements api.LogsProcessor.
func (n *CurlProcessor) ProcessLogs(logs plog.Logs) (plog.Logs, *api.Status) {
	// Make a GET request to example.com using http.Client
	resp, err := httpClient.Get("http://example.com")
	if err != nil {
		return logs, api.StatusError(err.Error())
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return logs, api.StatusError(err.Error())
	}

	// Output results
	println("Response received from example.com:")
	println("Status:", resp.Status)
	println("Body:", string(body[:min(len(body), 200)]), "...") // Print first 200 chars of body

	return logs, nil
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
