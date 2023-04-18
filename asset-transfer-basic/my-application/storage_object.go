package main

// import (
// 	"encoding/json"
// )

type Chunk struct {
    ChunkHash string `json:"chunkHash"`
}

func (c *Chunk) SetHashValue(hashValue string) {
    c.ChunkHash = hashValue
}

type Stripe struct {
    StripeHash  string  `json:"stripeHash"`
    ChunkHashes []Chunk `json:"chunkHashes"`
}

func (s *Stripe) SetHashValue(hashValue string) {
    s.StripeHash = hashValue
}

func (s *Stripe) AddChunk(chunk Chunk) {
    s.ChunkHashes = append(s.ChunkHashes, chunk)
}

type File struct {
    FileHash    string   `json:"fileHash"`
    StripeHashes []Stripe `json:"stripeHashes"`
}

func (f *File) SetHashValue(hashValue string) {
    f.FileHash = hashValue
}

func (f *File) AddStripe(stripe Stripe) {
    f.StripeHashes = append(f.StripeHashes, stripe)
}


