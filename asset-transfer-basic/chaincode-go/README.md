# Decentralized Storage System Chaincode

The chaincode of the decentralized storage system. Only modify ``chaincode-go/chaincode/smartcontract``.

## Functions

- ``UpdateNodeWeight``: updating the weight of a node.
- ``QueryNodeWeight``: querying the weight of a node.
- ``CreateVersionedHashSlot``: creating versioned hash slot table.
- ``QueryNodeID``: given the hash value of a file, querying which node should be used to store the file.
- ``GetHashSlotTable``: given a version number, querying the versioned hash slot table. 
- ``StoreFileTree``: storing the File object (structured like a tree).
- ``QueryFileTree``: querying the File object

## How to Install and Run

Follow https://hyperledger-fabric.readthedocs.io/en/release-2.5/write_first_app.html
