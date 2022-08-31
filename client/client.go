package main

//A simple client to test the server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	pb "therealbroker/api/proto"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

var concurrent bool

// var ids = make(chan int32, 5)
// var duration = time.Hour

var err error

func singlePublish(client pb.BrokerClient, subject string) {
	_, err = client.Publish(context.Background(), &pb.PublishRequest{
		Subject:           subject,
		Body:              []byte(randomString(16)),
		ExpirationSeconds: int32(0)})
	if err != nil {
		log.Fatalf("Error ocurred on publish: 21%v", err)
	}
}

var stime time.Time
var client pb.BrokerClient
var wg sync.WaitGroup
var count int

func Publish() {
	count = 0
	target := func() {
		defer wg.Done()
		count++

		singlePublish(client, "ali")
	}

	stime = time.Now()

	for {
		if count > 50000 {
			wg.Wait()
			fmt.Printf("Pub count: %d, in %v time\n", count, time.Since(stime))
			count = 0
			stime = time.Now()
		}

		wg.Add(1)

		if concurrent {
			go target()
		} else {
			target()
		}
	}
}

var conn *grpc.ClientConn
var url string

func newClient() (pb.BrokerClient, func()) {
	var err error

	// url := "envoy:8000"
	// url := "localhost:50051"
	// url := "192.168.70.194:31235"
	conn, err = grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return newClient()
	}

	cancel := func() {
		conn.Close()
	}

	return pb.NewBrokerClient(conn), cancel
}

func main() {
	concurrent = os.Args[1] == "conc"
	url = os.Args[2]

	client, _ = newClient()

	Publish()
}
