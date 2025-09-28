package prometheus

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.22.0"
)

// https://www.cncf.io/blog/2025/07/22/prometheus-labels-understanding-and-best-practices/

var (
	profilesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lazybackend_received_profiles_total",
		Help: "The total number of received profiles",
	}, []string{"container_id"})
	samplesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lazybackend_received_samples_total",
		Help: "The total number of received samples",
	}, []string{"container_id"})
	locationsReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "lazybackend_received_locations_total",
		Help: "The total number of received locations",
	}, []string{"container_id", "language"})
)

type Prometheus struct {
}

func NewReceiver() *Prometheus {
	return &Prometheus{}
}

func (r *Prometheus) Receive(ctx context.Context, pd pprofile.Profiles) error {
	locationTable := pd.Dictionary().LocationTable()
	attributeTable := pd.Dictionary().AttributeTable()
	stringTable := pd.Dictionary().StringTable()

	for rpi := 0; rpi < pd.ResourceProfiles().Len(); rpi++ {
		rp := pd.ResourceProfiles().At(rpi)

		containerID := "unknown"

		rp.Resource().Attributes().Range(func(key string, value pcommon.Value) bool {
			if key == string(semconv.ContainerIDKey) {
				containerIDValue := value.AsString()
				if containerIDValue != "" {
					containerID = containerIDValue
				}
			}
			return true
		})

		// Count profiles that we are receiving
		profilesReceived.WithLabelValues(containerID).Add(float64(rp.ScopeProfiles().Len()))
		for spi := 0; spi < rp.ScopeProfiles().Len(); spi++ {
			sp := rp.ScopeProfiles().At(spi)
			for pi := 0; pi < sp.Profiles().Len(); pi++ {
				p := sp.Profiles().At(pi)
				// Count samples (stack traces) that we are receiving
				samplesReceived.WithLabelValues(containerID).Add(float64(p.Sample().Len()))

				for sampleIdx := 0; sampleIdx < p.Sample().Len(); sampleIdx++ {
					s := p.Sample().At(sampleIdx)

					sampleLocationIndices := pd.Dictionary().StackTable().At(int(s.StackIndex())).LocationIndices()

					for m := 0; m < sampleLocationIndices.Len(); m++ {
						location := locationTable.At(int(sampleLocationIndices.At(int(m))))
						locationAttrs := location.AttributeIndices()

						frameType := "unknown"
						for la := 0; la < locationAttrs.Len(); la++ {
							attr := attributeTable.At(int(locationAttrs.At(la)))
							if stringTable.At(int(attr.KeyStrindex())) == "profile.frame.type" {
								frameType = attr.Value().AsString()
								break
							}
						}

						locationLine := location.Line()
						if locationLine.Len() == 0 {
							locationsReceived.WithLabelValues(containerID, frameType).Inc()
						}

						locationsReceived.WithLabelValues(containerID, frameType).Add(float64(locationLine.Len()))
					}
				}
			}
		}
	}

	return nil
}
