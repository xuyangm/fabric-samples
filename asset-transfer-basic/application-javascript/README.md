# Decentralized Storage System Applications

The code is modified based on asset-transfer-basic chaincode in fabric-samples. 

## Function of Each File

application-javascript

├─``app.js``                   : Run this script to interact with chaincode, eg. ```node app.js cddfb3.json```.

├─``cddfb3.json``              : An example json file of a File object.

├─``chunk_storage_service.py`` : This microservice accepts chunks from other nodes and stores them in the folder "memory".

├─``client.py``                : Run this client to request storing a file.

├─``config.py``                : Some configuration, eg. `n` and ``k`` of erasure coding, the size of a stripe.

├─``data_storage.py``          : Data structure, including class File, Stripe and Chunk.

├─``file_splitter_service.py`` : This microservice accepts request from the client and partitions the file into stripes and chunks.

├─``utils.py``                 : Some tools, including ``encode()`` and ``decode()`` for erasure coding, ``get_hash()`` for calculating the hash value of any string/bytes object.

├─``protos``                   : gRPC protocol definition.

├─``memory``

## Prerequisites

1. Install Zfec
```
pip install zfec
```

2. Install gRPC
```
python3 -m pip install grpcio
python3 -m pip install grpcio-tools
```

## Current Usage

### File Splitting

1. Create a file ``abc.txt`` for splitting.

2. Start chunk_storage_service:
```
python3 chunk_storage_service.py
```

3. Start file_splitter_service:
```
python3 file_splitter_service.py
```

4. Start client:
```
python3 client.py
```

5. Check the folder ``memory`` and the new json file.

### Interaction with Chaincode

1. Modify the code of ``app.js`` to invoke different functions of the chaincode:

  - ``UpdateNodeWeight``
  - ``QueryNodeWeight``
  - ``CreateVersionedHashSlot``
  - ``QueryNodeID``
  - ``GetHashSlotTable``
  - ``StoreFileTree``
  - ``QueryFileTree``

2. If invoking ``StoreFileTree``, use ``node app.js XXXX.json``. Or use ``node app.js``.
