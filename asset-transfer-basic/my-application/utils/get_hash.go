package utils

import (
    "bytes"
    "crypto/sha256"
    "encoding/hex"
    "github.com/klauspost/reedsolomon"
)

func GetHash(input []byte) string {
    hash := sha256.Sum256(input)
    return hex.EncodeToString(hash[:])
}

func Encode(n int, k int, input []byte) ([][]byte, error) {
    enc, err := reedsolomon.New(k, n-k)
    if err != nil {
        return nil, err
    }

    shards, err := enc.Split(input)
    if err != nil {
        return nil, err
    }
    err = enc.Encode(shards)
    if err != nil {
        return nil, err
    }
    return shards, nil
}

func Decode(n int, k int, shards [][]byte) ([]byte, error) {
    enc, err := reedsolomon.New(k, n-k)
    if err != nil {
        return nil, err
    }
    err = enc.Reconstruct(shards)
    if err != nil {
        return nil, err
    }
    var buf bytes.Buffer
    err = enc.Join(&buf, shards, len(shards[0])*k)
    if err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}





