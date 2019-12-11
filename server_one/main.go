package main

import (
	"context"
	"log"
	"math"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/devries/grpc-tutorial/api"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("Listening on port %s", port)

	s := grpc.NewServer()
	api.RegisterPrimesServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
}

type server struct{}

func (s *server) GetPrimes(ctx context.Context, in *api.PrimeCount) (*api.PrimeNumbers, error) {
	p, ok := peer.FromContext(ctx)
	if ok {
		log.Printf("Received Request from %s", p.Addr)
	} else {
		log.Printf("Received Request")
	}

	if in.Number < 0 {
		retErr := status.Errorf(codes.InvalidArgument, "Requested number of primes must be positive")
		log.Printf("Error: Asked for a negative amount")
		return nil, retErr
	}

	if in.Number > 500 {
		retErr := status.Errorf(codes.InvalidArgument, "%d is too many primes to return", in.Number)
		log.Printf("Error: Asked for too many primes")
		return nil, retErr
	}

	contentBox := make([]int64, in.Number)

	// Prepare prime generator
	ch := make(chan int64)
	ctx, cancel := context.WithCancel(ctx)

	go PrimeGenerator(ctx, ch)

	for i := int64(0); i < in.Number; i++ {
		contentBox[i] = <-ch
	}
	cancel()

	return &api.PrimeNumbers{Contents: contentBox}, nil
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
