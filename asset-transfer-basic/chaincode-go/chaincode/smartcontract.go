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
	WT map[string]int `json:"orgWeightTable"`
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
var hashSlotKey = "slt"
var numOfSlots = 16384

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// Create an empty weight table to initialize the weight of each org
	weightTable := WeightTable{
		WT: make(map[string]int),
	}

	weightTableJSON, err := json.Marshal(weightTable)
	if err != nil {
		return fmt.Errorf("failed to marshal WeightTable: %v", err)
	}

	err = ctx.GetStub().PutState(weightTableKey, weightTableJSON)
	if err != nil {
		return fmt.Errorf("failed to store WeightTable in state: %v", err)
	}

	return nil
}

func (s *SmartContract) UpdateOrgWeight(ctx contractapi.TransactionContextInterface, orgID string, weight int) error {
	weightTableJSON, err := ctx.GetStub().GetState(weightTableKey)
	if err != nil {
		return fmt.Errorf("failed to read weight table from state: %v", err)
	}

	var weightTable WeightTable

	if weightTableJSON == nil {
		weightTable = WeightTable{
			WT: make(map[string]int),
		}
	} else {
		err = json.Unmarshal(weightTableJSON, &weightTable)
		if err != nil {
			return fmt.Errorf("failed to unmarshal weight table: %v", err)
		}
	}

	weightTable.WT[orgID] = weight

	updatedWeightTableJSON, err := json.Marshal(weightTable)
	if err != nil {
		return fmt.Errorf("failed to marshal updated weight table: %v", err)
	}

	err = ctx.GetStub().PutState(weightTableKey, updatedWeightTableJSON)
	if err != nil {
		return fmt.Errorf("failed to update weight table in state: %v", err)
	}

	return nil
}

func (s *SmartContract) GetOrgID(ctx contractapi.TransactionContextInterface, stripeHash string) (string, error) {
	hashInt := new(big.Int)
	hashInt.SetString(stripeHash, 16)
	hashMod := hashInt.Mod(hashInt, big.NewInt(int64(numOfSlots)))
	slotID := int(hashMod.Int64())

	hashSlotTableJSON, err := ctx.GetStub().GetState(hashSlotKey)
	if err != nil {
		return "", fmt.Errorf("failed to read hash slot table from state: %v", err)
	}

	var hashSlotTable HashSlotTable

	if hashSlotTableJSON == nil {
		return "", fmt.Errorf("hash slot table does not exist")
	} else {
		err = json.Unmarshal(hashSlotTableJSON, &hashSlotTable)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal hash slot table: %v", err)
		}
	}

	for orgID, slot := range hashSlotTable.HST {
		if slotID >= slot.StartSlot && slotID <= slot.EndSlot {
			return orgID, nil
		}
	}

	return "", fmt.Errorf("no orgID found for hash value")
}

func (s *SmartContract) CreateHashSlotTable(ctx contractapi.TransactionContextInterface) error {
	weightTableJSON, err := ctx.GetStub().GetState(weightTableKey)
	if err != nil {
		return fmt.Errorf("failed to read weight table from state: %v", err)
	}

	var weightTable WeightTable

	if weightTableJSON == nil {
		return fmt.Errorf("empty weight table: %v", err)
	} else {
		err = json.Unmarshal(weightTableJSON, &weightTable)
		if err != nil {
			return fmt.Errorf("failed to unmarshal weight table: %v", err)
		}
	}

	orgIDs := make([]string, 0)

	totalWeight := 0
	for orgID, weight := range weightTable.WT {
		orgIDs = append(orgIDs, orgID)
		totalWeight += weight
	}
	sort.Strings(orgIDs)

	hashSlotTable := HashSlotTable{
		HST: make(map[string]Slot),
	}
	startSlot := 0
	var lastOrg string

	for _, orgID := range orgIDs {
		endSlot := startSlot + weightTable.WT[orgID]*numOfSlots/totalWeight
		hashSlotTable.HST[orgID] = Slot{
			StartSlot: startSlot,
			EndSlot:   endSlot,
		}
		startSlot = endSlot + 1
		lastOrg = orgID
	}

	if entry, ok := hashSlotTable.HST[lastOrg]; ok {
		entry.EndSlot = numOfSlots
		hashSlotTable.HST[lastOrg] = entry
	}

	hashSlotTableJSON, err := json.Marshal(hashSlotTable)
	if err != nil {
		return fmt.Errorf("failed to marshal hash slot table: %v", err)
	}

	err = ctx.GetStub().PutState(hashSlotKey, hashSlotTableJSON)
	if err != nil {
		return fmt.Errorf("failed to update hash slot table in state: %v", err)
	}
	return nil
}

func (s *SmartContract) GetHashSlotTable(ctx contractapi.TransactionContextInterface) (string, error) {
	hashSlotTableJSON, err := ctx.GetStub().GetState(hashSlotKey)
	if err != nil {
		return "", fmt.Errorf("failed to read hash slot table from state: %v", err)
	}

	if hashSlotTableJSON == nil {
		return "", fmt.Errorf("hash slot table does not exist")
	} 

	return string(hashSlotTableJSON), nil
}

func (s *SmartContract) GetFileTree(ctx contractapi.TransactionContextInterface, fileHash string) (string, error) {
	args, err := ctx.GetStub().GetState(fileHash)
	if err != nil {
		return "", fmt.Errorf("failed to read FileTree from state: %v", err)
	}

	if args == nil {
		return "", fmt.Errorf("FileTree does not exist")
	}

	return string(args), nil
}

func (s *SmartContract) StoreFileTree(ctx contractapi.TransactionContextInterface, fileHash string, fileTreeJSON string) (string, error) {
	args := []byte(fileTreeJSON)

	err := ctx.GetStub().PutState(fileHash, args)
	if err != nil {
		return "failed to store FileTree: failed to update FileTree in state", err
	}

	return "FileTree stored successfully", nil
}
