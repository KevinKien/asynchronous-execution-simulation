// config/config.go
package config

import "time"

// Config holds all the configuration parameters for the blockchain
type Config struct {
	// Basic blockchain parameters
	NumValidators          int
	NumTransactions        int
	BlockSize              int
	ExecutionDelay         int
	BlockInterval          time.Duration
	QuorumPercentage       float64
	
	// Execution strategy parameters
	SpeculativeEnabled     bool
	ParallelExecutionEnabled bool
	ParallelWorkers        int
	OptimizeTransactionOrder bool
	MinParallelBatchSize   int  // Min batch size to use parallel execution
	
	// Compute simulation parameters
	SimulateCompute        bool  // Whether to simulate compute-intensive transactions
	ComputeComplexity      int   // Complexity level for simulated computation
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		// Basic parameters
		NumValidators:          5,
		NumTransactions:        20,
		BlockSize:              5,
		ExecutionDelay:         3, // D blocks execution delay
		BlockInterval:          1 * time.Second,
		QuorumPercentage:       0.67, // 2/3 of validators
		
		// Execution parameters
		SpeculativeEnabled:     true,
		ParallelExecutionEnabled: true,
		ParallelWorkers:        4,
		OptimizeTransactionOrder: true,
		MinParallelBatchSize:   20, // Only use parallel for batches >= 20 transactions
		
		// Compute simulation
		SimulateCompute:        false,
		ComputeComplexity:      1, // Simple computation by default
	}
}

// NewConfig creates a new configuration with the specified parameters
func NewConfig(numValidators, numTransactions, blockSize, executionDelay int,
	blockInterval time.Duration, quorumPercentage float64, speculativeEnabled bool,
	parallelEnabled bool, workers int, optimizeOrder bool, minBatchSize int,
	simulateCompute bool, computeComplexity int) *Config {
	
	return &Config{
		NumValidators:          numValidators,
		NumTransactions:        numTransactions,
		BlockSize:              blockSize,
		ExecutionDelay:         executionDelay,
		BlockInterval:          blockInterval,
		QuorumPercentage:       quorumPercentage,
		SpeculativeEnabled:     speculativeEnabled,
		ParallelExecutionEnabled: parallelEnabled,
		ParallelWorkers:        workers,
		OptimizeTransactionOrder: optimizeOrder,
		MinParallelBatchSize:   minBatchSize,
		SimulateCompute:        simulateCompute,
		ComputeComplexity:      computeComplexity,
	}
}

// ConfigureForSmallTransactions sets configuration for simple, fast transactions
func (c *Config) ConfigureForSmallTransactions() {
	c.ParallelExecutionEnabled = false // sequential is faster for small transactions
	c.MinParallelBatchSize = 50        // high threshold to avoid parallel overhead
	c.SimulateCompute = false
	c.ComputeComplexity = 1
}

// ConfigureForMediumTransactions sets configuration for medium-complexity transactions
func (c *Config) ConfigureForMediumTransactions() {
	c.ParallelExecutionEnabled = true
	c.MinParallelBatchSize = 20
	c.SimulateCompute = true
	c.ComputeComplexity = 5
	c.OptimizeTransactionOrder = true
}

// ConfigureForComplexTransactions sets configuration for compute-intensive transactions
func (c *Config) ConfigureForComplexTransactions() {
	c.ParallelExecutionEnabled = true
	c.MinParallelBatchSize = 10 // low threshold to benefit from parallelism
	c.SimulateCompute = true
	c.ComputeComplexity = 20
	c.OptimizeTransactionOrder = true
}
