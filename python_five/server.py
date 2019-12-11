from concurrent import futures
import logging
import math
import itertools

import grpc

import primes_pb2
import primes_pb2_grpc

class Primes(primes_pb2_grpc.PrimesServicer):
    def GetPrimes(self, request, context):
        metadata = context.invocation_metadata()
        metadict = metadata_to_dict(metadata)
        if metadict.get('authorization')!='Bearer HelloWorld':
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "Authorizaiton Required")

        if request.number>500:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, f"{request.number} is too many primes to return")
        if request.number<0:
            context.abort(grpc.StatusCode.INVALID_ARGUMENT, "Requested number of primes must be positive")

        l = list(primes(context, request.number))
        return primes_pb2.PrimeNumbers(contents=l)

def metadata_to_dict(md):
    d = {}
    for m in md:
        d[m[0]]=m[1]

    return d
def serve():
    with open('minica.pem', 'rb') as f:
        root_cert = f.read()

    with open('localhost/cert.pem', 'rb') as f:
        cert = f.read()
    
    with open('localhost/key.pem', 'rb') as f:
        private_key = f.read()

    creds = grpc.ssl_server_credentials(((private_key, cert),),
            root_certificates=root_cert)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    primes_pb2_grpc.add_PrimesServicer_to_server(Primes(), server)
    server.add_secure_port('[::]:50051', creds)
    server.start()
    server.wait_for_termination()

def primes(context, nreq):
    primes = []
    yield 2

    i = 3
    ctr = 0
    for i in itertools.count(3,2):
        if not (context.is_active() and ctr<nreq):
            break

        isprime = True
        iSqrt = int(math.floor(math.sqrt(i)))

        for p in primes:
            if i%p == 0:
                isprime = False
                break
            if p > iSqrt:
                break

        if isprime:
            primes.append(i)
            ctr+=1
            yield i

if __name__=='__main__':
    logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %H:%M:%S')
    serve()
