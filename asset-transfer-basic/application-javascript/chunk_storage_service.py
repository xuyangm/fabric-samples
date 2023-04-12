import os
from utils import get_hash
import grpc
import protos.chunk_storage_pb2 as chunk_storage_pb2
import protos.chunk_storage_pb2_grpc as chunk_storage_pb2_grpc
from concurrent import futures

class ChunkStorageService(chunk_storage_pb2_grpc.ChunkStorageServicer):

    def __init__(self):
        self.memory_folder = "memory"
        if not os.path.exists(self.memory_folder):
            os.makedirs(self.memory_folder)

    def StoreChunk(self, request, context):
        # Calculate hash value of chunk
        calculated_hash_value = get_hash(request.chunk_data)

        # Compare calculated hash value with hash value in request
        if calculated_hash_value != request.chunk_hash:
            return chunk_storage_pb2.StoreChunkResponse(status="FAILURE")

        # Save chunk to file
        file_path = os.path.join(self.memory_folder, calculated_hash_value)
        with open(file_path, "wb") as f:
            f.write(request.chunk_data)

        return chunk_storage_pb2.StoreChunkResponse(status="SUCCESS")

def main():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    chunk_storage_pb2_grpc.add_ChunkStorageServicer_to_server(ChunkStorageService(), server)
    server.add_insecure_port('[::]:50052')
    server.start()
    print("ChunkStorageService starts...")
    server.wait_for_termination()

if __name__ == '__main__':
    main()









