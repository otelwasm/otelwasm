// Command server is a minimal OpAMP server for demonstrating remote
// management of otelwasmcol via the OpAMP protocol.
//
// It serves the collector configuration from a local file (see -config) as
// an OpAMP remote config. The file is polled for changes; when it changes,
// the new configuration is pushed to all connected agents, which lets you
// reconfigure a running otelwasmcol (e.g. swap or reconfigure WASM plugins)
// by simply editing the file.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server"
	"github.com/open-telemetry/opamp-go/server/types"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:4320", "address to listen on for OpAMP connections")
	configPath := flag.String("config", "remote-config.yaml", "path to the collector config to serve as OpAMP remote config")
	flag.Parse()

	logger := log.New(os.Stderr, "opamp-server: ", log.LstdFlags)

	srv := &opampServer{
		logger:     logger,
		configPath: *configPath,
		agents:     make(map[types.Connection][]byte),
	}
	if _, err := srv.loadConfig(); err != nil {
		logger.Fatalf("failed to load %s: %v", *configPath, err)
	}

	opamp := server.New(&opampLogger{logger})
	err := opamp.Start(server.StartSettings{
		Settings: server.Settings{
			Callbacks: types.Callbacks{
				OnConnecting: func(*http.Request) types.ConnectionResponse {
					return types.ConnectionResponse{
						Accept: true,
						ConnectionCallbacks: types.ConnectionCallbacks{
							OnConnected:       srv.onConnected,
							OnMessage:         srv.onMessage,
							OnConnectionClose: srv.onConnectionClose,
						},
					}
				},
			},
		},
		ListenEndpoint: *listen,
		ListenPath:     "/v1/opamp",
	})
	if err != nil {
		logger.Fatalf("failed to start OpAMP server: %v", err)
	}
	logger.Printf("listening on ws://%s/v1/opamp, serving remote config from %s", *listen, *configPath)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv.watchConfig(ctx)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := opamp.Stop(shutdownCtx); err != nil {
		logger.Printf("failed to stop OpAMP server: %v", err)
	}
}

type opampServer struct {
	logger     *log.Logger
	configPath string

	mu         sync.Mutex
	configBody []byte
	configHash []byte
	agents     map[types.Connection][]byte // connection -> instance UID
}

// loadConfig reads the config file and updates the current remote config.
// It reports whether the config changed.
func (s *opampServer) loadConfig() (bool, error) {
	body, err := os.ReadFile(s.configPath)
	if err != nil {
		return false, err
	}
	hash := sha256.Sum256(body)

	s.mu.Lock()
	defer s.mu.Unlock()
	if bytes.Equal(hash[:], s.configHash) {
		return false, nil
	}
	s.configBody = body
	s.configHash = hash[:]
	return true, nil
}

func (s *opampServer) remoteConfig() *protobufs.AgentRemoteConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &protobufs.AgentRemoteConfig{
		Config: &protobufs.AgentConfigMap{
			ConfigMap: map[string]*protobufs.AgentConfigFile{
				"": {Body: s.configBody, ContentType: "text/yaml"},
			},
		},
		ConfigHash: s.configHash,
	}
}

// watchConfig polls the config file and pushes the new config to all
// connected agents when it changes. Blocks until ctx is done.
func (s *opampServer) watchConfig(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := s.loadConfig()
			if err != nil {
				s.logger.Printf("failed to reload %s: %v", s.configPath, err)
				continue
			}
			if !changed {
				continue
			}
			s.logger.Printf("config %s changed, pushing to connected agents", s.configPath)
			cfg := s.remoteConfig()

			s.mu.Lock()
			agents := make(map[types.Connection][]byte, len(s.agents))
			for conn, uid := range s.agents {
				agents[conn] = uid
			}
			s.mu.Unlock()

			for conn, uid := range agents {
				err := conn.Send(ctx, &protobufs.ServerToAgent{
					InstanceUid:  uid,
					RemoteConfig: cfg,
				})
				if err != nil {
					s.logger.Printf("failed to push config to agent %x: %v", uid, err)
				}
			}
		}
	}
}

func (s *opampServer) onConnected(_ context.Context, conn types.Connection) {
	s.logger.Printf("agent connected: %s", conn.Connection().RemoteAddr())
}

func (s *opampServer) onConnectionClose(conn types.Connection) {
	s.mu.Lock()
	uid := s.agents[conn]
	delete(s.agents, conn)
	s.mu.Unlock()
	s.logger.Printf("agent disconnected: %x", uid)
}

func (s *opampServer) onMessage(_ context.Context, conn types.Connection, msg *protobufs.AgentToServer) *protobufs.ServerToAgent {
	s.mu.Lock()
	s.agents[conn] = msg.InstanceUid
	s.mu.Unlock()

	if d := msg.AgentDescription; d != nil {
		var attrs []string
		for _, kv := range d.IdentifyingAttributes {
			attrs = append(attrs, fmt.Sprintf("%s=%s", kv.Key, kv.Value.GetStringValue()))
		}
		s.logger.Printf("agent %x described itself: %v", msg.InstanceUid, attrs)
	}
	if h := msg.Health; h != nil {
		s.logger.Printf("agent %x health: healthy=%v status=%q", msg.InstanceUid, h.Healthy, h.Status)
	}
	if st := msg.RemoteConfigStatus; st != nil {
		s.logger.Printf("agent %x remote config status: %s (hash=%x) %s",
			msg.InstanceUid, st.Status, st.LastRemoteConfigHash, st.ErrorMessage)
	}

	resp := &protobufs.ServerToAgent{
		InstanceUid: msg.InstanceUid,
		Capabilities: uint64(protobufs.ServerCapabilities_ServerCapabilities_AcceptsStatus |
			protobufs.ServerCapabilities_ServerCapabilities_OffersRemoteConfig),
	}

	// Offer the current remote config unless the agent already has it.
	cfg := s.remoteConfig()
	if msg.RemoteConfigStatus == nil || !bytes.Equal(msg.RemoteConfigStatus.LastRemoteConfigHash, cfg.ConfigHash) {
		s.logger.Printf("offering remote config (hash=%x) to agent %x", cfg.ConfigHash, msg.InstanceUid)
		resp.RemoteConfig = cfg
	}
	return resp
}

// opampLogger adapts log.Logger to the opamp-go logger interface.
type opampLogger struct {
	logger *log.Logger
}

func (l *opampLogger) Debugf(_ context.Context, format string, v ...interface{}) {
	l.logger.Printf("DEBUG: "+format, v...)
}

func (l *opampLogger) Errorf(_ context.Context, format string, v ...interface{}) {
	l.logger.Printf("ERROR: "+format, v...)
}
