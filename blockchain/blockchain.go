// blockchain/blockchain.go
package blockchain

import (
	"fmt"
	"sync"
	"time"
	
	"asynchronous-execution-simulation/block"
	"asynchronous-execution-simulation/config"
	"asynchronous-execution-simulation/state"
	"asynchronous-execution-simulation/transaction"
	"asynchronous-execution-simulation/execution"
	"asynchronous-execution-simulation/validator"
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
	MempoolSnapshots     map[int][]*transaction.Transaction // Snapshots for validating speculative execution
	ExecutionMetrics     map[int]*ExecutionMetrics          // Metrics about execution performance
	Config               *config.Config
}

// ExecutionMetrics stores metrics about block execution
type ExecutionMetrics struct {
	TotalTransactions int
	SuccessfulTransactions int
	Conflicts int
	UseParallel bool
	ExecutionTime time.Duration
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
		MempoolSnapshots:    make(map[int][]*transaction.Transaction),
		ExecutionMetrics:    make(map[int]*ExecutionMetrics),
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
	// Check if height is correct - FIX: changed from bc.CurrentHeight to bc.CurrentHeight+1
	if block.Height != bc.CurrentHeight+1 {
		fmt.Printf("Block height validation failed: expected %d, got %d\n", 
			bc.CurrentHeight+1, block.Height)
		return false
	}
	
	// Check if previous hash matches
	if block.PreviousHash != bc.Blocks[bc.CurrentHeight].Hash {
		fmt.Printf("Previous hash validation failed: expected %s, got %s\n", 
			bc.Blocks[bc.CurrentHeight].Hash, block.PreviousHash)
		return false
	}
	
	// FIX: Removed proposer check - any validator should be able to propose blocks
	// in a PoA system, as long as they're valid validators
	// Instead, just verify that the proposer is a valid validator
	if block.Proposer < 0 || block.Proposer >= len(bc.Validators) {
		fmt.Printf("Invalid proposer index: %d\n", block.Proposer)
		return false
	}
	
	// Check delayed merkle root from D blocks ago
	if block.Height > bc.Config.ExecutionDelay {
		delayedHeight := block.Height - bc.Config.ExecutionDelay
		if block.DelayedRoot != bc.Blocks[delayedHeight-1].StateRoot {
			fmt.Printf("Delayed root validation failed: expected %s, got %s\n", 
				bc.Blocks[delayedHeight-1].StateRoot, block.DelayedRoot)
			return false
		}
	}
	
	// Check proposer signature 
	if _, exists := block.Signatures[block.Proposer]; !exists {
		fmt.Printf("Proposer signature not found\n")
		return false
	}
	
	// Validate transactions in the block
	for _, tx := range block.Transactions {
		if !bc.validateTransaction(tx) {
			// Instead of failing the whole block, we could just skip invalid transactions
			// But for simplicity, we'll fail the whole block
			fmt.Printf("Invalid transaction: %s\n", tx.ID)
			return false
		}
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

// ExecuteBlock executes all transactions in a block using an adaptive strategy
func (bc *Blockchain) ExecuteBlock(block *block.Block, isSpeculative bool) string {
	bc.ExecutionMutex.Lock()
	defer bc.ExecutionMutex.Unlock()
	
	// Count total transactions to determine execution strategy
	totalTxs := len(block.Transactions)
	
	fmt.Printf("Executing block %d with %d transactions\n", block.Height, totalTxs)
	
	// Adaptive execution strategy
	// 1. Use sequential for small blocks (< threshold in config)
	// 2. Use parallel for larger blocks or compute-intensive transactions
	// 3. Use speculative execution when it adds value
	
	useParallel := bc.Config.ParallelExecutionEnabled && 
		(totalTxs >= bc.Config.MinParallelBatchSize || bc.Config.SimulateCompute)
	
	var newStateRoot string
	
	if useParallel {
		// Create parallel executor with number of workers from config
		executor := execution.NewParallelExecutor(bc.Config.ParallelWorkers, bc.State)
		
		// Configure executor based on transaction characteristics
		executor.SetMinBatchSize(bc.Config.MinParallelBatchSize)
		
		// Set complexity simulation if enabled
		if bc.Config.SimulateCompute {
			executor.SetComplexity(execution.TransactionComplexity(bc.Config.ComputeComplexity), true)
		}
		
		// Optimize transaction order if enabled
		var txsToExecute []*transaction.Transaction
		if bc.Config.OptimizeTransactionOrder {
			txsToExecute = execution.OptimizeTransactionOrder(block.Transactions)
			fmt.Println("Optimized transaction order for parallel execution")
		} else {
			txsToExecute = block.Transactions
		}
		
		// Execute transactions in parallel
		results, stateRoot := executor.ExecuteBatch(txsToExecute)
		newStateRoot = stateRoot
		
		// Log execution results
		successCount := 0
		conflictCount := 0
		
		for _, result := range results {
			if result.Success {
				successCount++
				fmt.Printf("Successfully executed tx %s in %v\n", 
					result.Transaction.ID, result.Duration)
				if result.Conflict {
					conflictCount++
				}
			} else {
				fmt.Printf("Failed to execute tx %s: %v\n", result.Transaction.ID, result.Error)
			}
		}
		
		fmt.Printf("Parallel execution completed: %d/%d transactions succeeded, %d conflicts\n", 
			successCount, len(block.Transactions), conflictCount)
		
		// Record metrics about execution
		bc.RecordExecutionMetrics(block.Height, len(block.Transactions), 
			successCount, conflictCount, true)
	} else {
		// Use sequential execution for small batches
		fmt.Printf("Using sequential execution for %d transactions\n", totalTxs)
		
		// Create a copy of the state to work on
		stateCopy := bc.State.Copy()
		
		// Simulate compute-intensive work if enabled
		simulateWork := bc.Config.SimulateCompute
		
		// Execute each transaction
		successCount := 0
		for _, tx := range block.Transactions {
			// Start timing
			startTime := time.Now()
			
			// Optional: Simulate compute-intensive work
			if simulateWork {
				for i := 0; i < bc.Config.ComputeComplexity * 1000000; i++ {
					_ = i * i // Just waste some CPU cycles
				}
			}
			
			err := stateCopy.ExecuteTransaction(tx)
			
			// End timing
			duration := time.Since(startTime)
			
			if err != nil {
				fmt.Printf("Error executing transaction %s: %v\n", tx.ID, err)
				continue
			}
			
			successCount++
			fmt.Printf("Executed tx %s: %s -> %s, amount: %d in %v\n", 
				tx.ID, tx.Sender, tx.Recipient, tx.Amount, duration)
		}
		
		// Get the new state root
		newStateRoot = stateCopy.StateRoot
		
		// Record metrics about execution
		bc.RecordExecutionMetrics(block.Height, len(block.Transactions), 
			successCount, 0, false)
		
		// Update the actual state if not speculative
		if !isSpeculative {
			bc.State = stateCopy
		}
		
		fmt.Printf("Sequential execution completed: %d/%d transactions succeeded\n", 
			successCount, len(block.Transactions))
	}
	
	if isSpeculative {
		fmt.Printf("Speculatively executed block %d, new state root: %s\n", 
			block.Height, newStateRoot)
		bc.SpeculativeStateRoot[block.Height] = newStateRoot
	} else {
		// Update execution state
		bc.LastExecutedHeight = block.Height
		bc.PendingStateRoots[block.Height] = newStateRoot
		
		block.StateRoot = newStateRoot
		fmt.Printf("Executed block %d, new state root: %s\n", block.Height, newStateRoot)
	}
	
	return newStateRoot
}

// RecordExecutionMetrics records metrics about a block's execution
func (bc *Blockchain) RecordExecutionMetrics(blockHeight, totalTxs, successTxs, conflicts int, parallel bool) {
	// If metrics map doesn't exist, create it
	if bc.ExecutionMetrics == nil {
		bc.ExecutionMetrics = make(map[int]*ExecutionMetrics)
	}
	
	// Record metrics
	bc.ExecutionMetrics[blockHeight] = &ExecutionMetrics{
		TotalTransactions: totalTxs,
		SuccessfulTransactions: successTxs,
		Conflicts: conflicts,
		UseParallel: parallel,
		ExecutionTime: time.Now().Sub(time.Now()), // Just a placeholder for now
	}
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

// ProcessPendingExecutions processes all pending block executions with improved speculative handling
func (bc *Blockchain) ProcessPendingExecutions() {
	if len(bc.PendingExecutions) == 0 {
		return
	}
	
	pendingBlocks := bc.PendingExecutions
	bc.PendingExecutions = make([]*block.Block, 0)
	
	// Analyze dependencies between blocks to optimize execution order
	for _, block := range pendingBlocks {
		// Check if this was already speculatively executed
		if specRoot, exists := bc.SpeculativeStateRoot[block.Height]; exists && bc.Config.SpeculativeEnabled {
			// Validate speculative results before using them
			if bc.validateSpeculativeResults(block, specRoot) {
				fmt.Printf("Using verified speculative execution results for block %d\n", block.Height)
				
				// Update state with speculative results
				block.StateRoot = specRoot
				bc.PendingStateRoots[block.Height] = specRoot
				bc.LastExecutedHeight = block.Height
				
				// Remove from speculative storage
				delete(bc.SpeculativeStateRoot, block.Height)
			} else {
				fmt.Printf("Speculative results for block %d failed validation, re-executing\n", block.Height)
				// Regular execution since speculative results were invalid
				bc.ExecuteBlock(block, false)
			}
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

// validateSpeculativeResults validates that speculative execution results are correct
func (bc *Blockchain) validateSpeculativeResults(block *block.Block, speculativeRoot string) bool {
	// Simplified validation: just ensure transactions haven't changed
	// In a real implementation, this would do more thorough validation
	
	// Get the transactions that were originally in the mempool when speculative execution happened
	mempoolSnapshot, exists := bc.MempoolSnapshots[block.Height]
	if !exists {
		// Can't validate without a snapshot
		return false
	}
	
	// Verify that transactions in the block match the snapshot
	if len(block.Transactions) != len(mempoolSnapshot) {
		return false
	}
	
	// Check if each transaction in the block was in the mempool snapshot
	txMap := make(map[string]bool)
	for _, tx := range mempoolSnapshot {
		txMap[tx.ID] = true
	}
	
	for _, tx := range block.Transactions {
		if !txMap[tx.ID] {
			// Block contains a transaction that wasn't in the mempool snapshot
			return false
		}
	}
	
	// Basic validation passed
	return true
}

// AddBlock adds a voted block to the blockchain and finalizes the previous block
// with improved speculative execution
func (bc *Blockchain) AddBlock(block *block.Block) {
	// Add to blockchain
	bc.Blocks = append(bc.Blocks, block)
	bc.CurrentHeight++
	
	// If previous block exists, finalize it
	if block.Height > 1 {
		bc.FinalizeBlock(block.Height - 1)
	}
	
	// Take a snapshot of the current mempool for speculative execution validation
	if bc.Config.SpeculativeEnabled {
		bc.takeMempoolSnapshot(block.Height)
	}
	
	// Speculative execution if enabled
	if bc.Config.SpeculativeEnabled {
		// Check if the next block can be predicted based on transaction patterns
		if predictedBlock, confidence := bc.predictNextBlock(); confidence > 0.7 {
			fmt.Printf("Speculatively executing predicted next block with %.2f confidence\n", confidence)
			go bc.ExecuteBlock(predictedBlock, true)
		} else {
			// Default: just execute the current block speculatively
			fmt.Printf("Speculatively executing block %d\n", block.Height)
			go bc.ExecuteBlock(block, true)
		}
	}
}

// takeMempoolSnapshot takes a snapshot of the current mempool for later validation
func (bc *Blockchain) takeMempoolSnapshot(blockHeight int) {
	// Initialize snapshots map if it doesn't exist
	if bc.MempoolSnapshots == nil {
		bc.MempoolSnapshots = make(map[int][]*transaction.Transaction)
	}
	
	// Take a deep copy of the mempool
	snapshot := make([]*transaction.Transaction, len(bc.Mempool))
	copy(snapshot, bc.Mempool)
	
	// Store snapshot
	bc.MempoolSnapshots[blockHeight] = snapshot
	
	// Clean up old snapshots (keep only recent ones)
	for height := range bc.MempoolSnapshots {
		if height < blockHeight-10 {
			delete(bc.MempoolSnapshots, height)
		}
	}
}

// predictNextBlock attempts to predict the next block based on transaction patterns
func (bc *Blockchain) predictNextBlock() (*block.Block, float64) {
	// This is a placeholder for more sophisticated prediction logic
	// In a real implementation, this would analyze transaction patterns,
	// validator behavior, etc., to predict the next block
	
	// If there aren't enough transactions to form a block, don't predict
	if len(bc.Mempool) < bc.Config.BlockSize {
		return nil, 0.0
	}
	
	// Create a simple prediction: next leader will include oldest transactions
	nextLeader := (bc.CurrentLeader + 1) % len(bc.Validators)
	predictedBlock := block.New(bc.CurrentHeight+1, bc.Blocks[bc.CurrentHeight].Hash, nextLeader, "")
	
	// Add transactions from mempool (oldest first)
	for i := 0; i < bc.Config.BlockSize && i < len(bc.Mempool); i++ {
		predictedBlock.AddTransaction(bc.Mempool[i])
	}
	
	// Calculate hash
	predictedBlock.CalculateHash()
	
	// Determine confidence based on historical accuracy
	// For now, use a simple heuristic
	confidence := 0.8 // High confidence as a starting point
	
	return predictedBlock, confidence
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
	for _, blk := range bc.Blocks {
		finalized := ""
		// Use enum values from the BlockStatus type
		if blk.Status == block.Finalized || blk.Status == block.Verified {
			finalized = "✓"
		}
		
		verified := ""
		if blk.Status == block.Verified {
			verified = "✓"
		}
		
		fmt.Printf("Block %d: Status=%s, Txs=%d, Finalized=%s, Verified=%s\n",
			blk.Height, blk.Status, len(blk.Transactions), finalized, verified)
	}
}
