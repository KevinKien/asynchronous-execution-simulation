// validator/validator.go
package validator

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"asynchronous-execution-simulation/transaction"
)

// Validator represents a node with validation privileges
type Validator struct {
	ID        int
	Address   string
	PrivKey   *ecdsa.PrivateKey
	PubKey    *ecdsa.PublicKey
	IsLeader  bool
	Mempool   []*transaction.Transaction
	BlockVote map[string]bool // Track which blocks this validator has voted on
}

// New creates a new validator
func New(id int, isLeader bool) (*Validator, error) {
	// Generate ECDSA key pair
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	validator := &Validator{
		ID:        id,
		PrivKey:   privKey,
		PubKey:    &privKey.PublicKey,
		Address:   fmt.Sprintf("validator_%d", id),
		IsLeader:  isLeader,
		Mempool:   make([]*transaction.Transaction, 0),
		BlockVote: make(map[string]bool),
	}

	return validator, nil
}

// AddToMempool adds a transaction to the validator's mempool
func (v *Validator) AddToMempool(tx *transaction.Transaction) {
	v.Mempool = append(v.Mempool, tx)
}

// SignBlock signs a block hash with the validator's private key
func (v *Validator) SignBlock(blockHash string) ([]byte, error) {
	// Decode hex string to bytes
	hashBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash: %v", err)
	}

	// Sign the hash
	signature, err := ecdsa.SignASN1(rand.Reader, v.PrivKey, hashBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign block: %v", err)
	}

	// Record that this validator has voted for this block
	v.BlockVote[blockHash] = true

	return signature, nil
}

// SignData signs arbitrary data with the validator's private key
func (v *Validator) SignData(data []byte) ([]byte, error) {
	// Create a hash of the data
	hash := sha256.Sum256(data)
	
	// Sign the hash
	signature, err := ecdsa.SignASN1(rand.Reader, v.PrivKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %v", err)
	}

	return signature, nil
}

// VerifySignature verifies a signature against a validator's public key
func VerifySignature(pubKey *ecdsa.PublicKey, data []byte, signature []byte) bool {
	// Create a hash of the data
	hash := sha256.Sum256(data)
	
	// Verify the signature
	return ecdsa.VerifyASN1(pubKey, hash[:], signature)
}

// GetTxFromMempool gets transactions from the mempool up to a certain limit
func (v *Validator) GetTxFromMempool(limit int) []*transaction.Transaction {
	if len(v.Mempool) <= limit {
		result := v.Mempool
		v.Mempool = make([]*transaction.Transaction, 0)
		return result
	}

	result := v.Mempool[:limit]
	v.Mempool = v.Mempool[limit:]
	return result
}

// String returns a string representation of the validator
func (v *Validator) String() string {
	leader := ""
	if v.IsLeader {
		leader = " (LEADER)"
	}
	return fmt.Sprintf("Validator %d%s: %s, Mempool: %d txs", 
		v.ID, leader, v.Address, len(v.Mempool))
}
