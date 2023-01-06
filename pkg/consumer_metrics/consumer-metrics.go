package consumer_metrics

import (
	"context"
	"fmt"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"net/http"
	"time"
)

type eventProcessedEvent struct {
	streamId       string
	eventTimestamp time.Time
	timestamp      time.Time
}

type IndexingServiceMetricsServer struct {
	processedEvents chan eventProcessedEvent
}

func NewConsumerMetricsServer() *IndexingServiceMetricsServer {
	return new(IndexingServiceMetricsServer)
}

func (cm *IndexingServiceMetricsServer) EnableConsumerMetrics(ctx context.Context) *IndexingServiceMetricsServer {
	cm.processedEvents = make(chan eventProcessedEvent, 100)
	processingDelay := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "",
		Name:      "index_processing_delay",
		Help: "How many seconds lasted from last processed event." +
			" If many of streams are processing - the worst one",
	})
	eventsDelay := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "",
		Name:      "index_events_delay",
		Help: "How many seconds lasted from last processed event’s timestamp." +
			" If many of streams are processing - the worst one",
	})
	messagesPerSec := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "",
		Name:      "index_processing_speed",
		Help:      "This metric tells how many messages are processed per second",
	})
	prometheus.MustRegister(processingDelay)
	prometheus.MustRegister(eventsDelay)
	prometheus.MustRegister(messagesPerSec)

	go func() {
		type streamData struct {
			lastEventGotTime      time.Time
			lastEventTimestamp    time.Time
			eventsSinceLastUpdate int64
		}

		streams := make(map[string]*streamData)

		lastUpdateTime := time.Now()

		t := time.NewTicker(1 * time.Second)
		for ctx.Err() == nil {
			select {
			case <-t.C:
				lastEventGotTime := time.Now()
				lastEventTimestamp := time.Now()
				eventsSinceLastUpdate := int64(1e18)
				for _, data := range streams {
					if lastEventGotTime.After(data.lastEventGotTime) {
						lastEventGotTime = data.lastEventGotTime
					}
					if lastEventTimestamp.After(data.lastEventTimestamp) {
						lastEventTimestamp = data.lastEventTimestamp
					}
					if eventsSinceLastUpdate > data.eventsSinceLastUpdate {
						eventsSinceLastUpdate = data.eventsSinceLastUpdate
					}
					data.eventsSinceLastUpdate = 0
				}

				processingDelay.Set(time.Since(lastEventGotTime).Seconds())
				eventsDelay.Set(time.Since(lastEventTimestamp).Seconds())
				messagesPerSec.Set(
					1000 * float64(eventsSinceLastUpdate) / float64(time.Since(lastUpdateTime).Milliseconds()))

				lastUpdateTime = time.Now()

			case event := <-cm.processedEvents:
				if streams[event.streamId] == nil {
					streams[event.streamId] = new(streamData)
				}
				streams[event.streamId].eventsSinceLastUpdate++
				streams[event.streamId].lastEventTimestamp = event.eventTimestamp
				streams[event.streamId].lastEventGotTime = event.timestamp

			case <-ctx.Done():
				return
			}
		}
	}()
	return cm
}

func (cm *IndexingServiceMetricsServer) EnableServerMetrics(server *grpc.Server) *IndexingServiceMetricsServer {
	grpcPrometheus.EnableHandlingTimeHistogram()
	grpcPrometheus.Register(server)
	return cm
}

func (cm *IndexingServiceMetricsServer) Start(port int) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (cm *IndexingServiceMetricsServer) EventProcessed(stream string, timestamp time.Time) {
	if cm.processedEvents == nil {
		panic("cannot use consumer metrics with server-only IndexingServiceMetricsServer")
	}
	cm.processedEvents <- eventProcessedEvent{
		streamId:       stream,
		eventTimestamp: timestamp,
		timestamp:      time.Now(),
	}
}
