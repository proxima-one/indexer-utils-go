package grpc_metrics

import (
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"net/http"
)

func StartGrpcMetrics(serv *grpc.Server, port string) error {
	grpcPrometheus.EnableHandlingTimeHistogram()
	grpcPrometheus.Register(serv)
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+port, nil)
}
