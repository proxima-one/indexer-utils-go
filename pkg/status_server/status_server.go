package status_server

import (
	"context"
	"fmt"
	"github.com/proxima-one/indexer-utils-go/pkg/grpc_gateway"
	pb "github.com/proxima-one/indexer-utils-go/pkg/status_server/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net"
	"time"
)

type networkIndexingStatus struct {
	Network     string
	Timestamp   time.Time
	BlockNumber string
}

type StatusServer struct {
	states map[string]*networkIndexingStatus
}

func NewStatusServer() *StatusServer {
	return &StatusServer{
		states: make(map[string]*networkIndexingStatus),
	}
}

func (s *StatusServer) Start(ctx context.Context, grpcPort, httpPort int) {
	grpcServer := grpc.NewServer()
	pb.RegisterStatusServiceServer(grpcServer, s)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Println(grpcServer.Serve(lis).Error())
	}()
	go func() {
		log.Println(grpc_gateway.Run(
			ctx,
			fmt.Sprintf("localhost:%d", grpcPort),
			httpPort,
			pb.ProtoDir,
			"indexing_status.swagger.json",
			pb.RegisterStatusServiceHandler,
		).Error())
	}()
}

func (s *StatusServer) UpdateNetworkIndexingStatus(network string, timestamp time.Time, blockNumber string) {
	if _, ok := s.states[network]; !ok {
		s.states[network] = &networkIndexingStatus{}
	}
	s.states[network].Network = network
	s.states[network].Timestamp = timestamp
	s.states[network].BlockNumber = blockNumber
}

func (s *StatusServer) GetStatus(_ context.Context, _ *emptypb.Empty) (*pb.GetStatusResponse, error) {
	res := &pb.GetStatusResponse{Networks: make([]*pb.NetworkIndexingStatus, 0)}

	for _, state := range s.states {
		res.Networks = append(res.Networks, &pb.NetworkIndexingStatus{
			Network: state.Network,
			Status: &pb.IndexingStatus{
				Timestamp:   timestamppb.New(state.Timestamp),
				BlockNumber: &state.BlockNumber,
			},
		})
	}
	return res, nil
}
