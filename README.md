# gRPC Tutorial: A gRPC sample with security

In order to learn and understand [gRPC](https://grpc.io), I decided to
implement a simple prime number generator as a
gRPC service. The service is a simple unary remote procedure call with one
method to get some number of primes, which are returned by the server. The
[service definition](api/primes.proto) is in the `api` directory and
reproduced below.

```protobuf
syntax = "proto3";

package api;

service Primes {
  rpc GetPrimes(PrimeCount) returns (PrimeNumbers) {}
}

message PrimeCount {
  int64 number = 1;
}

message PrimeNumbers {
  repeated int64 contents = 1;
}
```

This protocol defines a service, `Primes` which has one method, `GetPrimes` and
takes a `PrimeCount` message as a request, returning a `PrimeNumbers` message as a
response. The `PrimeCount` message contains one 64 bit integer, while the
`PrimeNumbers` message contains an array of 64 bit integers.

Although there are many examples of writing simple unsecured servers and
clients, I wanted to learn how to build a real service, with error responses,
TLS encryption, and authentication. I did that in a series of steps using the
single service defined above.

I write the server in Go and have examples of clients written in both python
and Go. I have a total of five samples. They are:

- Clients and server with no encryption or authentication
    - [python client](python_one/client.py) (python_client_one)
    - [python server](python_one/server.py) (python_server_one)
    - [Go client](client_one/main.go) (client_one)
    - [Go server](server_one/main.go) (server_one)

    This server runs without any encryption, but does include error responses
    for requesting a negative amount of primes, or for requesting too many
    primes (which I set as more than 500 primes). 

- Clients with TLS, and a server behind a Cloud Run proxy
    - [python client](python_two/client.py) (python_client_two)
    - [Go client](client_two/main.go) (client_two)
    - [Go server](server_one/main.go) (server_one) - Note this is unchanged from the
      previous sample.

    This is a small detour, where I take the same server defined above and
    deploy it using Cloud Run. [Cloud Run](https://cloud.run/) can run unary gRPC services, and will
    require an encrypted connection, acting as a TLS terminating proxy. The
    certificate is automatically generated by Cloud Run and signed by [Let's
    Encrypt](https://letsencrypt.org/). The only changes I have to make is to
    require the python and Go clients to set up encrypted connections. The
    python client implicitly chooses default root certificates with the
    `grpc.ssl_channel_credentials()` call, while for the Go client the system
    default certificates are explicitly selected with the
    `x509.SystemCertPool()` call.

- Clients and server use TLS with a private CA
    - [python client](python_three/client.py) (python_client_three)
    - [python server](python_three/server.py) (python_server_three)
    - [Go client](client_three/main.go) (client_three)
    - [Go server](server_three/main.go) (server_three)

    Typically, you might expect to use a private CA for an internal API. These
    clients and server use a common root certificate authority (generated by
    minica) and the server uses a signed certificate for "localhost". The
    clients now explicitly add only the common root certificate, and will not
    recognize any endpoints that are signed by any other root certificates.

- Client and server use TLS with a private CA, and clients authenticate with a
  certificate during TLS negotiation
    - [python client](python_four/client.py) (python_client_four)
    - [python server](python_four/server.py) (python_server_four)
    - [Go client](client_four/main.go) (client_four)
    - [Go server](server_four/main.go) (server_four)

    This is the first go at client authorization using mutual TLS. The server
    requires that the client identify itself with a signed certificate. I let
    the clients use a certificate generated by minica for 127.0.0.1, while the
    server continues to use the localhost certificate generated above. Both
    the certificates are signed by the same root certificate, so the client
    and server can both verify the signatures against that same root. 

- Client and server use TLS with a private CA, and clients authenticate with a
  token.
    - [python client](python_five/client.py) (python_client_five)
    - [python server](python_five/server.py) (python_server_five)
    - [Go client](client_five/main.go) (client_five)
    - [Go server](server_five/main.go) (server_five)

    In this final example, I send an authorization bearer token from the
    client with the value "HelloWorld". The server is now somewhat more
    sophisticated, with an interceptor that logs the connection from the peer
    and validates the bearer token. It then attaches a value to the context
    indicating if the client is authorized or not. Ultimately the handler can
    determine if it wants to respond or not. Typically one might use the
    context variable to store information about the user or the role of the
    client. Then the handler could use that information to authorize the scope
    of the user's request. In this server, I optionally allow the client to
    use a client certificate. If a client certificate is passed, the server
    will validate that certificate and will not let a client with an invalid
    certificate connect. The interceptor would also be useful for rate
    limiting and any other cross-cutting functions across the service.

## Compiling the code

I use Go modules in this code, which means I use a `go.mod` file. Given this
file, when you build the client and server, the `google.golang.org/grpc`
library should automatically be installed. See the [gRPC](https://grpc.io)
page for more information about this.

To begin, install protocol buffers v3 from the [github project release
page](https://github.com/google/protobuf/releases). You will then need to
install the `protoc` plugin for Go using the command:

```sh
$ go get -u github.com/golang/protobuf/protoc-gen-go
```

Make sure the `protoc-gen-go` binary is within your `PATH`.

For the python side, set up a virtual environment with the command:

```sh
$ python -m venv venv
```

Enter the environment and install the gRPC libraries with the commands:

```sh
$ source venv/bin/activate
$ pip install grpcio
$ pip install grpcio-tools
```

In the root directory of the repository, you can generate the required Go code
with the command:

```sh
$ protoc api/primes.proto -I api/ --go_out=plugins=grpc:api
```

This will write the appropriate go file in the `api` directory where it can be
loaded. For the python client, I found it easier to run the following command
multiple times from within each python client directory:

```sh
$ python -m grpc_tools.protoc -I ../api --python_out=. --grpc_python_out=. ../api/primes.proto
```

This will generate the appropriate python files in each directory where they
can be easily imported by the client software.

The server can be built with the command:

```sh
$ go build -o primes_server server_one/main.go
```

The Go client can be built with the command:

```sh
$ go build -o primes_client client_one/main.go
```

## Running the Server and Clients

From the root directory, run the client using the command

```sh
$ ./primes_server
```

Then in another window or shell you can run the clients. The clients take one
optional argument `-n` followed by a number of primes to get. By default they
will request 5 primes. 

To request 20 primes with the Go client run

```sh
$ ./primes_client -n 20
```

from the root directory of the repository.

To use the python client to request 80 primes, from the root of the repository
run  the command:

```sh
$ python python_one/client.py -n 80
```

You can try very large or negative numbers too in order to see what an error
response looks like.

## Running TLS Clients with Cloud Run

I have put up a server at `primes-j6z4gxi7tq-uc.a.run.app:443` running on Cloud Run.
The clients are automatically set up to query that server. Hopefully the
clients will find and load the appropriate client root certificates for your
system, which should include the Let's Encrypt root certificate. If you
experience any issues, the system certificates may not have loaded correctly.

## Private Certificate Authority

For all the private certificate authority examples, the clients and servers
are hardcoded to look for a file called `minica.pem` as the root certificate
in the current directory. The clients uses a certificate in
`127.0.0.1/cert.pem` and a key in `127.0.0.1/key.pem` while the server uses a
certificate in `localhost/cert.pem` and a key in `localhost/key.pem`. All the
certificates should be signed by the root certificate.

This can easily be set up using the [minica](https://github.com/jsha/minica)
mini certificate authority, which is also available via
[homebrew](https://brew.sh/) on the Mac. Minica is a good certificate
authority for testing TLS enabled services and clients, but in production I
would recommend using something like [HashiCorp
Vault](https://www.vaultproject.io/) which can create short-lived certificates
on the fly in a secure manner. 

To set up the needed development certificates, run the following commands:

```sh
$ minica -domains localhost
$ minica -ip-addresses 127.0.0.1
```

This will create your root certificate and the server and client certificates.
