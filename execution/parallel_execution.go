// execution/parallel_execution.go
package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/KevinKien/asynchronous-execution-simulation/state"
	"github.com/KevinKien/asynchronous-execution-simulation/transaction"
)

// ReadSet represents all state reads by a transaction
type ReadSet map[string]interface{}

// WriteSet represents all state writes by a transaction
type WriteSet map[string]interface{}

// TransactionResult represents the result of executing a transaction
type TransactionResult struct {
	Transaction *transaction.Transaction
	ReadSet     ReadSet
	WriteSet    WriteSet
	Success     bool
	Error       error
}

// ConflictDetector detects conflicts between transactions
type ConflictDetector struct {
	mutex sync.Mutex
	reads map[string]int // Which transaction read this key
	writes map[string]int // Which transaction wrote this key
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		reads: make(map[string]int),
		writes: make(map[string]int),
	}
}

// CheckConflict checks if a transaction's read/write set conflicts with previous transactions
// Returns true if there's a conflict
func (cd *ConflictDetector) CheckConflict(txIndex int, readSet ReadSet, writeSet WriteSet) bool {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()
	
	// Check if this transaction's reads conflict with previous writes
	for key := range readSet {
		if writerIdx, exists := cd.writes[key]; exists && writerIdx < txIndex {
			// This transaction reads a key that was written by an earlier transaction
			return true
		}
	}
	
	// Check if this transaction's writes conflict with previous reads or writes
	for key := range writeSet {
		if readerIdx, exists := cd.reads[key]; exists && readerIdx < txIndex {
			// This transaction writes a key that was read by an earlier transaction
			return true
		}
		if writerIdx, exists := cd.writes[key]; exists && writerIdx < txIndex {
			// This transaction writes a key that was written by an earlier transaction
			return true
		}
	}
	
	// No conflicts, record this transaction's reads and writes
	for key := range readSet {
		cd.reads[key] = txIndex
	}
	for key := range writeSet {
		cd.writes[key] = txIndex
	}
	
	return false
}

// ParallelExecutor handles parallel execution of transactions
type ParallelExecutor struct {
	NumWorkers int
	State      *state.State
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(numWorkers int, s *state.State) *ParallelExecutor {
	return &ParallelExecutor{
		NumWorkers: numWorkers,
		State:      s,
	}
}

// ExecuteBatch executes a batch of transactions in parallel with conflict detection
func (pe *ParallelExecutor) ExecuteBatch(transactions []*transaction.Transaction) ([]*TransactionResult, string) {
	if len(transactions) == 0 {
		return []*TransactionResult{}, pe.State.StateRoot
	}
	
	// Create work channel and result channel
	workChan := make(chan int, len(transactions))
	resultChan := make(chan *TransactionResult, len(transactions))
	
	// Create worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < pe.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for txIndex := range workChan {
				// Execute transaction optimistically
				result := pe.executeTransaction(transactions[txIndex], pe.State.Copy())
				result.Transaction = transactions[txIndex]
				resultChan <- result
			}
		}()
	}
	
	// Send work to workers
	for i := range transactions {
		workChan <- i
	}
	close(workChan)
	
	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	results := make([]*TransactionResult, len(transactions))
	for result := range resultChan {
		txIndex := -1
		for i, tx := range transactions {
			if tx.ID == result.Transaction.ID {
				txIndex = i
				break
			}
		}
		if txIndex >= 0 {
			results[txIndex] = result
		}
	}
	
	// Apply results to state in original transaction order
	conflictDetector := NewConflictDetector()
	var finalResults []*TransactionResult
	stateCopy := pe.State.Copy()
	
	for i, result := range results {
		if !result.Success {
			fmt.Printf("Transaction %s failed execution: %v\n", 
				result.Transaction.ID, result.Error)
			finalResults = append(finalResults, result)
			continue
		}
		
		// Check for conflicts
		hasConflict := conflictDetector.CheckConflict(i, result.ReadSet, result.WriteSet)
		if hasConflict {
			fmt.Printf("Transaction %s has conflicts, re-executing sequentially\n", 
				result.Transaction.ID)
			
			// Re-execute sequentially with current state
			newResult := pe.executeTransaction(result.Transaction, stateCopy)
			newResult.Transaction = result.Transaction
			if newResult.Success {
				// Apply changes to state
				pe.applyChanges(stateCopy, newResult.WriteSet)
			}
			finalResults = append(finalResults, newResult)
		} else {
			// No conflicts, apply changes to state
			pe.applyChanges(stateCopy, result.WriteSet)
			finalResults = append(finalResults, result)
		}
	}
	
	// Calculate new state root
	stateCopy.CalculateStateRoot()
	newStateRoot := stateCopy.StateRoot
	
	// Update actual state
	*pe.State = *stateCopy
	
	return finalResults, newStateRoot
}

