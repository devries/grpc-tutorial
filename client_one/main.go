package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/devries/grpc-tutorial/api"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	nf := flag.Int64("n", 5, "number of primes to get")

	flag.Parse()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	c := api.NewPrimesClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := c.GetPrimes(ctx, &api.PrimeCount{Number: *nf})
	if err != nil {
		log.Fatalf("could not get primes: %s", err)
	}

	primeStrings := make([]string, 0, *nf)
	for _, p := range r.Contents {
		primeStrings = append(primeStrings, strconv.FormatInt(p, 10))
	}

	log.Printf("First %d primes: %s", *nf, strings.Join(primeStrings, ", "))
}
