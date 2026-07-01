package daemon

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

//go:embed noxie-sort.service
var Service []byte

var dlog = slog.With("module", "daemon")

type ServiceInfo struct {
	Name    string
	Path    string
	Content string
}

type Status struct {
	Condition string `json:"condition"`
	Name      string `json:"name"`
	Path      string `json:"path"`
}

func (s *ServiceInfo) LaunchingDaemon() error {
	dlog.Info("starting daemon installation")
	err := IsWorking()
	if err == nil {
		return errors.New("daemon is already running")
	}

	err = s.initDaemon()
	if err != nil {
		return err
	}

	return nil
}

func (s *ServiceInfo) OpenServer(ctx context.Context) {
	dlog.Info("opening server", "port", ":9999")
	srv := &http.Server{
		Addr:         ":9999",
		BaseContext:  func(net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(s),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()
}

func newHTTPHandler(s *ServiceInfo) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/status", http.HandlerFunc(s.Connection))
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}

func NewService() (*ServiceInfo, error) {
	service := &ServiceInfo{
		Name:    "",
		Path:    "",
		Content: "",
	}
	err := service.initService()
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (s *ServiceInfo) Connection(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("daemon service")
	_, span := tracer.Start(r.Context(), "Status")
	defer span.End()

	status := &Status{
		Condition: "Active",
		Name:      s.Name,
		Path:      s.Path,
	}

	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
