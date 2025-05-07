// performance/benchmarks.go
package performance

import (
	"fmt"
	"time"

	"asynchronous-execution-simulation/block"
	"asynchronous-execution-simulation/execution"
	"asynchronous-execution-simulation/state"
	"asynchronous-execution-simulation/transaction"
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
	TxTimes          []time.Duration // Individual transaction times
}

// BenchmarkResults contains benchmark comparison results
type BenchmarkResults struct {
	SequentialStats ExecutionStats
	ParallelStats   ExecutionStats
	Speedup         float64
}

// BenchmarkExecutor benchmarks different execution strategies
type BenchmarkExecutor struct {
	State                *state.State
	Transactions         []*transaction.Transaction
	NumWorkers           int
	EnabledOptimizations bool
	SimulateCompute      bool
	ComputeComplexity    int
}

// NewBenchmarkExecutor creates a new benchmark executor
func NewBenchmarkExecutor(s *state.State, txs []*transaction.Transaction, workers int, enableOptimizations bool) *BenchmarkExecutor {
	return &BenchmarkExecutor{
		State:                s,
		Transactions:         txs,
		NumWorkers:           workers,
		EnabledOptimizations: enableOptimizations,
		SimulateCompute:      false, // Default
		ComputeComplexity:    1,     // Default
	}
}

// SetComputeSimulation configures simulation of compute-intensive transactions
func (be *BenchmarkExecutor) SetComputeSimulation(enabled bool, complexity int) {
	be.SimulateCompute = enabled
	be.ComputeComplexity = complexity
}

// simulateComputation simulates compute-intensive work
func simulateComputation(complexity int) {
	// Simulate computation by performing a CPU-bound task
	for i := 0; i < complexity*1000000; i++ {
		_ = i * i // Just waste some CPU cycles
	}
}

