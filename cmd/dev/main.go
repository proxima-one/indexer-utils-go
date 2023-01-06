package main

import (
	"context"
	"github.com/proxima-one/indexer-utils-go/pkg/consume_status"
	"github.com/proxima-one/indexer-utils-go/pkg/prometheus_metrics"
	"time"
)

func main() {
	go func() {
		serv := consume_status.NewConsumeStatusServer()
		serv.Start(context.Background(), 27000, 8080)
		serv.UpdateNetworkIndexingStatus("test", time.Now(), "1")
		for {
			time.Sleep(time.Second)
		}
	}()
	go func() {
		metrics := prometheus_metrics.NewConsumerMetricsServer().EnableConsumerMetrics(context.Background())
		go metrics.Start(12228)
		for {
			metrics.EventProcessed("net1", time.Unix(time.Now().Unix()-100, 0))
			time.Sleep(700 * time.Millisecond)
			metrics.EventProcessed("net2", time.Unix(time.Now().Unix()-200, 0))
			time.Sleep(10 * time.Millisecond)
		}
	}()
	time.Sleep(1e18)
}
