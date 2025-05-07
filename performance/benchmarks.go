// performance/benchmarks.go
package performance

import (
	"fmt"
	"time"

	"github.com/KevinKien/asynchronous-execution-simulation/block"
	"github.com/KevinKien/asynchronous-execution-simulation/execution"
	"github.com/KevinKien/asynchronous-execution-simulation/state"
	"github.com/KevinKien/asynchronous-execution-simulation/transaction"
)

// ExecutionStats contains statistics about execution performance
type ExecutionStats struct {
	NumTransactions  int
	TotalTime        time.Duration
	AvgTimePerTx     time.Duration
	Success          int
	Failed           int
	Conflicts        int
	ReexecutionCount int
}

// BenchmarkResults contains benchmark comparison results
type BenchmarkResults struct {
	SequentialStats ExecutionStats
	ParallelStats   ExecutionStats
	Speedup         float64
}

// BenchmarkExecutor benchmarks different execution strategies
type BenchmarkExecutor struct {
	State         *state.State
	Transactions  []*transaction.Transaction
	NumWorkers    int
	EnabledOptimizations bool
}

// NewBenchmarkExecutor creates a new benchmark executor
func NewBenchmarkExecutor(s *state.State, txs []*transaction.Transaction, workers int, enableOptimizations bool) *BenchmarkExecutor {
	return &BenchmarkExecutor{
		State:         s,
		Transactions:  txs,
		NumWorkers:    workers,
		EnabledOptimizations: enableOptimizations,
	}
}

// RunSequential performs sequential execution and measures performance
func (be *BenchmarkExecutor) RunSequential() ExecutionStats {
	fmt.Printf("Starting sequential execution benchmark with %d transactions...\n", 
		len(be.Transactions))
	
	stateCopy := be.State.Copy()
	stats := ExecutionStats{
		NumTransactions: len(be.Transactions),
	}
	
	startTime := time.Now()
	
	// Execute each transaction sequentially
	for _, tx := range be.Transactions {
		err := stateCopy.ExecuteTransaction(tx)
		if err != nil {
			stats.Failed++
		} else {
			stats.Success++
		}
	}
	
	stats.TotalTime = time.Since(startTime)
	if stats.NumTransactions > 0 {
		stats.AvgTimePerTx = stats.TotalTime / time.Duration(stats.NumTransactions)
	}
	
	return stats
}

// RunParallel performs parallel execution and measures performance
func (be *BenchmarkExecutor) RunParallel() ExecutionStats {
	fmt.Printf("Starting parallel execution benchmark with %d transactions using %d workers...\n", 
		len(be.Transactions), be.NumWorkers)
	
	stateCopy := be.State.Copy()
	stats := ExecutionStats{
		NumTransactions: len(be.Transactions),
	}
	
	startTime := time.Now()
	
	// Create parallel executor
	executor := execution.NewParallelExecutor(be.NumWorkers, stateCopy)
	
	// Optimize transaction order if enabled
	var txsToExecute []*transaction.Transaction
	if be.EnabledOptimizations {
		txsToExecute = execution.OptimizeTransactionOrder(be.Transactions)
		fmt.Println("Optimized transaction order for parallel execution")
	} else {
		txsToExecute = be.Transactions
	}
	
	// Execute in parallel
	results, _ := executor.ExecuteBatch(txsToExecute)
	
	// Gather stats
	for _, result := range results {
		if result.Success {
			stats.Success++
		} else {
			stats.Failed++
		}
		
		// Count conflicts (transactions that needed re-execution)
		if result.Conflict {
			stats.Conflicts++
			stats.ReexecutionCount++
		}
	}
	
	stats.TotalTime = time.Since(startTime)
	if stats.NumTransactions > 0 {
		stats.AvgTimePerTx = stats.TotalTime / time.Duration(stats.NumTransactions)
	}
	
	return stats
}

// CompareBenchmarks runs both sequential and parallel benchmarks and compares them
func (be *BenchmarkExecutor) CompareBenchmarks() BenchmarkResults {
	sequentialStats := be.RunSequential()
	fmt.Println("Sequential execution complete")
	
	// Reset state between tests
	time.Sleep(100 * time.Millisecond)
	
	parallelStats := be.RunParallel()
	fmt.Println("Parallel execution complete")
	
	// Calculate speedup
	var speedup float64
	if sequentialStats.TotalTime > 0 {
		speedup = float64(sequentialStats.TotalTime) / float64(parallelStats.TotalTime)
	}
	
	return BenchmarkResults{
		SequentialStats: sequentialStats,
		ParallelStats:   parallelStats,
		Speedup:         speedup,
	}
}

// PrintResults prints benchmark results in a formatted table
func PrintResults(results BenchmarkResults) {
	fmt.Println("\n=== Execution Performance Benchmark Results ===")
	fmt.Println("Metric                  | Sequential     | Parallel        | Improvement")
	fmt.Println("------------------------+----------------+-----------------+------------")
	fmt.Printf("Total Time              | %12v | %13v | %.2fx\n",
		results.SequentialStats.TotalTime, results.ParallelStats.TotalTime, results.Speedup)
	fmt.Printf("Avg Time Per Tx         | %12v | %13v | %.2fx\n",
		results.SequentialStats.AvgTimePerTx, results.ParallelStats.AvgTimePerTx, results.Speedup)
	fmt.Printf("Success Rate            | %5d/%-6d | %5d/%-7d | -\n",
		results.SequentialStats.Success, results.SequentialStats.NumTransactions,
		results.ParallelStats.Success, results.ParallelStats.NumTransactions)
	fmt.Printf("Conflicts               | %12s | %13d | -\n",
		"N/A", results.ParallelStats.Conflicts)
	fmt.Printf("Reexecution Count       | %12s | %13d | -\n",
		"N/A", results.ParallelStats.ReexecutionCount)
	
	// Analysis of results
	fmt.Println("\nAnalysis:")
	if results.Speedup > 1.1 {
		fmt.Printf("- Parallel execution was %.2fx faster than sequential execution\n", results.Speedup)
	} else if results.Speedup < 0.9 {
		fmt.Printf("- Parallel execution was slower than sequential execution (%.2fx)\n", results.Speedup)
		fmt.Println("  This might be due to high conflict rate or overhead of parallelization")
	} else {
		fmt.Println("- Parallel execution performance was similar to sequential execution")
	}
	
	if results.ParallelStats.Conflicts > 0 {
		conflictRate := float64(results.ParallelStats.Conflicts) / float64(results.ParallelStats.NumTransactions) * 100
		fmt.Printf("- Conflict rate: %.1f%% of transactions had conflicts\n", conflictRate)
		
		if conflictRate > 30 {
			fmt.Println("  High conflict rate suggests many interdependent transactions")
		}
	} else {
		fmt.Println("- No conflicts detected, transactions were fully parallelizable")
	}
}

// RunBlockBenchmark benchmarks execution of a complete block
func RunBlockBenchmark(b *block.Block, s *state.State, numWorkers int) BenchmarkResults {
	fmt.Printf("\nRunning benchmark for block %d with %d transactions\n", 
		b.Height, len(b.Transactions))
	
	benchmarker := NewBenchmarkExecutor(s, b.Transactions, numWorkers, true)
	results := benchmarker.CompareBenchmarks()
	
	PrintResults(results)
	return results
}
