// blockchain/blockchain.go
package blockchain

import (
	"fmt"
	"sync"
	"time"

	"github.com/KevinKien/asynchronous-execution-simulation/block"
	"github.com/KevinKien/asynchronous-execution-simulation/config"
	"github.com/KevinKien/asynchronous-execution-simulation/state"
	"github.com/KevinKien/asynchronous-execution-simulation/transaction"
	"github.com/KevinKien/asynchronous-execution-simulation/validator"
)

// Blockchain represents the main blockchain data structure
type Blockchain struct {
	Blocks               []*block.Block
	Validators           []*validator.Validator
	CurrentLeader        int
	State                *state.State
	PendingExecutions    []*block.Block
	Mempool              []*transaction.Transaction
	ExecutionMutex       sync.Mutex
	CurrentHeight        int
	LastExecutedHeight   int
	LastVerifiedHeight   int
	LastFinalizedHeight  int
	PendingStateRoots    map[int]string
	SpeculativeStateRoot map[int]string
	Config               *config.Config
}

// New creates a new blockchain
func New(cfg *config.Config) (*Blockchain, error) {
	bc := &Blockchain{
		Blocks:              make([]*block.Block, 0),
		Validators:          make([]*validator.Validator, cfg.NumValidators),
		State:               state.New(),
		PendingExecutions:   make([]*block.Block, 0),
		Mempool:             make([]*transaction.Transaction, 0),
		CurrentHeight:       0,
		LastExecutedHeight:  -1,
		LastVerifiedHeight:  -1,
		LastFinalizedHeight: -1,
		PendingStateRoots:   make(map[int]string),
		SpeculativeStateRoot: make(map[int]string),
		Config:              cfg,
	}

	// Initialize genesis block
	genesisBlock := block.Genesis()
	bc.Blocks = append(bc.Blocks, genesisBlock)

	// Initialize validators
	for i := 0; i < cfg.NumValidators; i++ {
		v, err := validator.New(i, i == 0) // First validator starts as leader
		if err != nil {
			return nil, fmt.Errorf("failed to create validator %d: %v", i, err)
		}
		bc.Validators[i] = v

		// Add validator account to state with initial balance
		bc.State.CreateAccount(v.Address, 1000)
	}

	// Set current leader
	bc.CurrentLeader = 0

	// Initialize some user accounts
	for i := 0; i < 5; i++ {
		address := fmt.Sprintf("user_%d", i)
		bc.State.CreateAccount(address, 100)
	}

	return bc, nil
}

// SubmitTransaction adds a transaction to the mempool
func (bc *Blockchain) SubmitTransaction(tx *transaction.Transaction) bool {
	// Preliminary validation
	valid := bc.validateTransaction(tx)
	if !valid {
		fmt.Printf("Transaction %s failed preliminary validation\n", tx.ID)
		return false
	}
	
	// Add to mempool
	bc.Mempool = append(bc.Mempool, tx)
	
	// Propagate to all validators' mempools
	for _, v := range bc.Validators {
		v.AddToMempool(tx)
	}
	
	fmt.Printf("Transaction %s from %s submitted and propagated to validators\n", tx.ID, tx.Sender)
	return true
}

// Validates transaction format, signature and balance
func (bc *Blockchain) validateTransaction(tx *transaction.Transaction) bool {
	// Check if sender exists
	sender, exists := bc.State.GetAccount(tx.Sender)
	if !exists {
		return false
	}
	
	// Check nonce
	if tx.Nonce != sender.Nonce {
		fmt.Printf("Invalid nonce for tx %s: expected %d, got %d\n", tx.ID, sender.Nonce, tx.Nonce)
		return false
	}
	
	// Check balance (preliminary check)
	if sender.Balance < tx.Amount {
		return false
	}
	
	// Check signature (simplified)
	if tx.Signature == nil {
		return false
	}
	
	return true
}

// RotateLeader rotates to the next leader based on round-robin scheduling
func (bc *Blockchain) RotateLeader() {
	currentLeader := bc.CurrentLeader
	bc.Validators[currentLeader].IsLeader = false
	
	// Rotate to next leader
	nextLeader := (currentLeader + 1) % bc.Config.NumValidators
	bc.CurrentLeader = nextLeader
	bc.Validators[nextLeader].IsLeader = true
	
	fmt.Printf("Leader rotated from Validator %d to Validator %d\n", currentLeader, nextLeader)
}

