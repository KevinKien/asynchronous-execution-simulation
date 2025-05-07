// config/config.go
package config

import "time"

// Config holds all the configuration parameters for the blockchain
type Config struct {
	NumValidators          int
	NumTransactions        int
	BlockSize              int
	ExecutionDelay         int
	BlockInterval          time.Duration
	QuorumPercentage       float64
	SpeculativeEnabled     bool
	ParallelExecutionEnabled bool
	ParallelWorkers        int
	OptimizeTransactionOrder bool
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		NumValidators:          5,
		NumTransactions:        20,
		BlockSize:              5,
		ExecutionDelay:         3, // D blocks execution delay
		BlockInterval:          1 * time.Second,
		QuorumPercentage:       0.67, // 2/3 of validators
		SpeculativeEnabled:     true,
		ParallelExecutionEnabled: true,
		ParallelWorkers:        4,
		OptimizeTransactionOrder: true,
	}
}

// NewConfig creates a new configuration with the specified parameters
func NewConfig(numValidators, numTransactions, blockSize, executionDelay int,
	blockInterval time.Duration, quorumPercentage float64, speculativeEnabled bool) *Config {
	
	return &Config{
		NumValidators:      numValidators,
		NumTransactions:    numTransactions,
		BlockSize:          blockSize,
		ExecutionDelay:     executionDelay,
		BlockInterval:      blockInterval,
		QuorumPercentage:   quorumPercentage,
		SpeculativeEnabled: speculativeEnabled,
	}
}
