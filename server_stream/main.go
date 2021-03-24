package main

import (
	"context"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/devries/grpc-tutorial/api"

	"crypto/tls"
	"crypto/x509"
	"google.golang.org/grpc/credentials"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "50051"
	}

	certificate, err := tls.LoadX509KeyPair("localhost/cert.pem", "localhost/key.pem")
	if err != nil {
		log.Fatalf("could not load server key pair: %s", err)
	}

	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile("minica.pem")
	if err != nil {
		log.Fatalf("failed to read ca certificate: %s", err)
	}

	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		log.Fatal("failed to append ca certificate to certificate pool")
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("Listening on port %s", port)

	tlsConfig := &tls.Config{
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	}

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	api.RegisterPrimeStreamServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
}

type server struct{}

func (s *server) GetPrimes(in *api.PrimeCount, stream api.PrimeStream_GetPrimesServer) error {
	ctx := stream.Context()
	p, ok := peer.FromContext(ctx)
	if ok {
		log.Printf("Received Request from %s for %d primes", p.Addr, in.Number)
	} else {
		log.Printf("Received Request for %d primes", in.Number)
	}

	if in.Number < 0 {
		retErr := status.Errorf(codes.InvalidArgument, "Requested number of primes must be positive")
		log.Printf("Error: Asked for a negative amount")
		return retErr
	}

	if in.Number > 1000000 {
		retErr := status.Errorf(codes.InvalidArgument, "%d is too many primes to return", in.Number)
		log.Printf("Error: Asked for too many primes")
		return retErr
	}

	// contentBox := make([]int64, in.Number)

	// Prepare prime generator
	ch := make(chan int64)
	ctx, cancel := context.WithCancel(ctx)

	go PrimeGenerator(ctx, ch)

	for i := int64(0); i < in.Number; i++ {
		n := api.PrimeNumber{Count: i + 1, Value: <-ch}
		if err := stream.Send(&n); err != nil {
			return err
		}
	}
	cancel()

	return nil
}

func PrimeGenerator(ctx context.Context, ch chan<- int64) {
	primes := make([]int64, 0)
	ch <- int64(2)
	for i := int64(3); ; i += 2 {
		isprime := true
		iSqrt := int64(math.Floor(math.Sqrt(float64(i))))
		for _, p := range primes {
			if i%p == 0 {
				isprime = false
				break
			}
			if p > iSqrt {
				break
			}
		}
		if isprime {
			select {
			case ch <- i:
				primes = append(primes, i)
			case <-ctx.Done():
				close(ch)
				return
			}
		}
	}
}
