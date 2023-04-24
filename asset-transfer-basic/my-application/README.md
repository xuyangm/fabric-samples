# Decentralized Storage System

## Prerequisite

1. Install Golang (version 1.20.3)

2. Install gRPC

## How to Run

1. ```git clone https://github.com/xuyangm/fabric-samples.git```

If the following error is reported:
```
GnuTLS recv error (-110): The TLS connection was non-properly terminated.
```
Solve it using:
```
sudo apt-get install gnutls-bin
git config --global http.sslVerify false
git config --global http.postBuffer 1048576000
```

2. ```curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh && chmod +x install-fabric.sh```

3. ```./install-fabric.sh b d``` (fabric-version: 2.5.0 fabric-ca-version: 1.5.6)

4. ```cd fabric-samples/test-network```

5. ```./network.sh up createChannel -c mychannel -ca```

6. ```./network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go/ -ccl go```

7. ```cd ../asset-transfer-basic/my-application/```

8. ```go mod init```

9. ```go mod tidy```

10. ```go build boost.go```

11. ```./boost``` (use ./boost -h to see help message)

12. Build applications:
```
go build chunk_storage_service.go
go build file_partition_service.go storage_object.go
go build store_file.go
go build request_file.go storage_object.go
```

13. Start chunk_storage_service and file_partition_service respectively.<br>
Terminal 1 (Use ./chunk_storage_service -h to see help):
```
./chunk_storage_service
```
Terminal 2 (Use ./file_partition_service -h to see help):
```
./file_partition_service
```

14. To store a file (Use ./store_file -h to see help):
```
./store_file
```
The file hash will be shown on the terminal. The chunks are stored in the folder named "memory".

15. To request a file (Use ./request_file -h to see help):
```
./request_file -hash="REPLACE_WITH_THE_ACTUAL_FILE_HASH"
```
The output is stored as a file named "out".

16. Stop network:
```
cd ../../test-network
./network.sh down
```
