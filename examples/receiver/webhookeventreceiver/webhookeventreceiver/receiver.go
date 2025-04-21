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
	"github.com/rs/cors"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"

	wasinet "github.com/musaprg/otelwasm/examples/receiver/webhookeventreceiver/wasip1"
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
	handler = httpContentDecompressor(
		handler,
		er.cfg.MaxRequestBodySize,
		nil,
		er.cfg.CompressionAlgorithms,
		nil,
	)

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
