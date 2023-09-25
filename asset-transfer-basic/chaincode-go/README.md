# Decentralized Storage System Chaincode

The chaincode of the decentralized storage system. Only modify ``chaincode-go/chaincode/smartcontract``.

## Functions

- ``UpdateOrgWeight``: updating the weight of a master node.
- ``GetOrgID``: given the hash value of a file, querying which org should be used to store the file.
- ``CreateHashSlotTable``: creating inter-org hash slot table.
- ``GetHashSlotTable``: querying the inter-org hash slot table. 
- ``GetFileTree``: querying the File object
- ``StoreFileTree``: storing the File object (structured like a tree).

## How to Install and Run

Follow https://hyperledger-fabric.readthedocs.io/en/release-2.5/write_first_app.html
