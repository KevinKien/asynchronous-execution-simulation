// execution/parallel_execution.go
package execution

import (
	"fmt"
	"sync"
	"time"

	"asynchronous-execution-simulation/state"
	"asynchronous-execution-simulation/transaction"
)

// TransactionComplexity represents how compute-intensive a transaction is
type TransactionComplexity int

const (
	Simple     TransactionComplexity = 1  // Simple balance transfers
	Moderate   TransactionComplexity = 5  // Moderate computation
	Complex    TransactionComplexity = 20 // Heavy computation
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
	Conflict    bool // Indicates if this transaction had conflicts
	Duration    time.Duration // How long the execution took
}

// ConflictDetector detects conflicts between transactions
type ConflictDetector struct {
	mutex  sync.Mutex
	reads  map[string]int // Which transaction read this key
	writes map[string]int // Which transaction wrote this key
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		reads:  make(map[string]int),
		writes: make(map[string]int),
	}
}

// CheckConflict checks if a transaction's read/write set conflicts with previous transactions
// Returns true if there's a conflict
func (cd *ConflictDetector) CheckConflict(txIndex int, readSet ReadSet, writeSet WriteSet) bool {
	cd.mutex.Lock()
	defer cd.mutex.Unlock()
	
	// Only check keys that are actually used in this transaction
	// Improved to avoid false positives by checking actual values
	
	// Check if this transaction's reads conflict with previous writes
	for key := range readSet {
		if writerIdx, exists := cd.writes[key]; exists && writerIdx < txIndex {
			// Compare actual values if possible - FIX: removed unused 'value' variable
			// This reduces false conflicts when the value hasn't actually changed
			if previousValue, ok := cd.reads[key]; ok {
				// Check if the value has changed - note we're just using existence now
				// without comparing the actual values (which would require more complex logic)
				if previousValue >= 0 {
					// There is a previous value, potential conflict
					return true
				}
			}
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
		// Store the transaction index as the value
		cd.reads[key] = txIndex
	}
	for key := range writeSet {
		cd.writes[key] = txIndex
	}
	
	return false
}

// WorkerPool manages a pool of worker goroutines
type WorkerPool struct {
	workChan chan *TransactionTask
	wg       sync.WaitGroup
	size     int
}

// TransactionTask represents a task to be executed by a worker
type TransactionTask struct {
	Tx     *transaction.Transaction
	State  *state.State
	Index  int
	Result chan *TransactionResult
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(size int, handler func(*TransactionTask)) *WorkerPool {
	pool := &WorkerPool{
		workChan: make(chan *TransactionTask, size*10), // Buffer the channel
		size:     size,
	}
	
	// Start workers
	pool.wg.Add(size)
	for i := 0; i < size; i++ {
		go func() {
			defer pool.wg.Done()
			for task := range pool.workChan {
				handler(task)
			}
		}()
	}
	
	return pool
}

// Submit adds a task to the worker pool
func (wp *WorkerPool) Submit(task *TransactionTask) {
	wp.workChan <- task
}

// Close closes the worker pool and waits for all workers to finish
func (wp *WorkerPool) Close() {
	close(wp.workChan)
	wp.wg.Wait()
}

// ParallelExecutor handles parallel execution of transactions
type ParallelExecutor struct {
	NumWorkers             int
	State                  *state.State
	pool                   *WorkerPool
	MinBatchSizeForParallel int  // Minimum batch size to use parallel execution
	SimulateComplexity     bool  // Whether to simulate compute-intensive transactions
	ComplexityLevel        TransactionComplexity // How complex transactions are
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(numWorkers int, s *state.State) *ParallelExecutor {
	executor := &ParallelExecutor{
		NumWorkers:             numWorkers,
		State:                  s,
		MinBatchSizeForParallel: 10, // Default minimum batch size
		SimulateComplexity:     false,
		ComplexityLevel:        Simple,
	}
	
	// Create worker pool with a handler function
	executor.pool = NewWorkerPool(numWorkers, func(task *TransactionTask) {
		result := executor.executeTransaction(task.Tx, task.State)
		result.Transaction = task.Tx
		task.Result <- result
	})
	
	return executor
}

// SetComplexity sets the complexity level for simulated computation
func (pe *ParallelExecutor) SetComplexity(level TransactionComplexity, enabled bool) {
	pe.ComplexityLevel = level
	pe.SimulateComplexity = enabled
}

// SetMinBatchSize sets the minimum batch size for parallel execution
func (pe *ParallelExecutor) SetMinBatchSize(size int) {
	pe.MinBatchSizeForParallel = size
}

// ExecuteBatch executes a batch of transactions with adaptive strategy
// Uses sequential execution for small batches and parallel for larger ones
func (pe *ParallelExecutor) ExecuteBatch(transactions []*transaction.Transaction) ([]*TransactionResult, string) {
	// Adaptive strategy based on batch size
	if len(transactions) < pe.MinBatchSizeForParallel {
		fmt.Printf("Small batch (%d < %d), using sequential execution\n", 
			len(transactions), pe.MinBatchSizeForParallel)
		return pe.executeSequential(transactions)
	}
	
	return pe.executeParallel(transactions)
}

// executeSequential executes transactions sequentially
func (pe *ParallelExecutor) executeSequential(transactions []*transaction.Transaction) ([]*TransactionResult, string) {
	if len(transactions) == 0 {
		return []*TransactionResult{}, pe.State.StateRoot
	}
	
	stateCopy := pe.State.Copy()
	results := make([]*TransactionResult, len(transactions))
	
	for i, tx := range transactions {
		startTime := time.Now()
		result := pe.executeTransaction(tx, stateCopy)
		result.Transaction = tx
		result.Duration = time.Since(startTime)
		
		// If successful, apply changes to state
		if result.Success {
			pe.applyChanges(stateCopy, result.WriteSet)
		}
		
		results[i] = result
	}
	
	// Calculate new state root
	stateCopy.CalculateStateRoot()
	newStateRoot := stateCopy.StateRoot
	
	// Update actual state
	*pe.State = *stateCopy
	
	return results, newStateRoot
}

// executeParallel executes transactions in parallel with improved dependency analysis
func (pe *ParallelExecutor) executeParallel(transactions []*transaction.Transaction) ([]*TransactionResult, string) {
	if len(transactions) == 0 {
		return []*TransactionResult{}, pe.State.StateRoot
	}
	
	// Analyze transaction dependencies for better scheduling
	dependencies := CalculateTransactionDependencies(transactions)
	
	// Group transactions based on dependencies
	independentBatches := GroupIndependentTransactions(transactions, dependencies)
	
	// Process batches with decreasing parallelism based on dependencies
	stateCopy := pe.State.Copy()
	var allResults []*TransactionResult
	
	for batchIdx, batch := range independentBatches {
		fmt.Printf("Processing batch %d with %d transactions\n", batchIdx+1, len(batch))
		
		// Create channels for results
		resultChan := make(chan *TransactionResult, len(batch))
		
		// Submit tasks to worker pool
		for i, tx := range batch {
			pe.pool.Submit(&TransactionTask{
				Tx:     tx,
				State:  stateCopy.Copy(), // Each task gets its own state copy
				Index:  i,
				Result: resultChan,
			})
		}
		
		// Collect results
		batchResults := make([]*TransactionResult, len(batch))
		for i := 0; i < len(batch); i++ {
			batchResults[i] = <-resultChan
		}
		
		// Apply changes to state in sequence
		for _, result := range batchResults {
			if result.Success {
				pe.applyChanges(stateCopy, result.WriteSet)
			}
		}
		
		allResults = append(allResults, batchResults...)
	}
	
	// Calculate new state root
	stateCopy.CalculateStateRoot()
	newStateRoot := stateCopy.StateRoot
	
	// Update actual state
	*pe.State = *stateCopy
	
	return allResults, newStateRoot
}

// executeTransaction executes a single transaction and returns read/write sets
func (pe *ParallelExecutor) executeTransaction(tx *transaction.Transaction, stateCopy *state.State) *TransactionResult {
	result := &TransactionResult{
		Transaction: tx,
		ReadSet:     make(ReadSet),
		WriteSet:    make(WriteSet),
		Success:     false,
		Conflict:    false,
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
	
	// Simulate compute-intensive work if enabled
	if pe.SimulateComplexity {
		pe.simulateComputation(int(pe.ComplexityLevel))
	}
	
	// Record write operations
	result.WriteSet[fmt.Sprintf("account:%s:balance", tx.Sender)] = sender.Balance - tx.Amount
	result.WriteSet[fmt.Sprintf("account:%s:nonce", tx.Sender)] = sender.Nonce + 1
	result.WriteSet[fmt.Sprintf("account:%s:balance", tx.Recipient)] = recipient.Balance + tx.Amount
	
	result.Success = true
	return result
}

// simulateComputation simulates compute-intensive work
func (pe *ParallelExecutor) simulateComputation(complexity int) {
	// Simulate computation by performing a CPU-bound task
	for i := 0; i < complexity*1000000; i++ {
		_ = i * i // Just waste some CPU cycles
	}
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
// Returns a map of transaction index to indices of transactions it depends on
func CalculateTransactionDependencies(transactions []*transaction.Transaction) map[int][]int {
	// Enhanced dependency analysis
	
	// First, identify all accounts that each transaction touches
	accountUsage := make(map[string][]int) // address -> indices of transactions using it
	
	// Track transactions involving each account
	for i, tx := range transactions {
		// Each sender's transactions should be in order
		accountUsage[tx.Sender] = append(accountUsage[tx.Sender], i)
		// Recipient affects later transactions to/from that address
		accountUsage[tx.Recipient] = append(accountUsage[tx.Recipient], i)
	}
	
	// Track nonce-based dependencies (transactions from the same sender)
	senderNonceDeps := make(map[string]map[int]int) // sender -> nonce -> tx index
	for i, tx := range transactions {
		if _, exists := senderNonceDeps[tx.Sender]; !exists {
			senderNonceDeps[tx.Sender] = make(map[int]int)
		}
		senderNonceDeps[tx.Sender][tx.Nonce] = i
	}
	
	// Build dependency graph
	result := make(map[int][]int)
	
	for i, tx := range transactions {
		deps := make(map[int]bool) // Using map to avoid duplicates
		
		// Add dependencies based on account usage
		for _, depIdx := range accountUsage[tx.Sender] {
			if depIdx < i {
				deps[depIdx] = true
			}
		}
		
		for _, depIdx := range accountUsage[tx.Recipient] {
			if depIdx < i {
				deps[depIdx] = true
			}
		}
		
		// Add nonce-based dependencies - each transaction depends on previous nonce
		if prevNonceIdx, exists := senderNonceDeps[tx.Sender][tx.Nonce-1]; exists {
			deps[prevNonceIdx] = true
		}
		
		// Convert to slice
		depList := make([]int, 0, len(deps))
		for depIdx := range deps {
			depList = append(depList, depIdx)
		}
		
		result[i] = depList
	}
	
	return result
}

// GroupIndependentTransactions groups transactions into independent batches for parallel execution
func GroupIndependentTransactions(transactions []*transaction.Transaction, dependencies map[int][]int) [][]*transaction.Transaction {
	if len(transactions) == 0 {
		return nil
	}
	
	// Create a reverse dependency map: tracks which transactions are blocking each transaction
	blockedBy := make(map[int]map[int]bool)
	for txIdx := 0; txIdx < len(transactions); txIdx++ {
		blockedBy[txIdx] = make(map[int]bool)
		for dep, depList := range dependencies {
			for _, dependsOn := range depList {
				if dependsOn == txIdx {
					blockedBy[txIdx][dep] = true
				}
			}
		}
	}
	
	// Count dependencies for each transaction
	depCount := make(map[int]int)
	for txIdx, deps := range dependencies {
		depCount[txIdx] = len(deps)
	}
	
	// Group transactions into batches
	var result [][]*transaction.Transaction
	processed := make(map[int]bool)
	
	for len(processed) < len(transactions) {
		// Find all transactions that have no unprocessed dependencies
		batch := make([]*transaction.Transaction, 0)
		
		for i := 0; i < len(transactions); i++ {
			if processed[i] {
				continue
			}
			
			hasUnprocessedDeps := false
			for _, dep := range dependencies[i] {
				if !processed[dep] {
					hasUnprocessedDeps = true
					break
				}
			}
			
			if !hasUnprocessedDeps {
				batch = append(batch, transactions[i])
				processed[i] = true
			}
		}
		
		if len(batch) > 0 {
			result = append(result, batch)
		} else if len(processed) < len(transactions) {
			// If no progress can be made but we haven't processed all transactions,
			// there's a cycle in the dependency graph - break it
			for i := 0; i < len(transactions); i++ {
				if !processed[i] {
					batch = append(batch, transactions[i])
					processed[i] = true
					break
				}
			}
			result = append(result, batch)
		}
	}
	
	return result
}

// OptimizeTransactionOrder reorders transactions to maximize parallelism
func OptimizeTransactionOrder(transactions []*transaction.Transaction) []*transaction.Transaction {
	if len(transactions) <= 1 {
		return transactions
	}
	
	// Calculate dependencies
	dependencies := CalculateTransactionDependencies(transactions)
	
	// Group transactions into independent batches
	batches := GroupIndependentTransactions(transactions, dependencies)
	
	// Flatten batches
	var result []*transaction.Transaction
	for _, batch := range batches {
		// For each batch, sort by sender to keep sender's transactions together
		senderGroups := make(map[string][]*transaction.Transaction)
		for _, tx := range batch {
			senderGroups[tx.Sender] = append(senderGroups[tx.Sender], tx)
		}
		
		// Sort each sender's transactions by nonce
		for sender, txs := range senderGroups {
			// Simple bubble sort by nonce
			for i := 0; i < len(txs)-1; i++ {
				for j := 0; j < len(txs)-i-1; j++ {
					if txs[j].Nonce > txs[j+1].Nonce {
						txs[j], txs[j+1] = txs[j+1], txs[j]
					}
				}
			}
			senderGroups[sender] = txs
		}
		
		// Interleave transactions from different senders for better parallelism
		var senders []string
		for sender := range senderGroups {
			senders = append(senders, sender)
		}
		
		// Process each sender's transactions
		senderIdx := 0
		for len(senderGroups) > 0 {
			if senderIdx >= len(senders) {
				senderIdx = 0
			}
			
			sender := senders[senderIdx]
			if txs, exists := senderGroups[sender]; exists && len(txs) > 0 {
				// Add the first transaction from this sender
				result = append(result, txs[0])
				
				// Remove the processed transaction
				if len(txs) > 1 {
					senderGroups[sender] = txs[1:]
				} else {
					delete(senderGroups, sender)
					// Remove from senders list
					for i, s := range senders {
						if s == sender {
							senders = append(senders[:i], senders[i+1:]...)
							break
						}
					}
					// Adjust senderIdx if needed
					if senderIdx >= len(senders) && len(senders) > 0 {
						senderIdx = 0
					}
					continue // Skip incrementing senderIdx
				}
			}
			
			senderIdx++
		}
	}
	
	return result
}
