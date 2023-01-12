package consume_status

import (
	"context"
	"fmt"
	pb "github.com/proxima-one/indexer-utils-go/v2/pkg/consume_status/internal/proto"
	"github.com/proxima-one/indexer-utils-go/v2/pkg/grpc_gateway"
	"github.com/proxima-one/indexer-utils-go/v2/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"log"
	"net"
	"sync"
	"time"
)

type indexingStatus struct {
	Timestamp   time.Time
	BlockNumber string
}

type ConsumeStatusServer struct {
	mu                sync.RWMutex
	statusByStreamId  map[string]indexingStatus
	networkByStreamId map[string]string
}

func NewConsumeStatusServer() *ConsumeStatusServer {
	return &ConsumeStatusServer{
		statusByStreamId:  make(map[string]indexingStatus),
		networkByStreamId: make(map[string]string),
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

func (s *ConsumeStatusServer) RegisterStream(streamId, network string) {
	s.mu.Lock()
	s.networkByStreamId[streamId] = network
	s.mu.Unlock()
}

// UpdateStreamStatus updates stream status of a stream. Stream must be already registered
func (s *ConsumeStatusServer) UpdateStreamStatus(streamId string, timestamp time.Time, blockNumber string) {
	s.mu.RLock()
	if _, ok := s.networkByStreamId[streamId]; !ok {
		s.mu.RUnlock()
		utils.PanicOnError(fmt.Errorf("stream %s is not registered", streamId))
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusByStreamId[streamId] = indexingStatus{
		Timestamp:   timestamp,
		BlockNumber: blockNumber,
	}
}

func (s *ConsumeStatusServer) GetStatus(_ context.Context, _ *emptypb.Empty) (*pb.GetStatusResponse, error) {
	s.mu.RLock()
	networkStatuses := make(map[string]*indexingStatus)
	for streamId, streamStatus := range s.statusByStreamId {
		network := s.networkByStreamId[streamId]
		if _, ok := networkStatuses[network]; !ok {
			networkStatuses[network] = &indexingStatus{
				Timestamp:   streamStatus.Timestamp,
				BlockNumber: streamStatus.BlockNumber,
			}
		} else {
			networkStatus := networkStatuses[network]
			if utils.MustConvStrToInt64(streamStatus.BlockNumber) < utils.MustConvStrToInt64(networkStatus.BlockNumber) {
				networkStatus.Timestamp = streamStatus.Timestamp
				networkStatus.BlockNumber = streamStatus.BlockNumber
			}
		}
	}
	s.mu.RUnlock()

	res := &pb.GetStatusResponse{Networks: make([]*pb.NetworkIndexingStatus, 0)}
	for network, status := range networkStatuses {
		res.Networks = append(res.Networks, &pb.NetworkIndexingStatus{
			Network: network,
			Status: &pb.IndexingStatus{
				Timestamp:   timestamppb.New(status.Timestamp),
				BlockNumber: &status.BlockNumber,
			},
		})
	}
	return res, nil
}
