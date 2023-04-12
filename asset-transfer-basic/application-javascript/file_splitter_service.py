import grpc
import protos.file_splitter_pb2 as file_splitter_pb2
import protos.file_splitter_pb2_grpc as file_splitter_pb2_grpc
import protos.chunk_storage_pb2 as chunk_storage_pb2
import protos.chunk_storage_pb2_grpc as chunk_storage_pb2_grpc
from concurrent import futures
from utils import encode, get_hash
from data_storage import File, Stripe, Chunk
import config as cfg
import subprocess
import json

class FileSplitterServicer(file_splitter_pb2_grpc.FileSplitterServicer):
    def __init__(self):
        self.file_objs = None

    def SplitFile(self, request, context):
        file_obj = File()

        # Accept file from remote nodes
        file_content = request.content
        file_obj.set_hash_value(get_hash(file_content))

        # Divide file into stripes
        stripes = []
        for i in range(0, len(file_content), cfg.stripe_size):
            stripe = file_content[i:i+cfg.stripe_size]
            if len(stripe) < cfg.stripe_size:
                stripe += b'\x00' * (cfg.stripe_size - len(stripe))
            stripes.append(stripe)
            encoded_chunks = encode(cfg.k, cfg.n, stripe)

            stripe_obj = Stripe()
            stripe_obj.set_hash_value(get_hash(stripe))
            for chunk in encoded_chunks:
                chunk_hash = get_hash(chunk)
                chunk_obj = Chunk()
                chunk_obj.set_hash_value(chunk_hash)
                stripe_obj.add_chunk(chunk_obj)

                # Store chunk in a file
                with grpc.insecure_channel('localhost:50052') as channel:
                    # create a stub
                    stub = chunk_storage_pb2_grpc.ChunkStorageStub(channel)
                    # create a request object
                    request = chunk_storage_pb2.StoreChunkRequest(chunk_data=chunk, chunk_hash=chunk_hash)
                    # send the request and get the response
                    result = stub.StoreChunk(request)
                    print(result.status)

            file_obj.add_stripe(stripe_obj)

        # Store file in the form of a File object
        if self.file_objs is None:
            self.file_objs = []
        self.file_objs.append(file_obj)
        _ = file_obj.output_to_json_file()

        # result = subprocess.run(["node", "app.js", fn], capture_output=True, text=True)
        # json_file = json.loads(result.stdout.split("*** JSONFILE: ")[1])
        # content = json_file['stripeHashes']
        # print(content)

        return file_splitter_pb2.FileResponse(message='File split and stored successfully')

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    file_splitter_pb2_grpc.add_FileSplitterServicer_to_server(FileSplitterServicer(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("FileSplitterService starts...")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()
