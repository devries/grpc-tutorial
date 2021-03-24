import logging
import argparse

import grpc

import primestream_pb2
import primestream_pb2_grpc

#address = 'localhost:30301'

def get_primes(stub, n):
    request = primestream_pb2.PrimeCount(number=n)
    for p in stub.GetPrimes(request):
        logging.info(f"Received prime {p.count}: {p.value}")

def main():
    logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %H:%M:%S')
    parser = argparse.ArgumentParser(description="Get some primes")
    parser.add_argument('-n', dest='number', default=5, type=int)
    parser.add_argument('-a', dest='host', default='localhost')
    parser.add_argument('-p', dest='port', default=30301)
    args = parser.parse_args()

    address = f'{args.host}:{args.port}'

    with open('minica.pem', 'rb') as f:
        creds = grpc.ssl_channel_credentials(f.read())

    with grpc.secure_channel(address, creds) as channel:
        try:
            stub = primestream_pb2_grpc.PrimeStreamStub(channel)
            r = get_primes(stub, args.number)
        except grpc.RpcError as e:
            logging.error(f"{e.code()}: {e.details()}")

if __name__=='__main__':
    main()