// ProposeBlock creates a new block proposal
func (bc *Blockchain) ProposeBlock() *block.Block {
	// Check if there are enough transactions
	if len(bc.Mempool) == 0 {
		fmt.Println("No transactions in mempool, skipping block proposal")
		return nil
	}
	
	// Get the current leader
	leader := bc.CurrentLeader
	leaderValidator := bc.Validators[leader]
	
	// Get previous block hash
	prevBlock := bc.Blocks[bc.CurrentHeight]
	
	// Determine delayed root (from D blocks ago)
	var delayedRoot string
	if bc.CurrentHeight+1 > bc.Config.ExecutionDelay {
		delayedHeight := bc.CurrentHeight + 1 - bc.Config.ExecutionDelay
		delayedRoot = bc.Blocks[delayedHeight-1].StateRoot
	} else {
		// For the first D blocks, use genesis state root
		delayedRoot = "genesis_state_root"
	}
	
	// Create a new block
	newBlock := block.New(bc.CurrentHeight+1, prevBlock.Hash, leader, delayedRoot)
	
	// Select transactions from mempool
	txCount := 0
	txsToInclude := leaderValidator.GetTxFromMempool(bc.Config.BlockSize)
	
	// Add transactions to block
	for _, tx := range txsToInclude {
		newBlock.AddTransaction(tx)
		txCount++
	}
	
	// Calculate block hash
	newBlock.CalculateHash()
	
	// Leader signs the block
	signature, err := leaderValidator.SignData([]byte(newBlock.Hash))
	if err != nil {
		fmt.Printf("Error signing block: %v\n", err)
		return nil
	}
	newBlock.Sign(leader, signature)
	
	fmt.Printf("Block %d proposed by Validator %d with %d transactions\n", 
		newBlock.Height, leader, len(newBlock.Transactions))
	
	// Remove the included transactions from the mempool
	if len(txsToInclude) > 0 {
		newTxs := make([]*transaction.Transaction, 0)
		for _, tx := range bc.Mempool {
			included := false
			for _, includedTx := range txsToInclude {
				if tx.ID == includedTx.ID {
					included = true
					break
				}
			}
			if !included {
				newTxs = append(newTxs, tx)
			}
		}
		bc.Mempool = newTxs
	}
	
	return newBlock
}

// VoteOnBlock simulates validators voting on a block
func (bc *Blockchain) VoteOnBlock(block *block.Block) bool {
	votes := 1 // Proposer already voted
	
	// Each validator validates and votes
	for _, v := range bc.Validators {
		// Skip the proposer who already signed
		if v.ID == block.Proposer {
			continue
		}
		
		// Validate block without executing transactions
		valid := bc.validateBlockWithoutExecution(block)
		if !valid {
			fmt.Printf("Validator %d rejected block %d\n", v.ID, block.Height)
			continue
		}
		
		// Sign the block
		signature, err := v.SignData([]byte(block.Hash))
		if err != nil {
			fmt.Printf("Error when validator %d signing block %d: %v\n", 
				v.ID, block.Height, err)
			continue
		}
		
		// Add signature to block
		block.Sign(v.ID, signature)
		
		// Mark as voted by this validator
		v.BlockVote[block.Hash] = true
		votes++
		
		fmt.Printf("Validator %d voted for block %d\n", v.ID, block.Height)
		
		// Check if we have a quorum
		if float64(votes)/float64(bc.Config.NumValidators) >= bc.Config.QuorumPercentage {
			block.MarkAsVoted()
			fmt.Printf("Block %d received quorum with %d/%d votes\n", 
				block.Height, votes, bc.Config.NumValidators)
			return true
		}
	}
	
	// Not enough votes
	fmt.Printf("Block %d failed to get quorum with only %d/%d votes\n", 
		block.Height, votes, bc.Config.NumValidators)
	return false
}

// Validates a block without executing the transactions
func (bc *Blockchain) validateBlockWithoutExecution(block *block.Block) bool {
	// Check if height is correct
	if block.Height != bc.CurrentHeight {
		return false
	}
	
	// Check if previous hash matches
	if block.PreviousHash != bc.Blocks[bc.CurrentHeight-1].Hash {
		return false
	}
	
	// Check if proposer is the current leader
	if block.Proposer != bc.CurrentLeader {
		return false
	}
	
	// Check delayed merkle root from D blocks ago
	if block.Height > bc.Config.ExecutionDelay {
		delayedHeight := block.Height - bc.Config.ExecutionDelay
		if block.DelayedRoot != bc.Blocks[delayedHeight-1].StateRoot {
			return false
		}
	}
	
	// Check proposer signature 
	if _, exists := block.Signatures[block.Proposer]; !exists {
		return false
	}
	
	return true
}

// FinalizeBlock marks a block as finalized when the next block is voted
func (bc *Blockchain) FinalizeBlock(height int) {
	if height <= 0 || height >= len(bc.Blocks) {
		return
	}
	
	block := bc.Blocks[height]
	block.MarkAsFinalized()
	bc.LastFinalizedHeight = height
	
	fmt.Printf("Block %d finalized\n", height)
	
	// Add to execution queue
	bc.PendingExecutions = append(bc.PendingExecutions, block)
}

