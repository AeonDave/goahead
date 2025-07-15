//go:ahead functions
//go:build ignore

package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

func getString() string {
	return "Hello World"
}

func getInt() int {
	return 42
}

func deriveKeyStream(seedBytes []byte, length int) []byte {
	if length <= 0 {
		panic("length must be greater than 0")
	}
	numBlocks := (length + 31) / 32
	keyStream := make([]byte, 0, numBlocks*32)
	for counter := 0; counter < numBlocks; counter++ {
		var ctrBytes [4]byte
		binary.BigEndian.PutUint32(ctrBytes[:], uint32(counter))
		toHash := append(seedBytes, ctrBytes[:]...)
		sum := sha256.Sum256(toHash)
		keyStream = append(keyStream, sum[:]...)
	}
	return keyStream[:length]
}

func shadow(data []byte, seedString string) string {
	// Decode the hex seed string to bytes, matching runtime behavior
	seedBytes, _ := hex.DecodeString(seedString)
	keyStream := deriveKeyStream(seedBytes, len(data))
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ keyStream[i]
	}
	return hex.EncodeToString(out)
}

func ShadowStr(input string) string {
	if input == "" {
		return ""
	}
	data := []byte(input)
	return shadow(data, "deadc0de")
}

func ShadowHex(hexInput string) string {
	if hexInput == "" {
		return ""
	}
	data, err := hex.DecodeString(hexInput)
	if err != nil {
		return ""
	}
	return shadow(data, "deadc0de")
}

func HashStr(input string) string {
	if input == "" {
		return ""
	}
	seed := "deadc0de"
	combined := append([]byte(seed), []byte(input)...)
	sum := sha256.Sum256(combined)
	return hex.EncodeToString(sum[:])
}
