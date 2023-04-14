import grpc
import protos.file_splitter_pb2 as file_splitter_pb2
import protos.file_splitter_pb2_grpc as file_splitter_pb2_grpc

# create a gRPC channel and stub
channel = grpc.insecure_channel('localhost:50051')
stub = file_splitter_pb2_grpc.FileSplitterStub(channel)

# open the file to be sent
with open('abc.txt', 'rb') as f:
    file_contents = f.read()

print(type(file_contents))
# create a request message with the file contents
request = file_splitter_pb2.FileRequest(content=file_contents)
print(type(request))
# call the remote method to send the file
response = stub.SplitFile(request)

# print the response message
print(response.message)


