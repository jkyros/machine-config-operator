package common

import (
	"context"
	"net/http"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// DefaultBindAddress is the port for the metrics listener
	DefaultBindAddress = ":8797"

	// MCCImportantConfigPaused logs when important config is being held up by pause
	MCCImportantConfigPaused = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "machine_config_controller_important_config_paused",
			Help: "important config is waiting in specified paused pool",
			// "platform config kubelet ca certificate is stuck in paused pool worker"
		}, []string{"pool"})

	metricsList = []prometheus.Collector{
		MCCImportantConfigPaused,
	}
)

func registerMCCMetrics() error {
	for _, metric := range metricsList {
		err := prometheus.Register(metric)
		if err != nil {
			return err
		}
	}

	// Initialize to default of config is not currently paused
	MCCImportantConfigPaused.WithLabelValues("").Set(0)
	return nil
}

// StartMetricsListener is metrics listener via http on localhost
func StartMetricsListener(addr string, stopCh <-chan struct{}) {
	if addr == "" {
		addr = DefaultBindAddress
	}

	glog.Info("Registering Prometheus metrics")
	if err := registerMCCMetrics(); err != nil {
		glog.Errorf("unable to register metrics: %v", err)
	}

	glog.Infof("Starting metrics listener on %s", addr)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	s := http.Server{Addr: addr, Handler: mux}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			glog.Errorf("metrics listener exited with error: %v", err)
		}
	}()
	<-stopCh
	if err := s.Shutdown(context.Background()); err != http.ErrServerClosed {
		glog.Errorf("error stopping metrics listener: %v", err)
	}
}
