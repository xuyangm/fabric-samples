package chaincode

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Slot struct {
	StartSlot    int     `json:"startSlot"`
	EndSlot      int     `json:"endSlot"`
}

type HashSlotTable struct {
	HST map[string]Slot `json:"hashSlotTable"`
}

type WeightTable struct {
	WT map[string]int `json:"nodeWeightTable"`
}

// storing the file tree
type Chunk struct {
	ChunkHash string `json:"chunkHash"`
}

type StripeTree struct {
	StripeHash  string    `json:"stripeHash"`
	ChunkHashes []Chunk  `json:"chunkHashes"`
}

type FileTree struct {
	StripeHashes []StripeTree  `json:"stripeHashes"`
}

var weightTableKey = "wt"
var numOfSlots = 16384

// This function is used for updating the weight of each node.
func (s *SmartContract) UpdateNodeWeight(ctx contractapi.TransactionContextInterface, nodeID string, weight int) error {
	// check if weight table exists
	weightTableJSON, err := ctx.GetStub().GetState(weightTableKey)
	if err != nil {
		return fmt.Errorf("failed to read weight table from state: %v", err)
	}

	var weightTable WeightTable

	// if weight table does not exist, create a new one
	if weightTableJSON == nil {
		weightTable = WeightTable{
			WT: make(map[string]int),
		}
	} else {
		// unmarshal the weight table
		err = json.Unmarshal(weightTableJSON, &weightTable)
		if err != nil {
			return fmt.Errorf("failed to unmarshal weight table: %v", err)
		}
	}

	// update the weight of the node
	weightTable.WT[nodeID] = weight

	// marshal the updated weight table
	updatedWeightTableJSON, err := json.Marshal(weightTable)
	if err != nil {
		return fmt.Errorf("failed to marshal updated weight table: %v", err)
	}

	// put the updated weight table back into the state
	err = ctx.GetStub().PutState(weightTableKey, updatedWeightTableJSON)
	if err != nil {
		return fmt.Errorf("failed to update weight table in state: %v", err)
	}

	return nil
}

// This function is used for querying the weight of a node.
func (s *SmartContract) QueryNodeWeight(ctx contractapi.TransactionContextInterface, nodeID string) (int, error) {
	// check if weight table exists
	weightTableJSON, err := ctx.GetStub().GetState(weightTableKey)
	if err != nil {
		return 0, fmt.Errorf("failed to read weight table from state: %v", err)
	}

	var weightTable WeightTable

	// if weight table does not exist, return an error
	if weightTableJSON == nil {
		return 0, fmt.Errorf("weight table does not exist")
	} else {
		// unmarshal the weight table
		err = json.Unmarshal(weightTableJSON, &weightTable)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal weight table: %v", err)
		}
	}

	// check if node exists in weight table
	weight, ok := weightTable.WT[nodeID]
	if !ok {
		return 0, fmt.Errorf("node does not exist in weight table")
	}

	return weight, nil
}

func (s *SmartContract) CreateVersionedHashSlot(ctx contractapi.TransactionContextInterface, blockHeight int) error {
	// get version
	currentVersion := blockHeight/10
	weightTableJSON, err := ctx.GetStub().GetState(weightTableKey)
	if err != nil {
		return fmt.Errorf("failed to read weight table from state: %v", err)
	}

	var weightTable WeightTable

	// if weight table does not exist, return an empty map
	if weightTableJSON == nil {
		return fmt.Errorf("empty weight table: %v", err)
	} else {
		// unmarshal the weight table
		err = json.Unmarshal(weightTableJSON, &weightTable)
		if err != nil {
			return fmt.Errorf("failed to unmarshal weight table: %v", err)
		}
	}

	nodeIDs := make([]string, 0)

	// iterate through the weight table and print each node ID and its weight
	totalWeight := 0
	for nodeID, weight := range weightTable.WT {
		nodeIDs = append(nodeIDs, nodeID)
		totalWeight += weight
	}
	sort.Strings(nodeIDs)

	hashSlotTable := HashSlotTable{
		HST: make(map[string]Slot),
	}
	startSlot := 0
	var lastNode string

	for _, nodeID := range nodeIDs {
		endSlot := startSlot + weightTable.WT[nodeID]*numOfSlots/totalWeight
		hashSlotTable.HST[nodeID] = Slot{
			StartSlot: startSlot,
			EndSlot:   endSlot,
		}
		startSlot = endSlot + 1
		lastNode = nodeID
	}

	if entry, ok := hashSlotTable.HST[lastNode]; ok {
		entry.EndSlot = numOfSlots
		hashSlotTable.HST[lastNode] = entry
	}

	// marshal the hash slot table
	hashSlotTableJSON, err := json.Marshal(hashSlotTable)
	if err != nil {
		return fmt.Errorf("failed to marshal hash slot table: %v", err)
	}

	// put the hash slot table into the state
	err = ctx.GetStub().PutState(fmt.Sprintf("%d", currentVersion), hashSlotTableJSON)
	if err != nil {
		return fmt.Errorf("failed to update hash slot table in state: %v", err)
	}
	return nil
}

