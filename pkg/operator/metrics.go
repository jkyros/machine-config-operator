package operator

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	DefaultBindAddress = ":8797"
)

// All of the Metrics
var (
	// mcoState is the state of the machine config operator
	// pause, updated, updating, degraded
	mcoState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mco_state",
			Help: "state of operator on specified node",
		}, []string{"state", "reason"})

	mcoMachineCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mco_machine_count",
			Help: "total number of machines on specified node",
		}, []string{"pool"})

	mcoUpdatedMachineCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mco_updated_machine_count",
			Help: "total number of machines on specified node",
		}, []string{"pool"})

	mcoDegradedMachineCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mco_degraded_machine_count",
			Help: "total number of machines on specified node",
		}, []string{"pool"})

	allMetrics = []prometheus.Collector{
		mcoState, mcoMachineCount, mcoUpdatedMachineCount, mcoDegradedMachineCount,
	}
)

func RegisterMCOMetrics() error {
	for _, metric := range allMetrics {
		err := prometheus.Register(metric)
		if err != nil {
			return fmt.Errorf("could not register metric: %v", err)
		}
	}

	return nil
}