// ExecuteBlock executes all transactions in a block
func (bc *Blockchain) ExecuteBlock(block *block.Block, isSpeculative bool) string {
	bc.ExecutionMutex.Lock()
	defer bc.ExecutionMutex.Unlock()
	
	// Create a copy of the state to work on
	stateCopy := bc.State.Copy()
	
	// Execute each transaction
	for _, tx := range block.Transactions {
		err := stateCopy.ExecuteTransaction(tx)
		if err != nil {
			fmt.Printf("Error executing transaction %s: %v\n", tx.ID, err)
			continue
		}
		
		fmt.Printf("Executed tx %s: %s -> %s, amount: %d\n", 
			tx.ID, tx.Sender, tx.Recipient, tx.Amount)
	}
	
	// Get the new state root
	newStateRoot := stateCopy.StateRoot
	
	if isSpeculative {
		fmt.Printf("Speculatively executed block %d, new state root: %s\n", 
			block.Height, newStateRoot)
		bc.SpeculativeStateRoot[block.Height] = newStateRoot
	} else {
		// Update the actual state
		bc.State = stateCopy
		
		// Update execution state
		bc.LastExecutedHeight = block.Height
		bc.PendingStateRoots[block.Height] = newStateRoot
		
		block.StateRoot = newStateRoot
		fmt.Printf("Executed block %d, new state root: %s\n", 
			block.Height, newStateRoot)
	}
	
	return newStateRoot
}

// VerifyBlock marks a block as verified after execution
func (bc *Blockchain) VerifyBlock(height int) {
	if height < 0 || height >= len(bc.Blocks) {
		return
	}
	
	block := bc.Blocks[height]
	
	// Update block status
	block.MarkAsVerified()
	bc.LastVerifiedHeight = height
	
	// Update state root for future blocks to verify against
	stateRoot, exists := bc.PendingStateRoots[height]
	if exists {
		block.StateRoot = stateRoot
		delete(bc.PendingStateRoots, height) // Remove from pending
	}
	
	fmt.Printf("Block %d verified with state root: %s\n", height, block.StateRoot)
}

// CheckConsistency checks if execution results match consensus
func (bc *Blockchain) CheckConsistency(height int) bool {
	if height <= bc.Config.ExecutionDelay || height >= len(bc.Blocks) {
		return true // Not enough blocks to check consistency
	}
	
	// Get the block to check
	block := bc.Blocks[height]
	
	// Get the block that should include this block's state root
	futureBlock := bc.Blocks[height+bc.Config.ExecutionDelay]
	
	// Check if state roots match
	if block.StateRoot != futureBlock.DelayedRoot {
		fmt.Printf("Consistency error at block %d: State root mismatch\n", height)
		return false
	}
	
	return true
}

// ProcessPendingExecutions processes all pending block executions
func (bc *Blockchain) ProcessPendingExecutions() {
	if len(bc.PendingExecutions) == 0 {
		return
	}
	
	pendingBlocks := bc.PendingExecutions
	bc.PendingExecutions = make([]*block.Block, 0)
	
	for _, block := range pendingBlocks {
		// Check if this was already speculatively executed
		if specRoot, exists := bc.SpeculativeStateRoot[block.Height]; exists && bc.Config.SpeculativeEnabled {
			fmt.Printf("Using speculative execution results for block %d\n", block.Height)
			
			// Update state with speculative results
			block.StateRoot = specRoot
			bc.PendingStateRoots[block.Height] = specRoot
			bc.LastExecutedHeight = block.Height
			
			// Remove from speculative storage
			delete(bc.SpeculativeStateRoot, block.Height)
		} else {
			// Regular execution
			bc.ExecuteBlock(block, false)
		}
		
		// If this block is D blocks before current height, we can verify it
		if block.Height + bc.Config.ExecutionDelay <= bc.CurrentHeight {
			bc.VerifyBlock(block.Height)
			
			// Check consistency
			bc.CheckConsistency(block.Height)
		}
	}
}

// AddBlock adds a voted block to the blockchain and finalizes the previous block
func (bc *Blockchain) AddBlock(block *block.Block) {
	// Add to blockchain
	bc.Blocks = append(bc.Blocks, block)
	bc.CurrentHeight++
	
	// If previous block exists, finalize it
	if block.Height > 1 {
		bc.FinalizeBlock(block.Height - 1)
	}
	
	// Speculative execution if enabled
	if bc.Config.SpeculativeEnabled {
		fmt.Printf("Speculatively executing block %d\n", block.Height)
		go bc.ExecuteBlock(block, true)
	}
}

// PrintState shows the current state of the blockchain
func (bc *Blockchain) PrintState() {
	fmt.Println("\n=== Blockchain State ===")
	fmt.Printf("Current Height: %d\n", bc.CurrentHeight)
	fmt.Printf("Last Executed: %d\n", bc.LastExecutedHeight)
	fmt.Printf("Last Verified: %d\n", bc.LastVerifiedHeight)
	fmt.Printf("Current Leader: Validator %d\n", bc.CurrentLeader)
	
	fmt.Println("\n=== Account Balances ===")
	fmt.Print(bc.State.String())
	
	fmt.Println("\n=== Block Status ===")
	for _, block := range bc.Blocks {
		finalized := ""
		if block.Status == block.Finalized || block.Status == block.Verified {
			finalized = "✓"
		}
		
		verified := ""
		if block.Status == block.Verified {
			verified = "✓"
		}
		
		fmt.Printf("Block %d: Status=%s, Txs=%d, Finalized=%s, Verified=%s\n",
			block.Height, block.Status, len(block.Transactions), finalized, verified)
	}
}
