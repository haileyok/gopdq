package helpers

import (
	"encoding/hex"
	"fmt"
	"math/bits"
)

// Converts a 64-character hexidecimal PDQ hash into a 256-character binary string
// representation, useful for inserting into some vector stores.
func PdqHashToBinary(input string) (string, error) {
	hashb, err := hex.DecodeString(input)
	if err != nil {
		return "", err
	}

	result := make([]byte, len(hashb)*8)
	for i, b := range hashb {
		for j := 7; j >= 0; j-- {
			if (b>>j)&1 == 1 {
				result[i*8+(7-j)] = '1'
			} else {
				result[i*8+(7-j)] = '0'
			}
		}
	}

	return string(result), nil
}

// Calculate the hamming distance between two PDQ hashes. Input hashes should be 64-character
// hexidecimal strings. Returns a value between 0 (identical) and 256 (completely different).
func HammingDistance(hashOne, hashTwo string) (int, error) {
	bytes1, err := hex.DecodeString(hashOne)
	if err != nil {
		return 0, fmt.Errorf("invalid hash1: %w", err)
	}

	bytes2, err := hex.DecodeString(hashTwo)
	if err != nil {
		return 0, fmt.Errorf("invalid hash2: %w", err)
	}

	if len(bytes1) != 32 {
		return 0, fmt.Errorf("first hash has invalid length: expected 32 bytes, got %d", len(bytes1))
	}
	if len(bytes2) != 32 {
		return 0, fmt.Errorf("second hash has invalid length: expected 32 bytes, got %d", len(bytes2))
	}

	distance := 0
	for i := range 32 {
		xor := bytes1[i] ^ bytes2[i]
		distance += bits.OnesCount8(xor)
	}

	return distance, nil
}
