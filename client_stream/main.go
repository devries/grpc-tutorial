package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/devries/grpc-tutorial/api"
	"google.golang.org/grpc"

	"crypto/x509"
	"google.golang.org/grpc/credentials"
)

func main() {
	nf := flag.Int64("n", 5, "number of primes to get")

	flag.Parse()

	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	pool := x509.NewCertPool()
	bs, err := ioutil.ReadFile("minica.pem")
	if err != nil {
		log.Fatalf("Unable to load ca certificate: %s", err)
	}

	ok := pool.AppendCertsFromPEM(bs)
	if !ok {
		log.Fatal("failed to append ca certificate to pool")
	}

	address := fmt.Sprintf("localhost:%s", port)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(pool, "")))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := api.NewPrimeStreamClient(conn)
	ctx := context.Background()

	stream, err := c.GetPrimes(ctx, &api.PrimeCount{Number: *nf})
	if err != nil {
		log.Fatalf("could not get primes: %s", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error: %s", err)
		}

		log.Printf("Received prime %d: %d", res.GetCount(), res.GetValue())
	}
}
