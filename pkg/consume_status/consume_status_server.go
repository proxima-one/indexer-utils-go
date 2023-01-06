package consume_status

import (
	"context"
	"fmt"
	pb "github.com/proxima-one/indexer-utils-go/pkg/consume_status/internal/proto"
	"github.com/proxima-one/indexer-utils-go/pkg/grpc_gateway"
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

type ConsumeStatusServer struct {
	states map[string]*networkIndexingStatus
}

func NewConsumeStatusServer() *ConsumeStatusServer {
	return &ConsumeStatusServer{
		states: make(map[string]*networkIndexingStatus),
	}
}

func (s *ConsumeStatusServer) Start(ctx context.Context, grpcPort, httpPort int) {
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

func (s *ConsumeStatusServer) UpdateNetworkIndexingStatus(network string, timestamp time.Time, blockNumber string) {
	if _, ok := s.states[network]; !ok {
		s.states[network] = &networkIndexingStatus{}
	}
	s.states[network].Network = network
	s.states[network].Timestamp = timestamp
	s.states[network].BlockNumber = blockNumber
}

func (s *ConsumeStatusServer) GetStatus(_ context.Context, _ *emptypb.Empty) (*pb.GetStatusResponse, error) {
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