// This function takes a hash value as input and calculates the node ID.
func (s *SmartContract) QueryNodeID(ctx contractapi.TransactionContextInterface, hashValue string, blockHeight int) (string, error) {
	// calculate slot ID
	hashInt := new(big.Int)
	hashInt.SetString(hashValue, 16)
	hashMod := hashInt.Mod(hashInt, big.NewInt(int64(numOfSlots)))
	slotID := int(hashMod.Int64())

	// get current versioned hash slot table from state
	currentVersion := blockHeight/10

	hashSlotTableJSON, err := ctx.GetStub().GetState(fmt.Sprintf("%d", currentVersion))
	if err != nil {
		return "", fmt.Errorf("failed to read hash slot table from state: %v", err)
	}

	var hashSlotTable HashSlotTable

	// if hash slot table does not exist, return an error
	if hashSlotTableJSON == nil {
		return "", fmt.Errorf("hash slot table does not exist")
	} else {
		// unmarshal the hash slot table
		err = json.Unmarshal(hashSlotTableJSON, &hashSlotTable)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal hash slot table: %v", err)
		}
	}

	// iterate through the hash slot table and find the corresponding nodeID
	for nodeID, slot := range hashSlotTable.HST {
		if slotID >= slot.StartSlot && slotID <= slot.EndSlot {
			return nodeID, nil
		}
	}

	return "", fmt.Errorf("no nodeID found for hash value")
}

// This function returns the range of hash values corresponding to each node. Only for test.
func (s *SmartContract) GetHashSlotTable(ctx contractapi.TransactionContextInterface, blockHeight int) (string, error) {
	version := blockHeight/10
	// get hash slot table from state
	hashSlotTableJSON, err := ctx.GetStub().GetState(fmt.Sprintf("%d", version))
	if err != nil {
		return "", fmt.Errorf("failed to read hash slot table from state: %v", err)
	}

	// if hash slot table does not exist, return an error
	if hashSlotTableJSON == nil {
		return "", fmt.Errorf("hash slot table does not exist")
	} 

	return string(hashSlotTableJSON), nil
}

func (s *SmartContract) StoreFileTree(ctx contractapi.TransactionContextInterface, fileHash string, fileTreeJSON string) (string, error) {
	args := []byte(fileTreeJSON)

	err := ctx.GetStub().PutState(fileHash, args)
	if err != nil {
		return "failed to store FileTree: failed to update FileTree in state", err
	}

	return "FileTree stored successfully", nil
}

// This function accepts a hash of a file and returns the FileTree of this file in the form of a json string.
func (s *SmartContract) QueryFileTree(ctx contractapi.TransactionContextInterface, fileHash string) (string, error) {
	// get the FileTree from the state
	args, err := ctx.GetStub().GetState(fileHash)
	if err != nil {
		return "", fmt.Errorf("failed to read FileTree from state: %v", err)
	}

	// if FileTree does not exist, return an error
	if args == nil {
		return "", fmt.Errorf("FileTree does not exist")
	}

	return string(args), nil
}