// RunSequential performs sequential execution and measures performance
func (be *BenchmarkExecutor) RunSequential() ExecutionStats {
	fmt.Printf("Starting sequential execution benchmark with %d transactions...\n", 
		len(be.Transactions))
	
	stateCopy := be.State.Copy()
	stats := ExecutionStats{
		NumTransactions: len(be.Transactions),
		TxTimes:         make([]time.Duration, len(be.Transactions)),
	}
	
	startTime := time.Now()
	
	// Execute each transaction sequentially
	for i, tx := range be.Transactions {
		txStartTime := time.Now()
		
		// Simulate compute-intensive work if enabled
		if be.SimulateCompute {
			simulateComputation(be.ComputeComplexity)
		}
		
		err := stateCopy.ExecuteTransaction(tx)
		
		// Record transaction time
		stats.TxTimes[i] = time.Since(txStartTime)
		
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
		TxTimes:         make([]time.Duration, len(be.Transactions)),
	}
	
	startTime := time.Now()
	
	// Create parallel executor
	executor := execution.NewParallelExecutor(be.NumWorkers, stateCopy)
	
	// Configure compute simulation
	if be.SimulateCompute {
		executor.SetComplexity(execution.TransactionComplexity(be.ComputeComplexity), true)
	}
	
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
	for i, result := range results {
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
		
		// Record execution time if available
		if i < len(stats.TxTimes) {
			stats.TxTimes[i] = result.Duration
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
	// Run sequential benchmark
	sequentialStats := be.RunSequential()
	fmt.Println("Sequential execution complete")
	
	// Reset state between tests
	time.Sleep(100 * time.Millisecond)
	
	// Run parallel benchmark
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

// analyzeTxTimes calculates min, max, and median transaction times
func analyzeTxTimes(times []time.Duration) (min, max, median time.Duration) {
	if len(times) == 0 {
		return 0, 0, 0
	}
	
	// Make a copy to avoid modifying the original
	timesCopy := make([]time.Duration, len(times))
	copy(timesCopy, times)
	
	// Sort the times
	for i := 0; i < len(timesCopy)-1; i++ {
		for j := 0; j < len(timesCopy)-i-1; j++ {
			if timesCopy[j] > timesCopy[j+1] {
				timesCopy[j], timesCopy[j+1] = timesCopy[j+1], timesCopy[j]
			}
		}
	}
	
	// Find min, max, median
	min = timesCopy[0]
	max = timesCopy[len(timesCopy)-1]
	
	// Calculate median
	middle := len(timesCopy) / 2
	if len(timesCopy)%2 == 0 {
		median = (timesCopy[middle-1] + timesCopy[middle]) / 2
	} else {
		median = timesCopy[middle]
	}
	
	return min, max, median
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
	if results.Speedup > 1.5 {
		fmt.Printf("- Parallel execution was %.2fx faster than sequential execution\n", results.Speedup)
		
		if results.ParallelStats.Conflicts > 0 {
			conflictRate := float64(results.ParallelStats.Conflicts) / float64(results.ParallelStats.NumTransactions) * 100
			fmt.Printf("  Even with a %.1f%% conflict rate, parallelism provided significant benefits\n", conflictRate)
		} else {
			fmt.Println("  With no conflicts detected, transactions were fully parallelizable")
		}
		
		if results.SequentialStats.AvgTimePerTx > 1*time.Millisecond {
			fmt.Println("  Transactions were compute-intensive enough to benefit from parallelism")
		}
	} else if results.Speedup > 1.0 {
		fmt.Printf("- Parallel execution was slightly faster (%.2fx) than sequential execution\n", results.Speedup)
		
		if results.ParallelStats.Conflicts > 0 {
			conflictRate := float64(results.ParallelStats.Conflicts) / float64(results.ParallelStats.NumTransactions) * 100
			fmt.Printf("  A %.1f%% conflict rate limited potential speedup\n", conflictRate)
		}
	} else {
		fmt.Printf("- Parallel execution was slower than sequential execution (%.2fx)\n", results.Speedup)
		
		if results.ParallelStats.Conflicts > 0 {
			conflictRate := float64(results.ParallelStats.Conflicts) / float64(results.ParallelStats.NumTransactions) * 100
			fmt.Printf("  High conflict rate (%.1f%%) significantly reduced parallel performance\n", conflictRate)
		}
		
		if results.SequentialStats.AvgTimePerTx < 10*time.Microsecond {
			fmt.Println("  Transactions were too simple to benefit from parallelism")
			fmt.Println("  Overhead of parallelization exceeded execution time savings")
		}
	}
	
	// Transaction time distribution analysis
	if len(results.SequentialStats.TxTimes) > 0 {
		// Calculate min/max/median times
		seqMin, seqMax, seqMedian := analyzeTxTimes(results.SequentialStats.TxTimes)
		parMin, parMax, parMedian := analyzeTxTimes(results.ParallelStats.TxTimes)
		
		fmt.Println("\nTransaction Time Distribution:")
		fmt.Printf("- Sequential: min=%v, median=%v, max=%v\n", seqMin, seqMedian, seqMax)
		fmt.Printf("- Parallel:   min=%v, median=%v, max=%v\n", parMin, parMedian, parMax)
	}
}

// RunBlockBenchmark benchmarks execution of a complete block
func RunBlockBenchmark(b *block.Block, s *state.State, numWorkers int) BenchmarkResults {
	fmt.Printf("\nRunning benchmark for block %d with %d transactions\n", 
		b.Height, len(b.Transactions))
	
	benchmarker := NewBenchmarkExecutor(s, b.Transactions, numWorkers, true)
	
	// Detect if transactions might be compute-intensive
	// In a real system, we'd have better heuristics
	if len(b.Transactions) > 30 {
		// Simulate medium complexity for larger blocks
		benchmarker.SetComputeSimulation(true, 5)
	}
	
	results := benchmarker.CompareBenchmarks()
	
	PrintResults(results)
	return results
}
