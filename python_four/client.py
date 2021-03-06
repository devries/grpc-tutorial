import logging
import argparse

import grpc

import primes_pb2
import primes_pb2_grpc

address = 'localhost:50051'

def get_primes(stub, n):
    request = primes_pb2.PrimeCount(number=n)
    response = stub.GetPrimes(request)

    return response.contents

def main():
    logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %H:%M:%S')
    parser = argparse.ArgumentParser(description="Get some primes")
    parser.add_argument('-n', dest='number', default=5, type=int)
    args = parser.parse_args()

    with open('minica.pem', 'rb') as f:
        root_cert = f.read()

    with open('127.0.0.1/cert.pem', 'rb') as f:
        cert = f.read()
    
    with open('127.0.0.1/key.pem', 'rb') as f:
        private_key = f.read()

    creds = grpc.ssl_channel_credentials(certificate_chain=cert,
            private_key=private_key,
            root_certificates=root_cert)

    with grpc.secure_channel(address, creds) as channel:
        try:
            stub = primes_pb2_grpc.PrimesStub(channel)
            r = get_primes(stub, args.number)
            logging.info(f"First {args.number} primes: "+', '.join([str(i) for i in r]))
        except grpc.RpcError as e:
            logging.error(f"{e.code()}: {e.details()}")

if __name__=='__main__':
    main()
