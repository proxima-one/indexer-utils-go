package main

import (
	"context"
	"github.com/proxima-one/indexer-utils-go/v2/pkg/consume_status"
	"github.com/proxima-one/indexer-utils-go/v2/pkg/prometheus_metrics"
	"time"
)

func main() {
	go func() {
		serv := consume_status.NewConsumeStatusServer()
		serv.Start(context.Background(), 27000, 8080)
		serv.RegisterStream("eth-main-0", "eth-main")
		serv.RegisterStream("eth-main-1", "eth-main")
		serv.RegisterStream("polygon-mumbai-0", "polygon-mumbai")
		serv.RegisterStream("polygon-mumbai-1", "polygon-mumbai")

		serv.UpdateStreamStatus("eth-main-0", time.Unix(time.Now().Unix()-1000000, 0), "100")
		serv.UpdateStreamStatus("eth-main-1", time.Unix(time.Now().Unix(), 0), "200")
		serv.UpdateStreamStatus("polygon-mumbai-0", time.Unix(time.Now().Unix()-1200000, 0), "110")
		serv.UpdateStreamStatus("polygon-mumbai-1", time.Unix(time.Now().Unix()-100, 0), "250")
		for {
			time.Sleep(time.Second)
		}
	}()
	go func() {
		metrics := prometheus_metrics.NewPrometheusMetricsServer().EnableConsumerMetrics(context.Background())
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