// executeTransaction executes a single transaction and returns read/write sets
func (pe *ParallelExecutor) executeTransaction(tx *transaction.Transaction, stateCopy *state.State) *TransactionResult {
	result := &TransactionResult{
		Transaction: tx,
		ReadSet:     make(ReadSet),
		WriteSet:    make(WriteSet),
		Success:     false,
	}
	
	// Get sender account
	sender, exists := stateCopy.GetAccount(tx.Sender)
	if !exists {
		result.Error = fmt.Errorf("sender account %s not found", tx.Sender)
		return result
	}
	
	// Record read of sender
	result.ReadSet[fmt.Sprintf("account:%s:balance", tx.Sender)] = sender.Balance
	result.ReadSet[fmt.Sprintf("account:%s:nonce", tx.Sender)] = sender.Nonce
	
	// Verify nonce
	if tx.Nonce != sender.Nonce {
		result.Error = fmt.Errorf("invalid nonce for tx %s: expected %d, got %d", 
			tx.ID, sender.Nonce, tx.Nonce)
		return result
	}
	
	// Verify balance
	if sender.Balance < tx.Amount {
		result.Error = fmt.Errorf("insufficient balance for tx %s", tx.ID)
		return result
	}
	
	// Get or create recipient account
	recipient, exists := stateCopy.GetAccount(tx.Recipient)
	if exists {
		result.ReadSet[fmt.Sprintf("account:%s:balance", tx.Recipient)] = recipient.Balance
	} else {
		// Create new account if it doesn't exist
		recipient = &state.Account{
			Address: tx.Recipient,
			Balance: 0,
			Nonce:   0,
		}
		stateCopy.Accounts[tx.Recipient] = recipient
	}
	
	// Record write operations
	result.WriteSet[fmt.Sprintf("account:%s:balance", tx.Sender)] = sender.Balance - tx.Amount
	result.WriteSet[fmt.Sprintf("account:%s:nonce", tx.Sender)] = sender.Nonce + 1
	result.WriteSet[fmt.Sprintf("account:%s:balance", tx.Recipient)] = recipient.Balance + tx.Amount
	
	result.Success = true
	return result
}

// applyChanges applies write set changes to the state
func (pe *ParallelExecutor) applyChanges(stateCopy *state.State, writeSet WriteSet) {
	for key, value := range writeSet {
		// Parse key format "account:address:field"
		var address, field string
		fmt.Sscanf(key, "account:%s:%s", &address, &field)
		
		account, exists := stateCopy.GetAccount(address)
		if !exists {
			account = &state.Account{
				Address: address,
				Balance: 0,
				Nonce:   0,
			}
			stateCopy.Accounts[address] = account
		}
		
		// Update field
		switch field {
		case "balance":
			if balance, ok := value.(int); ok {
				account.Balance = balance
			}
		case "nonce":
			if nonce, ok := value.(int); ok {
				account.Nonce = nonce
			}
		}
	}
}

// CalculateTransactionDependencies analyzes transactions to find dependencies
func CalculateTransactionDependencies(transactions []*transaction.Transaction) map[int][]int {
	// Create a map of address -> transactions that use it
	addressUsage := make(map[string][]int)
	
	// Track which transaction affects which addresses
	for i, tx := range transactions {
		// Sender
		addressUsage[tx.Sender] = append(addressUsage[tx.Sender], i)
		// Recipient
		addressUsage[tx.Recipient] = append(addressUsage[tx.Recipient], i)
	}
	
	// Create dependency map: tx index -> indices of txs it depends on
	dependencies := make(map[int][]int)
	
	// For each transaction, find its dependencies
	for i, tx := range transactions {
		deps := make(map[int]bool) // To avoid duplicates
		
		// Check sender dependencies (all transactions involving this sender with lower indices)
		for _, depIdx := range addressUsage[tx.Sender] {
			if depIdx < i {
				deps[depIdx] = true
			}
		}
		
		// Check recipient dependencies
		for _, depIdx := range addressUsage[tx.Recipient] {
			if depIdx < i {
				deps[depIdx] = true
			}
		}
		
		// Convert to slice
		depList := make([]int, 0)
		for depIdx := range deps {
			depList = append(depList, depIdx)
		}
		dependencies[i] = depList
	}
	
	return dependencies
}

// OptimizeTransactionOrder reorders transactions to maximize parallelism
func OptimizeTransactionOrder(transactions []*transaction.Transaction) []*transaction.Transaction {
	if len(transactions) <= 1 {
		return transactions
	}
	
	// Calculate dependencies
	dependencies := CalculateTransactionDependencies(transactions)
	
	// Group transactions by their sender (to maintain nonce order)
	senderGroups := make(map[string][]*transaction.Transaction)
	for _, tx := range transactions {
		senderGroups[tx.Sender] = append(senderGroups[tx.Sender], tx)
	}
	
	// Sort each sender group by nonce
	for sender, txs := range senderGroups {
		sorted := make([]*transaction.Transaction, len(txs))
		copy(sorted, txs)
		// Simple bubble sort by nonce
		for i := 0; i < len(sorted)-1; i++ {
			for j := 0; j < len(sorted)-i-1; j++ {
				if sorted[j].Nonce > sorted[j+1].Nonce {
					sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
				}
			}
		}
		senderGroups[sender] = sorted
	}
	
	// Flatten groups back into a single list, trying to interleave different senders
	// to maximize parallelism (simple greedy approach)
	var result []*transaction.Transaction
	senderQueue := make([]string, 0)
	
	// Initialize queue with all senders
	for sender := range senderGroups {
		if len(senderGroups[sender]) > 0 {
			senderQueue = append(senderQueue, sender)
		}
	}
	
	// Round-robin between senders
	for len(senderQueue) > 0 {
		sender := senderQueue[0]
		senderQueue = senderQueue[1:]
		
		if len(senderGroups[sender]) > 0 {
			// Take the first transaction from this sender
			tx := senderGroups[sender][0]
			senderGroups[sender] = senderGroups[sender][1:]
			result = append(result, tx)
			
			// Put sender back in queue if it has more transactions
			if len(senderGroups[sender]) > 0 {
				senderQueue = append(senderQueue, sender)
			}
		}
	}
	
	return result
}
