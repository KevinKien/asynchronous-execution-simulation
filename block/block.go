// block/block.go
package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/yourusername/blockchain-poa/transaction"
)

// BlockStatus represents the different stages a block can be in
type BlockStatus int

const (
	Proposed BlockStatus = iota
	Voted
	Finalized
	Verified
)

func (s BlockStatus) String() string {
	return [...]string{"Proposed", "Voted", "Finalized", "Verified"}[s]
}

// Block represents a blockchain block
type Block struct {
	Height         int
	Hash           string
	PreviousHash   string
	Transactions   []*transaction.Transaction
	Proposer       int
	StateRoot      string // Merkle root of state after execution
	DelayedRoot    string // Merkle root from D blocks ago
	Signatures     map[int][]byte
	ProposedAt     time.Time
	VotedAt        time.Time
	FinalizedAt    time.Time
	VerifiedAt     time.Time
	Status         BlockStatus
	SpeculativeExe bool // Was this block speculatively executed?
}

// New creates a new block
func New(height int, prevHash string, proposer int, delayedRoot string) *Block {
	return &Block{
		Height:       height,
		PreviousHash: prevHash,
		Proposer:     proposer,
		Transactions: make([]*transaction.Transaction, 0),
		DelayedRoot:  delayedRoot,
		Signatures:   make(map[int][]byte),
		ProposedAt:   time.Now(),
		Status:       Proposed,
	}
}

// AddTransaction adds a transaction to the block
func (b *Block) AddTransaction(tx *transaction.Transaction) {
	b.Transactions = append(b.Transactions, tx)
}

// CalculateHash calculates and updates the block's hash
func (b *Block) CalculateHash() {
	// Create a string representation of the block data
	blockData := fmt.Sprintf("%d_%s_%d_%s", 
		b.Height, 
		b.PreviousHash, 
		b.Proposer,
		b.DelayedRoot)
	
	// Add transaction IDs to the data
	for _, tx := range b.Transactions {
		blockData += "_" + tx.ID
	}
	
	// Calculate hash
	hash := sha256.Sum256([]byte(blockData))
	b.Hash = hex.EncodeToString(hash[:])
}

// Sign adds a validator's signature to the block
func (b *Block) Sign(validatorID int, signature []byte) {
	b.Signatures[validatorID] = signature
}

// HasQuorum checks if the block has enough signatures for a quorum
func (b *Block) HasQuorum(totalValidators int, quorumPercentage float64) bool {
	return float64(len(b.Signatures))/float64(totalValidators) >= quorumPercentage
}

// MarkAsVoted marks the block as voted and sets the timestamp
func (b *Block) MarkAsVoted() {
	b.Status = Voted
	b.VotedAt = time.Now()
}

// MarkAsFinalized marks the block as finalized and sets the timestamp
func (b *Block) MarkAsFinalized() {
	b.Status = Finalized
	b.FinalizedAt = time.Now()
}

// MarkAsVerified marks the block as verified and sets the timestamp
func (b *Block) MarkAsVerified() {
	b.Status = Verified
	b.VerifiedAt = time.Now()
}

// Genesis creates a genesis block
func Genesis() *Block {
	genesisBlock := &Block{
		Height:       0,
		Hash:         "genesis",
		PreviousHash: "",
		Proposer:     0,
		StateRoot:    "genesis_state_root",
		DelayedRoot:  "genesis_state_root",
		Signatures:   make(map[int][]byte),
		Status:       Verified,
		FinalizedAt:  time.Now(),
		VerifiedAt:   time.Now(),
	}
	return genesisBlock
}

// String returns a string representation of the block
func (b *Block) String() string {
	return fmt.Sprintf("Block %d [%s]: Status=%s, Txs=%d", 
		b.Height, b.Hash[:8], b.Status, len(b.Transactions))
}
