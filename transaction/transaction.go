// transaction/transaction.go
package transaction

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"time"
)

// Transaction represents a user transaction
type Transaction struct {
	ID        string
	Sender    string
	Recipient string
	Amount    int
	Nonce     int
	Signature []byte
	Timestamp time.Time
}

// New creates a new transaction
func New(sender, recipient string, amount int, nonce int, privKey *ecdsa.PrivateKey) *Transaction {
	tx := &Transaction{
		ID:        fmt.Sprintf("tx_%s_%d", sender, nonce),
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
		Nonce:     nonce,
		Timestamp: time.Now(),
	}
	
	// Sign the transaction
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s_%s_%d_%d", sender, recipient, amount, nonce)))
	sig, _ := ecdsa.SignASN1(rand.Reader, privKey, hash[:])
	tx.Signature = sig
	
	return tx
}

// GetHash returns a hash representation of the transaction
func (tx *Transaction) GetHash() []byte {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s_%s_%d_%d", 
		tx.Sender, tx.Recipient, tx.Amount, tx.Nonce)))
	return hash[:]
}

// Validate performs basic transaction validation
func (tx *Transaction) Validate() bool {
	// Simple validation - just check if it has a signature
	return tx.Signature != nil
}

// String returns a string representation of the transaction
func (tx *Transaction) String() string {
	return fmt.Sprintf("TX[%s]: %s -> %s, Amount: %d, Nonce: %d", 
		tx.ID, tx.Sender, tx.Recipient, tx.Amount, tx.Nonce)
}
