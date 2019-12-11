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

	"google.golang.org/grpc/metadata"
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

	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)), grpc.UnaryInterceptor(AuthenticationInterceptor))
	api.RegisterPrimesServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
}

type contextKey int

var contextKeyAuthorized = contextKey(1)

// This function returns authorized from the context
func Authorized(ctx context.Context) bool {
	authorized, ok := ctx.Value(contextKeyAuthorized).(bool)
	if !ok {
		return false
	}

	return authorized
}

// This is a UnaryServerInterceptor type which is a function with the signature below.
func AuthenticationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// We can perform a simple logging function here, getting the client address.
	p, ok := peer.FromContext(ctx)
	if ok {
		log.Printf("Received Request from %s", p.Addr)
	} else {
		log.Printf("Received Request")
	}

	// We will get the token from the context
	md, ok := metadata.FromIncomingContext(ctx)
	authorized := false
	if !ok {
		log.Printf("Received no metadata")
		authorized = false
	} else {
		tokens := md.Get("authorization")
		if len(tokens) < 1 || tokens[0] != "Bearer HelloWorld" {
			log.Printf("Invalid or Missing token")
			authorized = false
		} else {
			log.Printf("Received valid authorization token")
			authorized = true
		}
	}

	ctx = context.WithValue(ctx, contextKeyAuthorized, authorized)

	h, err := handler(ctx, req)
	return h, err
}

type server struct{}

func (s *server) GetPrimes(ctx context.Context, in *api.PrimeCount) (*api.PrimeNumbers, error) {
	// The interceptor places a boolean in the context to let us know if the client is authorized.
	authorized := Authorized(ctx)

	// Next we check to see if it is false.
	if !authorized {
		log.Printf("Unauthorized client")
		return nil, status.Errorf(codes.Unauthenticated, "Invalid or missing authorization token")
	}
	log.Printf("Authorized client")

	// Finally we handle the logic of the server
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
