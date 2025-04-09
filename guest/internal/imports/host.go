/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package imports

import (
	"runtime"

	"github.com/musaprg/otelwasm/guest/api"
	"github.com/musaprg/otelwasm/guest/internal/mem"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// StatusToCode returns a WebAssembly compatible result for the input status,
// after sending any reason to the host.
func StatusToCode(s *api.Status) uint32 {
	// Nil status is the same as one with a success code.
	if s == nil || s.Code == api.StatusCodeSuccess {
		return uint32(api.StatusCodeSuccess)
	}
	return uint32(s.Code)
}

func CurrentTraces() ptrace.Traces {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return currentTraces(ptr, limit)
	})
	unmarshaler := ptrace.ProtoUnmarshaler{}
	traces, err := unmarshaler.UnmarshalTraces(rawMsg)
	if err != nil {
		panic(err)
	}
	return traces
}

func CurrentMetrics() pmetric.Metrics {
	rawMsg := mem.GetBytes(func(ptr uint32, limit mem.BufLimit) (len uint32) {
		return currentMetrics(ptr, limit)
	})
	unmarshaler := pmetric.ProtoUnmarshaler{}
	metrics, err := unmarshaler.UnmarshalMetrics(rawMsg)
	if err != nil {
		panic(err)
	}
	return metrics
}

func SetResultTraces(traces ptrace.Traces) {
	marshaler := ptrace.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalTraces(traces)
	if err != nil {
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultTraces(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}

func SetResultMetrics(metrics pmetric.Metrics) {
	marshaler := pmetric.ProtoMarshaler{}
	rawMsg, err := marshaler.MarshalMetrics(metrics)
	if err != nil {
		panic(err)
	}
	ptr, size := mem.BytesToPtr(rawMsg)
	setResultMetrics(ptr, size)
	runtime.KeepAlive(rawMsg) // until ptr is no longer needed
}
