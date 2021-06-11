package main

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"sync"

	"github.com/Hanekawa-chan/chat/protoc"
	"google.golang.org/grpc"
)

type chatServiceServer struct {
	protoc.UnimplementedChatServiceServer
	mu       sync.Mutex
	channel  map[string][]chan *protoc.Message
	password map[string]string
}

func (s *chatServiceServer) JoinChannel(ch *protoc.Channel, msgStream protoc.ChatService_JoinChannelServer) error {

	msgChannel := make(chan *protoc.Message)
	password := ch.Password
	if s.channel[ch.Name] != nil {
		if s.password[ch.Name] == password {
			s.channel[ch.Name] = append(s.channel[ch.Name], msgChannel)
		} else {
			return status.Errorf(codes.Unauthenticated, "wrong password")
		}
	} else {
		s.channel[ch.Name] = append(s.channel[ch.Name], msgChannel)
		s.password[ch.Name] = password
	}

	for {
		select {
		case <-msgStream.Context().Done():
			return nil
		case msg := <-msgChannel:
			fmt.Printf("GO ROUTINE (got message): %v \n", msg)
			msgStream.Send(msg)
		}
	}
}

func (s *chatServiceServer) SendMessage(msgStream protoc.ChatService_SendMessageServer) error {
	msg, err := msgStream.Recv()

	if err == io.EOF {
		return nil
	}

	if err != nil {
		return err
	}

	ack := protoc.MessageAck{Status: "SENT"}
	err = msgStream.SendAndClose(&ack)
	if err != nil {
		return err
	}

	go func() {
		streams := s.channel[msg.Channel.Name]
		for _, msgChan := range streams {
			msgChan <- msg
		}
	}()

	return nil
}

func newServer() *chatServiceServer {
	s := &chatServiceServer{
		channel: make(map[string][]chan *protoc.Message),
	}
	fmt.Println(s)
	return s
}

func main() {
	fmt.Println("--- SERVER APP ---")
	lis, err := net.Listen("tcp", "localhost:5400")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	protoc.RegisterChatServiceServer(grpcServer, newServer())
	err = grpcServer.Serve(lis)
	if err != nil {
		return
	}
}
