// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package webhookeventreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/webhookeventreceiver"

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/julienschmidt/httprouter"
	wasinet "github.com/musaprg/dispatchrunnet/wasip1"
	"github.com/rs/cors"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
)

var (
	errNilLogsConsumer       = errors.New("missing a logs consumer")
	errInvalidRequestMethod  = errors.New("invalid method. Valid method is POST")
	errInvalidEncodingType   = errors.New("invalid encoding type")
	errEmptyResponseBody     = errors.New("request body content length is zero")
	errMissingRequiredHeader = errors.New("request was missing required header or incorrect header value")
)

const healthyResponse = `{"text": "Webhookevent receiver is healthy"}`

type eventReceiver struct {
	cfg                 *Config
	logConsumer         consumer.Logs
	server              *http.Server
	shutdownWG          sync.WaitGroup
	gzipPool            *sync.Pool
	includeHeadersRegex *regexp.Regexp
}

func NewLogsReceiver(cfg Config, consumer consumer.Logs) (receiver.Logs, error) {
	if consumer == nil {
		return nil, errNilLogsConsumer
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	var includeHeaderRegex *regexp.Regexp
	if cfg.HeaderAttributeRegex != "" {
		// Valdiate() call above has already ensured this will compile
		includeHeaderRegex, _ = regexp.Compile(cfg.HeaderAttributeRegex)
	}

	// create eventReceiver instance
	er := &eventReceiver{
		cfg:                 &cfg,
		logConsumer:         consumer,
		gzipPool:            &sync.Pool{New: func() any { return new(gzip.Reader) }},
		includeHeadersRegex: includeHeaderRegex,
	}

	return er, nil
}

// Start function manages receiver startup tasks. part of the receiver.Logs interface.
func (er *eventReceiver) Start(ctx context.Context, host component.Host) error {
	// noop if not nil. if start has not been called before these values should be nil.
	if er.server != nil && er.server.Handler != nil {
		return nil
	}

	ln, err := wasinet.Listen("tcp", er.cfg.Endpoint)
	if err != nil {
		return err
	}

	if er.cfg.TLSSetting != nil {
		var tlsCfg *tls.Config
		tlsCfg, err = er.cfg.TLSSetting.LoadTLSConfig(ctx)
		if err != nil {
			return err
		}
		tlsCfg.NextProtos = []string{http2.NextProtoTLS, "http/1.1"}
		ln = tls.NewListener(ln, tlsCfg)
	}

	// set up router.
	router := httprouter.New()

	router.POST(er.cfg.Path, er.handleReq)
	router.GET(er.cfg.HealthPath, er.handleHealthCheck)

	// webhook server standup and configuration
	if er.cfg.MaxRequestBodySize <= 0 {
		er.cfg.MaxRequestBodySize = defaultMaxRequestBodySize
	}

	if er.cfg.CompressionAlgorithms == nil {
		er.cfg.CompressionAlgorithms = defaultCompressionAlgorithms
	}

	var handler http.Handler = router

	if er.cfg.MaxRequestBodySize > 0 {
		handler = maxRequestBodySizeInterceptor(handler, er.cfg.MaxRequestBodySize)
	}

	if er.cfg.Auth != nil {
		server, err := er.cfg.Auth.GetServerAuthenticator(context.Background(), host.GetExtensions())
		if err != nil {
			return err
		}

		handler = authInterceptor(handler, server, er.cfg.Auth.RequestParameters)
	}

	if er.cfg.CORS != nil && len(er.cfg.CORS.AllowedOrigins) > 0 {
		co := cors.Options{
			AllowedOrigins:   er.cfg.CORS.AllowedOrigins,
			AllowCredentials: true,
			AllowedHeaders:   er.cfg.CORS.AllowedHeaders,
			MaxAge:           er.cfg.CORS.MaxAge,
		}
		handler = cors.New(co).Handler(handler)
	}

	if er.cfg.ResponseHeaders != nil {
		handler = responseHeadersHandler(handler, er.cfg.ResponseHeaders)
	}

	// Enable OpenTelemetry observability plugin.
	handler = otelhttp.NewHandler(handler, "")

	// wrap the current handler in an interceptor that will add client.Info to the request's context
	handler = &clientInfoHandler{
		next:            handler,
		includeMetadata: er.cfg.IncludeMetadata,
	}

	er.server = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: er.cfg.ReadHeaderTimeout,
		IdleTimeout:       er.cfg.IdleTimeout,
	}

	readTimeout, err := time.ParseDuration(er.cfg.ReadTimeout)
	if err != nil {
		return err
	}

	writeTimeout, err := time.ParseDuration(er.cfg.WriteTimeout)
	if err != nil {
		return err
	}

	// set timeouts
	er.server.ReadHeaderTimeout = readTimeout
	er.server.WriteTimeout = writeTimeout

	// shutdown
	er.shutdownWG.Add(1)
	go func() {
		defer er.shutdownWG.Done()
		if errHTTP := er.server.Serve(ln); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(errHTTP))
		}
	}()

	return nil
}

