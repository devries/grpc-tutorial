package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/devries/grpc-tutorial/api"
	"google.golang.org/grpc"

	"crypto/tls"
	"crypto/x509"
	"google.golang.org/grpc/credentials"
)

const (
	address = "localhost:50051"
)

func main() {
	nf := flag.Int64("n", 5, "number of primes to get")

	flag.Parse()

	certificates, err := tls.LoadX509KeyPair("127.0.0.1/cert.pem", "127.0.0.1/key.pem")
	if err != nil {
		log.Fatalf("Unable to load client certificate and key: %s", err)
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

	transportCreds := credentials.NewTLS(&tls.Config{
		ServerName:   "localhost",
		Certificates: []tls.Certificate{certificates},
		RootCAs:      pool,
	})

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(transportCreds))
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
