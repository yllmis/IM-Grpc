package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry       *prometheus.Registry
	processed      *prometheus.CounterVec
	processingTime *prometheus.HistogramVec
	throttleWait   prometheus.Histogram
	retries        prometheus.Counter
	deadLetters    prometheus.Counter
	duplicates     prometheus.Counter
	lag            *prometheus.GaugeVec
	lagErrors      prometheus.Counter
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		processed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "messages_processed_total",
			Help: "Number of chat messages handled by task-mq.",
		}, []string{"status"}),
		processingTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "message_processing_seconds",
			Help:    "Time spent handling one chat message after rate limiting.",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 14),
		}, []string{"status"}),
		throttleWait: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "throttle_wait_seconds",
			Help:    "Time messages wait for the application rate limiter.",
			Buckets: prometheus.ExponentialBuckets(0.0005, 2, 15),
		}),
		retries: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "message_retries_total",
			Help: "Number of application-level message retry attempts.",
		}),
		deadLetters: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "dead_letters_total",
			Help: "Number of messages published to the dead-letter topic.",
		}),
		duplicates: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "duplicate_messages_total",
			Help: "Number of duplicate message IDs detected during persistence.",
		}),
		lag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "consumer_lag",
			Help: "Kafka log-end offset minus committed consumer-group offset.",
		}, []string{"topic", "group", "partition"}),
		lagErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "im", Subsystem: "task_mq", Name: "lag_collection_errors_total",
			Help: "Number of failures while collecting Kafka lag.",
		}),
	}
	registry.MustRegister(m.processed, m.processingTime, m.throttleWait, m.retries,
		m.deadLetters, m.duplicates, m.lag, m.lagErrors)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) ObserveProcessing(duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "failed"
	}
	m.processed.WithLabelValues(status).Inc()
	m.processingTime.WithLabelValues(status).Observe(duration.Seconds())
}

func (m *Metrics) ObserveThrottle(duration time.Duration) { m.throttleWait.Observe(duration.Seconds()) }
func (m *Metrics) IncRetry()                              { m.retries.Inc() }
func (m *Metrics) IncDeadLetter()                         { m.deadLetters.Inc() }
func (m *Metrics) IncDuplicate()                          { m.duplicates.Inc() }
func (m *Metrics) IncLagError()                           { m.lagErrors.Inc() }

func (m *Metrics) SetLag(topic, group string, partition int, lag int64) {
	m.lag.WithLabelValues(topic, group, strconv.Itoa(partition)).Set(float64(lag))
}