// Shutdown function manages receiver shutdown tasks. part of the receiver.Logs interface.
func (er *eventReceiver) Shutdown(_ context.Context) error {
	// server must exist to be closed.
	if er.server == nil {
		return nil
	}

	err := er.server.Close()
	er.shutdownWG.Wait()
	return err
}

// handleReq handles incoming request from webhook. On success returns a 200 response code to the webhook
func (er *eventReceiver) handleReq(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		er.failBadReq(ctx, w, http.StatusBadRequest, errInvalidRequestMethod)
		return
	}

	if er.cfg.RequiredHeader.Key != "" {
		requiredHeaderValue := r.Header.Get(er.cfg.RequiredHeader.Key)
		if requiredHeaderValue != er.cfg.RequiredHeader.Value {
			er.failBadReq(ctx, w, http.StatusUnauthorized, errMissingRequiredHeader)
			return
		}
	}

	encoding := r.Header.Get("Content-Encoding")
	// only support gzip if encoding header is set.
	if encoding != "" && encoding != "gzip" {
		er.failBadReq(ctx, w, http.StatusUnsupportedMediaType, errInvalidEncodingType)
		return
	}

	if r.ContentLength == 0 {
		er.failBadReq(ctx, w, http.StatusBadRequest, errEmptyResponseBody)
	}

	bodyReader := r.Body
	// gzip encoded case
	if encoding == "gzip" || encoding == "x-gzip" {
		reader := er.gzipPool.Get().(*gzip.Reader)
		err := reader.Reset(bodyReader)
		if err != nil {
			er.failBadReq(ctx, w, http.StatusBadRequest, err)
			_, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
			return
		}
		bodyReader = reader
		defer er.gzipPool.Put(reader)
	}

	// send body into a scanner and then convert the request body into a log
	sc := bufio.NewScanner(bodyReader)
	ld, _ := er.reqToLog(sc, r.Header, r.URL.Query())
	consumerErr := er.logConsumer.ConsumeLogs(ctx, ld)

	_ = bodyReader.Close()

	if consumerErr != nil {
		er.failBadReq(ctx, w, http.StatusInternalServerError, consumerErr)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// Simple healthcheck endpoint.
func (er *eventReceiver) handleHealthCheck(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	// somehow adding MIME header failed because of the following weird runtime error
	/*
		fatal error: bulkBarrierPreWrite: unaligned arguments

		goroutine 9 gp=0x1400fc0 m=0 mp=0xfa8d40 [running]:
		runtime.throw({0x252a69, 0x28})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1096 +0x3 fp=0x14c1390 sp=0x14c1368 pc=0x15fb0003
		runtime.bulkBarrierPreWriteSrcOnly(0x1580000, 0x808080808080652c, 0x232f1a8, 0xc7aa0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mbitmap.go:505 +0x36 fp=0x14c1418 sp=0x14c1390 pc=0x12030036
		runtime.growslice(0x808080808080652c, 0x232f1c, 0x4, 0x1, 0xc7aa0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/slice.go:280 +0x8f fp=0x14c1468 sp=0x14c1418 pc=0x1628008f
		net/textproto.MIMEHeader.Add(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/textproto/header.go:15
		net/http.Header.Add(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/header.go:31
		github.com/musaprg/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver.(*eventReceiver).handleHealthCheck(0x1477040, {0x3501d0, 0x14587e0}, 0x1566000, {0x0, 0x0, 0x0})
				/Users/musaprg/workspace/personal/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver/receiver.go:253 +0xd fp=0x14c14d8 sp=0x14c1468 pc=0x6a5c000d
		github.com/musaprg/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver.(*eventReceiver).handleHealthCheck-fm({0x3501d0, 0x14587e0}, 0x1566000, {0x0, 0x0, 0x0})
				<autogenerated>:1 +0x2 fp=0x14c1518 sp=0x14c14d8 pc=0x6a6c0002
		github.com/julienschmidt/httprouter.(*Router).ServeHTTP(0x14584e0, {0x3501d0, 0x14587e0}, 0x1566000)
				/Users/musaprg/go/pkg/mod/github.com/julienschmidt/httprouter@v1.3.0/router.go:387 +0x9e fp=0x14c1640 sp=0x14c1518 pc=0x69cc009e
		github.com/musaprg/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver.(*eventReceiver).Start.maxRequestBodySizeInterceptor.func2({0x3501d0, 0x14587e0}, 0x1566000)
				/Users/musaprg/workspace/personal/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver/confighttp.go:443 +0x9 fp=0x14c1698 sp=0x14c1640 pc=0x6a580009
		net/http.HandlerFunc.ServeHTTP(0x1457140, {0x3501d0, 0x14587e0}, 0x1566000)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/server.go:2294 +0x4 fp=0x14c16b8 sp=0x14c1698 pc=0x2fd30004
		go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp.(*middleware).serveHTTP(0x14e4100, {0x34f2f8, 0x14e80e0}, 0x14ede00, {0x347918, 0x1457140})
				/Users/musaprg/go/pkg/mod/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@v0.60.0/handler.go:179 +0x11f fp=0x14c1aa8 sp=0x14c16b8 pc=0x3ad8011f
		go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp.NewMiddleware.func1.1({0x34f2f8, 0x14e80e0}, 0x14ede00)
				/Users/musaprg/go/pkg/mod/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@v0.60.0/handler.go:67 +0x2 fp=0x14c1ae0 sp=0x14c1aa8 pc=0x3ad30002
		net/http.HandlerFunc.ServeHTTP(0x14571e0, {0x34f2f8, 0x14e80e0}, 0x14ede00)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/server.go:2294 +0x4 fp=0x14c1b00 sp=0x14c1ae0 pc=0x2fd30004
		github.com/musaprg/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver.(*clientInfoHandler).ServeHTTP(0x140aba0, {0x34f2f8, 0x14e80e0}, 0x14edcc0)
				/Users/musaprg/workspace/personal/otelwasm/examples/receiver/webhookeventreceiver/webhookeventreceiver/clientinfohandler.go:26 +0x14 fp=0x14c1b40 sp=0x14c1b00 pc=0x6a4c0014
		net/http.serverHandler.ServeHTTP({0x14e4200}, {0x34f2f8, 0x14e80e0}, 0x14edcc0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/server.go:3301 +0x13 fp=0x14c1b78 sp=0x14c1b40 pc=0x30da0013
		net/http.(*conn).serve(0x14d8e10, {0x351e50, 0x1544ae0})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/server.go:2102 +0x90 fp=0x14c1fc0 sp=0x14c1b78 pc=0x2fc70090
		net/http.(*Server).Serve.gowrap3()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/server.go:3454 +0x2 fp=0x14c1fe0 sp=0x14c1fc0 pc=0x2ff60002
		runtime.goexit({})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/asm_wasm.s:434 +0x1 fp=0x14c1fe8 sp=0x14c1fe0 pc=0x166e0001
		created by net/http.(*Server).Serve in goroutine 8
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/net/http/server.go:3454 +0x3e

		goroutine 1 gp=0x14001c0 m=nil [chan receive]:
		runtime.gopark(0x270f28, 0x145a1b0, 0xe, 0x7, 0x2)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:435 +0x22 fp=0x151bbb0 sp=0x151bb88 pc=0x15fd0022
		runtime.chanrecv(0x145a150, 0x0, 0x1)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/chan.go:664 +0x6e fp=0x151bc30 sp=0x151bbb0 pc=0x11a3006e
		runtime.chanrecv1(0x145a150, 0x0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/chan.go:506 +0x2 fp=0x151bc58 sp=0x151bc30 pc=0x11a10002
		main.(*WebhookEventReceiver).StartLogs(0xfb7e40, {0x351e88, 0x14b29b0})
				/Users/musaprg/workspace/personal/otelwasm/examples/receiver/webhookeventreceiver/main.go:64 +0x18 fp=0x151bf78 sp=0x151bc58 pc=0x6aaa0018
		github.com/musaprg/otelwasm/guest/logsreceiver._startLogsReceiver()
				/Users/musaprg/workspace/personal/otelwasm/guest/logsreceiver/logsreceiver.go:46 +0xc fp=0x151bfd8 sp=0x151bf78 pc=0x6a8a000c
		startLogsReceiver()
				<autogenerated>:1 fp=0x151bfe0 sp=0x151bfd8 pc=0x6a8d0000
		runtime.goexit({})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/asm_wasm.s:434 +0x1 fp=0x151bfe8 sp=0x151bfe0 pc=0x166e0001

		goroutine 2 gp=0x1400380 m=nil [force gc (idle)]:
		runtime.gopark(0x2711f0, 0xfa5310, 0x11, 0xa, 0x1)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:435 +0x22 fp=0x1450fb0 sp=0x1450f88 pc=0x15fd0022
		runtime.goparkunlock(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:441
		runtime.forcegchelper()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:348 +0x1b fp=0x1450fe0 sp=0x1450fb0 pc=0x13a8001b
		runtime.goexit({})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/asm_wasm.s:434 +0x1 fp=0x1450fe8 sp=0x1450fe0 pc=0x166e0001
		created by runtime.init.6 in goroutine 1
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:336 +0x2

		goroutine 3 gp=0x1400540 m=nil [GC sweep wait]:
		runtime.gopark(0x2711f0, 0xfa5ae0, 0xc, 0x9, 0x1)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:435 +0x22 fp=0x1451790 sp=0x1451768 pc=0x15fd0022
		runtime.goparkunlock(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:441
		runtime.bgsweep(0x1454000)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mgcsweep.go:276 +0xc fp=0x14517d0 sp=0x1451790 pc=0x12b3000c
		runtime.gcenable.gowrap1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mgc.go:204 +0x2 fp=0x14517e0 sp=0x14517d0 pc=0x12340002
		runtime.goexit({})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/asm_wasm.s:434 +0x1 fp=0x14517e8 sp=0x14517e0 pc=0x166e0001
		created by runtime.gcenable in goroutine 1
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mgc.go:204 +0x6

		goroutine 4 gp=0x1400700 m=nil [runnable]:
		runtime.gopark(0x2711f0, 0xfa7ac0, 0xd, 0xa, 0x2)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:435 +0x22 fp=0x1451f80 sp=0x1451f58 pc=0x15fd0022
		runtime.goparkunlock(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:441
		runtime.(*scavengerState).park(0xfa7ac0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mgcscavenge.go:425 +0xc fp=0x1451fa8 sp=0x1451f80 pc=0x1291000c
		runtime.bgscavenge(0x1454000)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mgcscavenge.go:653 +0x4 fp=0x1451fd0 sp=0x1451fa8 pc=0x12960004
		runtime.gcenable.gowrap2()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/mgc.go:205 +0x2 fp=0x1451fe0 sp=0x1451fd0 pc=0x12330002
		runtime.goexit({})
				/opt/homebrewfatal error: fd_write failed
		panic during panic

		runtime stack:
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1096 +0x3 fp=0xfc0610 sp=0xfc05e8 pc=0x15fb0003
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc0650 sp=0xfc0610 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc0678 sp=0xfc0650 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc06a0 sp=0xfc0678 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc06d0 sp=0xfc06a0 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc0718 sp=0xfc06d0 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc0740 sp=0xfc0718 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc0768 sp=0xfc0740 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc07a8 sp=0xfc0768 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc07d0 sp=0xfc07a8 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc07f8 sp=0xfc07d0 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc0828 sp=0xfc07f8 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc0870 sp=0xfc0828 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc0898 sp=0xfc0870 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc08c0 sp=0xfc0898 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc0900 sp=0xfc08c0 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc0928 sp=0xfc0900 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc0950 sp=0xfc0928 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc0980 sp=0xfc0950 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc09c8 sp=0xfc0980 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc09f0 sp=0xfc09c8 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc0a18 sp=0xfc09f0 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc0a58 sp=0xfc0a18 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc0a80 sp=0xfc0a58 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc0aa8 sp=0xfc0a80 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc0ad8 sp=0xfc0aa8 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc0b20 sp=0xfc0ad8 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc0b48 sp=0xfc0b20 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc0b70 sp=0xfc0b48 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc0bb0 sp=0xfc0b70 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc0bd8 sp=0xfc0bb0 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc0c00 sp=0xfc0bd8 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc0c30 sp=0xfc0c00 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc0c78 sp=0xfc0c30 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc0ca0 sp=0xfc0c78 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc0cc8 sp=0xfc0ca0 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc0d08 sp=0xfc0cc8 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc0d30 sp=0xfc0d08 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc0d58 sp=0xfc0d30 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc0d88 sp=0xfc0d58 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc0dd0 sp=0xfc0d88 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc0df8 sp=0xfc0dd0 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc0e20 sp=0xfc0df8 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc0e60 sp=0xfc0e20 pc=0x135a0005
		...14 frames elided...
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc10d0 sp=0xfc10a8 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc1110 sp=0xfc10d0 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc1138 sp=0xfc1110 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc1160 sp=0xfc1138 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc1190 sp=0xfc1160 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc11d8 sp=0xfc1190 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc1200 sp=0xfc11d8 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc1228 sp=0xfc1200 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc1268 sp=0xfc1228 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc1290 sp=0xfc1268 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc12b8 sp=0xfc1290 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc12e8 sp=0xfc12b8 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc1330 sp=0xfc12e8 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc1358 sp=0xfc1330 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc1380 sp=0xfc1358 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc13c0 sp=0xfc1380 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc13e8 sp=0xfc13c0 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc1410 sp=0xfc13e8 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc1440 sp=0xfc1410 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc1488 sp=0xfc1440 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc14b0 sp=0xfc1488 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc14d8 sp=0xfc14b0 pc=0x15fb0002
		runtime.write1(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc1518 sp=0xfc14d8 pc=0x135a0005
		runtime.write(0x2, 0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc1540 sp=0xfc1518 pc=0x16370005
		runtime.writeErrData(0x23aaf6, 0xd)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc1568 sp=0xfc1540 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x23aaf6, 0xd, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc1598 sp=0xfc1568 pc=0x13960010
		runtime.printstring({0x23aaf6, 0xd})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc15e0 sp=0xfc1598 pc=0x13a10006
		runtime.throw.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1091 +0x3 fp=0xfc1608 sp=0xfc15e0 pc=0x13830003
		runtime.throw({0x23c12f, 0xf})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1090 +0x2 fp=0xfc1630 sp=0xfc1608 pc=0x15fb0002
		runtime.write1(0x2, 0x341e70, 0x1)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/os_wasip1.go:168 +0x5 fp=0xfc1670 sp=0xfc1630 pc=0x135a0005
		runtime.write(0x2, 0x341e70, 0x1)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/time_nofake.go:57 +0x5 fp=0xfc1698 sp=0xfc1670 pc=0x16370005
		runtime.writeErrData(0x341e70, 0x1)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/runtime.go:232 +0x2 fp=0xfc16c0 sp=0xfc1698 pc=0x14440002
		runtime.writeErr(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/write_err.go:12
		runtime.gwrite({0x341e70, 0x1, 0x1})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:97 +0x10 fp=0xfc16f0 sp=0xfc16c0 pc=0x13960010
		runtime.printstring({0x341e70, 0x1})
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/print.go:246 +0x6 fp=0xfc1738 sp=0xfc16f0 pc=0x13a10006
		runtime.traceback2(0xfc1a88, 0x0, 0x0, 0x2c)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/traceback.go:999 +0x71 fp=0xfc19a0 sp=0xfc1738 pc=0x14e20071
		runtime.traceback1.func1(0x0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/traceback.go:903 +0x3 fp=0xfc1a60 sp=0xfc19a0 pc=0x14e10003
		runtime.traceback1(0xffffffffffffffff, 0xffffffffffffffff, 0x0, 0x1400700, 0x0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/traceback.go:926 +0x31 fp=0xfc1c60 sp=0xfc1a60 pc=0x14e00031
		runtime.traceback(...)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/traceback.go:803
		runtime.tracebackothers.func1(0x1400700)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/traceback.go:1279 +0x25 fp=0xfc1c98 sp=0xfc1c60 pc=0x14ed0025
		runtime.forEachGRace(0xfc1d00)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/proc.go:720 +0xb fp=0xfc1cc8 sp=0xfc1c98 pc=0x13b7000b
		runtime.tracebackothers(0x1400fc0)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/traceback.go:1265 +0x10 fp=0xfc1d28 sp=0xfc1cc8 pc=0x14ec0010
		runtime.dopanic_m(0x1400fc0, 0x15fb0003, 0x14c1368)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1422 +0x2d fp=0xfc1d78 sp=0xfc1d28 pc=0x138c002d
		runtime.fatalthrow.func1()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/panic.go:1276 +0x3 fp=0xfc1db8 sp=0xfc1d78 pc=0x13880003
		runtime.systemstack(0xfc1dc8)
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/asm_wasm.s:172 +0x3 fp=0xfc1dc0 sp=0xfc1db8 pc=0x16430003
		runtime.mstart()
				/opt/homebrew/Cellar/go/1.24.0/libexec/src/runtime/asm_wasm.s:29 fp=0xfc1dc8 sp=0xfc1dc0 pc=0x163f0000
		Error in runLogs: wasm error: unreachable
		wasm stack trace:
				.runtime.abort(i32) i32
				.runtime.fatalthrow.func1(i32) i32
				.runtime.systemstack(i32) i32
				.runtime.fatalthrow(i32) i32
				.runtime.throw(i32) i32
				.runtime.write1(i32) i32
				.runtime.write(i32) i32
				.runtime.writeErrData(i32) i32
				.runtime.gwrite(i32) i32
				.runtime.printstring(i32) i32
				.runtime.throw.func1(i32) i32
				.runtime.systemstack(i32) i32
				.runtime.throw(i32) i32
				.runtime.write1(i32) i32
				.runtime.write(i32) i32
				.runtime.writeErrData(i32) i32
				.runtime.gwrite(i32) i32
				.runtime.printstring(i32) i32
				.runtime.throw.func1(i32) i32
				.runtime.systemstack(i32) i32
				.runtime.throw(i32) i32
				.runtime.write1(i32) i32
				.runtime.write(i32) i32
				.runtime.writeErrData(i32) i32
				.runtime.gwrite(i32) i32
				.runtime.printstring(i32) i32
				.runtime.throw.func1(i32) i32
				.runtime.systemstack(i32) i32
				.runtime.throw(i32) i32
				.runtime.write1(i32) i32
				... maybe followed by omitted frames
	*/
	// println("adding header")
	// println(w.Header())
	// println(w.Header().Get("Content-Type"))
	// println(textproto.CanonicalMIMEHeaderKey("Content-Type"))
	// h := w.Header()
	// MIMEHeader(h).Add("Content-Type1", "application/json")
	// MIMEHeader(h).Set("Content-Type2", "application/json")
	w.Header().Add("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(healthyResponse))
}

// write response on a failed/bad request. Generates a small json body based on the thrown by
// the handle func and the appropriate http status code. many webhooks will either log these responses or
// notify webhook users should a none 2xx code be detected.
func (er *eventReceiver) failBadReq(_ context.Context,
	w http.ResponseWriter,
	httpStatusCode int,
	err error,
) {
	jsonResp, err := jsoniter.Marshal(err.Error())
	if err != nil {
		// TODO: Enable after supporting settings
		// er.settings.Logger.Warn("failed to marshall error to json")
	}

	// write response to webhook
	w.WriteHeader(httpStatusCode)
	if len(jsonResp) > 0 {
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(jsonResp)
		if err != nil {
			// TODO: Enable after supporting settings
			// er.settings.Logger.Warn("failed to write json response", zap.Error(err))
		}
	}

	// log bad webhook request if debug is enabled
	// TODO: Enable after supporting settings
	// if er.settings.Logger.Core().Enabled(zap.DebugLevel) {
	// 	msg := string(jsonResp)
	// 	er.settings.Logger.Debug(msg, zap.Int("http_status_code", httpStatusCode), zap.Error(err))
	// }
}
