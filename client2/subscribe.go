package main

import (
	"context"
	"google.golang.org/grpc"
	"io"
	"log"
	pb "pubsub/protoc"
	"time"
)

const (
	address     = "localhost:9900"
)

func subscibe(client pb.PubSubClient, req *pb.SubscribeRequest)  {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, err := client.Subscribe(ctx, req)
	if err != nil {
		log.Fatalf("subscibe fail: %v", err.Error())
	}
	log.Printf("subscibe: %s", r.Topic)
}


func unsubscibe(client pb.PubSubClient, req *pb.SubscribeRequest)  {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, err := client.Unsubscribe(ctx, req)
	if err != nil {
		log.Fatalf("unsubscibe fail: %v", err)
	}
	log.Printf("unsubscibe: %s", r.Topic)
}


func pull(client pb.PubSubClient, clientID string) {
	ctx := context.Background()
	stream, err := client.Pull(ctx, &pb.Identity{ClientId:clientID})
	if err != nil {
		log.Fatalf("%v.ListFeatures(_) = _, %v", client, err)
	}

	for {
		reply, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		if reply.Topic != "ping" {
			log.Printf("message：%s", reply.Payload)
		}
	}
}

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	var clientID = "clinetID1"
	var Topic = "a|b|c|d"
	client := pb.NewPubSubClient(conn)
	subscibe(client, &pb.SubscribeRequest{Identity:&pb.Identity{ClientId:clientID}, Subscription: &pb.Subscription{Topic:Topic}})

	//Topic = "a|b|c|22222222222"
	//subscibe(client, &pb.SubscribeRequest{Identity:&pb.Identity{ClientId:clientID}, Subscription: &pb.Subscription{Topic:Topic}})
	////unsubscibe(client, &pb.SubscribeRequest{Identity:&pb.Identity{ClientId:clientID}, Subscription: &pb.Subscription{Topic:Topic}})

	pull(client, clientID)
}