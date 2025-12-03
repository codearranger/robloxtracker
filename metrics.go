package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	UserPresenceType *prometheus.GaugeVec
}

func robloxMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		UserPresenceType: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "UserPresenceType",
			Help: "Offline, Online, InGame, InStudio, Unknown",
		}, []string{"userid"}),
	}

	reg.MustRegister(m.UserPresenceType)

	return m
}
