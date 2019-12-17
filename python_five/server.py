from concurrent import futures
import logging
import math
import itertools

import grpc

import primes_pb2
import primes_pb2_grpc

def authorization_role(func):
    def wrapper(self, request, context):
        metadict = metadata_to_dict(context.invocation_metadata())
        if metadict.get('authorization')=='Bearer HelloWorld':
            context.user='standard'
        else:
            context.user=None
        return func(self, request, context)
    return wrapper

class Primes(primes_pb2_grpc.PrimesServicer):
    @authorization_role
    def GetPrimes(self, request, context):
        if context.user!='standard':
            context.abort(grpc.StatusCode.UNAUTHENTICATED, "Standard user role required")

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

    authorizer = AuthorizationInterceptor('HelloWorld')

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10), interceptors=(authorizer,))
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

def terminator(code, details):
    def terminate(request, context):
        context.abort(code, details)

    return grpc.unary_unary_rpc_method_handler(terminate)

class AuthorizationInterceptor(grpc.ServerInterceptor):
    def __init__(self, auth_token):
        self.auth_token = auth_token

    def intercept_service(self, continuation, handler_call_details):
        metadict = metadata_to_dict(handler_call_details.invocation_metadata)

        if metadict.get('authorization')!=f'Bearer {self.auth_token}':
            return terminator(grpc.StatusCode.UNAUTHENTICATED, "Authorizaiton Required")
        else:
            return continuation(handler_call_details)

if __name__=='__main__':
    logging.basicConfig(level=logging.DEBUG, format='%(asctime)s %(message)s', datefmt='%Y/%m/%d %H:%M:%S')
    serve()
