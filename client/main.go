package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Hanekawa-chan/chat/protoc"
	"io"
	"log"
	"os"

	"google.golang.org/grpc"
)

var tcpServer = flag.String("server", ":5400", "Tcp server")
var channelName, senderName string

func joinChannel(ctx context.Context, client protoc.ChatServiceClient, channel *protoc.Channel) {

	stream, err := client.JoinChannel(ctx, channel)
	if err != nil {
		log.Fatalf("client.JoinChannel(ctx, &channel) throws: %v", err)
	}

	fmt.Printf("Joined channel: %v \n", channel.Name)

	waitc := make(chan struct{})

	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive message from channel joining. \nErr: %v", err)
			}

			if channel.SendersName != in.Sender {
				fmt.Printf("MESSAGE: (%v) -> %v \n", in.Sender, in.Message)
			}
		}
	}()

	<-waitc
}

func sendMessage(ctx context.Context, client protoc.ChatServiceClient, message string) {
	stream, err := client.SendMessage(ctx)
	if err != nil {
		log.Printf("Cannot send message: error: %v", err)
	}
	msg := protoc.Message{
		Channel: &protoc.Channel{
			Name:        channelName,
			SendersName: senderName},
		Message: message,
		Sender:  senderName,
	}
	err = stream.Send(&msg)
	if err != nil {
		return
	}

	ack, err := stream.CloseAndRecv()
	fmt.Printf("Message sent: %v \n", ack)
}

func main() {

	flag.Parse()

	fmt.Println("--- CLIENT APP ---")
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithInsecure())

	conn, err := grpc.Dial(*tcpServer, opts...)
	if err != nil {
		log.Fatalf("Fail to dail: %v", err)
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)

	ctx := context.Background()
	client := protoc.NewChatServiceClient(conn)

	scanner := bufio.NewScanner(os.Stdin)
	channel := protoc.Channel{}

	for {
		if channelName != "" {
			for scanner.Scan() {
				if scanner.Text() == "/" {
					channelName = ""
					senderName = ""
					break
				} else {
					go sendMessage(ctx, client, scanner.Text())
				}
			}
		} else {
			fmt.Println("Enter channel name")
			for scanner.Scan() {
				channel.Name = scanner.Text()
				break
			}
			fmt.Println("Enter your name")
			for scanner.Scan() {
				channel.SendersName = scanner.Text()
				break
			}
			fmt.Println("Enter password for channel")
			for scanner.Scan() {
				channel.Password = scanner.Text()
				break
			}
			channelName = channel.Name
			senderName = channel.SendersName
			go joinChannel(ctx, client, &channel)
		}
	}
}
