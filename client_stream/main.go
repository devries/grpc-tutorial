package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	_ "embed"

	"github.com/devries/grpc-tutorial/apistream"
	"google.golang.org/grpc"

	"crypto/x509"
	"google.golang.org/grpc/credentials"
)

//go:embed minica.pem
var bs []byte

func main() {
	nf := flag.Int64("n", 5, "number of primes to get")
	host := flag.String("h", "localhost", "host name")
	port := flag.Int("p", 55551, "port number")

	flag.Parse()

	pool := x509.NewCertPool()
	bs, err := os.ReadFile("minica.pem")
	if err != nil {
		log.Fatalf("Unable to load ca certificate: %s", err)
	}

	ok := pool.AppendCertsFromPEM(bs)
	if !ok {
		log.Fatal("failed to append ca certificate to pool")
	}

	address := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(pool, "")))
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := apistream.NewPrimeStreamClient(conn)
	ctx := context.Background()

	var displaydivisor int64 = 1
	switch {
	case *nf < 1000:
		displaydivisor = 1
	case *nf < 10000:
		displaydivisor = 10
	case *nf < 100000:
		displaydivisor = 100
	default:
		displaydivisor = 1000
	}

	stream, err := c.GetPrimes(ctx, &apistream.PrimeCount{Number: *nf})
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

		if res.GetCount()%displaydivisor == 0 {
			log.Printf("Received prime %7d: %8d", res.GetCount(), res.GetValue())
		}
	}
}
