package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	pb "pubsub/protoc"
)

const (
	address     = "localhost:9900"
)

func publish(client pb.PubSubClient, request *pb.PublishRequest)  {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := client.Publish(ctx, request)
	if err != nil {
		log.Fatalf("publish fail: %v", err)
	}
	log.Printf("published: %s", r.Result)
}

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	var arr2 = []*pb.Message{{Payload:"{io:23434,la:43543}",DataType:"1"}}

	client := pb.NewPubSubClient(conn)
	publish(client, &pb.PublishRequest{Topic:"a|b|c|d", Messages:arr2})
	//publish(client, &pb.PublishRequest{Topic:"bbb", Messages:arr2})

}