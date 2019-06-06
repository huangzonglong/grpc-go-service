package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"log"
	"net"
	pb "pubsub/protoc"
	"regexp"
	"strings"
	"sync"
	"time"
)

type PubSubServer struct{}

type StreamPool struct {
	sync.Map
}

var poolClientIDStreams *StreamPool
var poolTopicClientID sync.Map
var clientIds sync.Map

func (s *PubSubServer) Subscribe(ctx context.Context, request *pb.SubscribeRequest) (*pb.Subscription, error) {
	if request.Identity.ClientId == ""{
		return nil,status.Errorf(3, "clientId not allow nil")
	}

	if err := HandelClientIdTopic(request.Subscription.Topic, request.Identity.ClientId); err != nil {
		return nil, err
	}

	return &pb.Subscription{Topic:request.Subscription.Topic},nil
}

func (s *PubSubServer) Unsubscribe(ctx context.Context, request *pb.SubscribeRequest) (*pb.Subscription, error) {
	poolClientIDStreams.Delete(request.Identity.ClientId)
	poolTopicClientID.Delete(request.Subscription.Topic+ "..." +request.Identity.ClientId)
	clientIds.Delete(request.Identity.ClientId)
	return &pb.Subscription{Topic:request.Subscription.Topic}, nil
}

func (s *PubSubServer) Publish(ctx context.Context, request *pb.PublishRequest) (*pb.PublishResponse, error) {
	if err := HandelClientIdTopic(request.Topic, ""); err != nil {
		return nil, err
	}

	go poolClientIDStreams.Brocast(request)

	return &pb.PublishResponse{Result:"ok"},nil
}

func (s *PubSubServer) Pull(request *pb.Identity, stream pb.PubSub_PullServer) error {
	if _, exist := clientIds.Load(request.ClientId); exist {
		return status.Errorf(6, "clientId already existed")
	}

	clientIds.Store(request.ClientId, time.Now())

	poolClientIDStreams.Store(request.ClientId, stream)

	//客户端退出时
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			poolClientIDStreams.Delete(request.ClientId)
			clientIds.Delete(request.ClientId)
			poolTopicClientID.Range(func(tc, clientId interface{}) bool {
				if strings.HasSuffix(tc.(string), request.ClientId) {
					poolTopicClientID.Delete(tc)
				}
				return true
			})

			return ctx.Err()
		default:
			time.Sleep(2 *time.Second)
		_:stream.Send(&pb.Message{Topic:"ping"})
		}
	}
}

//send消息
func (p *StreamPool) Brocast(request *pb.PublishRequest) {
	log.Println(request.Topic)
	poolTopicClientID.Range(func(tc, clientId interface{}) bool {
		recTopicArr := strings.Split(tc.(string), "...")
		reqTopicArr := strings.Split(request.Topic, "|")
		newTopic := strings.Replace(request.Topic, reqTopicArr[3], "+", 1 )//单层比配模式订阅的

		if recTopicArr[0] == request.Topic || recTopicArr[0] == newTopic {
			stream := p.GetStreamInPoll(clientId.(string))
			if stream != nil {
				for _, message := range request.Messages {
				_:stream.Send(&pb.Message{Topic: request.Topic, Payload:message.Payload, DataType:message.DataType})
				}
			}
		}

		return true
	})
}

//检测订阅和发布的topic正确行
func HandelClientIdTopic(Topic string, ClientId string) error {
	//log.Println(Topic,clientIds)
	if Topic == "" {
		return status.Errorf(3, "clientId and topic not allowed nil")
	}

	reg, _ := regexp.Compile("^([a-zA-Z0-9|])+([+]?)$")

	arr := strings.Split(Topic, "|")
	if len(arr) != 4 || strings.HasPrefix(Topic, "|") || strings.HasSuffix(Topic, "|") || !reg.MatchString(Topic) {
		return status.Errorf(3, "incorrect format of topic")
	}

	if ClientId != ""{//sub
		reg, _ := regexp.Compile("^[a-zA-Z0-9_\\-.]+$")
		if !reg.MatchString(ClientId){
			return status.Errorf(3, "incorrect format of ClientId")
		}

		poolTopicClientID.Store(Topic + "..." + ClientId, ClientId)
	}

	return nil
}

func (p *StreamPool) GetStreamInPoll(name string) pb.PubSub_PullServer {
	if stream, ok := p.Load(name); ok {
		return stream.(pb.PubSub_PullServer)
	} else {
		return nil
	}
}

func (s *PubSubServer) Start(port string) {
	var opts []grpc.ServerOption

	lis, err := net.Listen("tcp", ":" + port)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("Start Server :%s", port)

	grpcServer := grpc.NewServer(opts...)
	pb.RegisterPubSubServer(grpcServer, &PubSubServer{})
	grpcServer.Serve(lis)
}

func main() {
	poolClientIDStreams = &StreamPool{}
	pbserv := new(PubSubServer)
	pbserv.Start("9900")
}

