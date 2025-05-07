// main.go - Entry point for the blockchain simulation
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"runtime"
	// Removed unused "sync" import
	"time"
	"flag"

	"asynchronous-execution-simulation/blockchain"
	"asynchronous-execution-simulation/config"
	// Removed unused "execution" import
	"asynchronous-execution-simulation/performance"
	"asynchronous-execution-simulation/transaction"
	"asynchronous-execution-simulation/visualization"
)

// Command-line flags for configuration
var (
	numTransactions = flag.Int("txs", 500, "Number of transactions to generate")
	blockSize = flag.Int("block-size", 50, "Size of each block (transactions per block)")
	parallelWorkers = flag.Int("workers", runtime.NumCPU(), "Number of parallel worker threads")
	minBatchSize = flag.Int("min-batch", 20, "Minimum batch size for parallel execution")
	simulateCompute = flag.Bool("compute", false, "Simulate compute-intensive transactions")
	computeComplexity = flag.Int("complexity", 1, "Complexity level for simulated computation (1-20)")
	skipParallel = flag.Bool("sequential-only", false, "Skip parallel execution")
	testMode = flag.String("test-mode", "mixed", "Test mode: small, medium, complex, mixed")
)

func main() {
	// Parse command-line flags
	flag.Parse()
	
	fmt.Println("Starting PoA Blockchain with Asynchronous Execution Simulation")
	
	// Create configuration based on test mode
	cfg := createConfiguration()
	
	// Initialize blockchain
	bc, err := blockchain.New(cfg)
	if err != nil {
		fmt.Printf("Error initializing blockchain: %v\n", err)
		return
	}
	
	// Start submitting transactions
	go generateTransactions(bc, cfg.NumTransactions, *testMode)
	
	// Run the blockchain simulation
	runSimulation(bc)
}

// createConfiguration creates a configuration based on command-line flags and test mode
func createConfiguration() *config.Config {
	cfg := config.DefaultConfig()
	
	// Set values from command-line flags
	cfg.NumTransactions = *numTransactions
	cfg.BlockSize = *blockSize
	cfg.ParallelWorkers = *parallelWorkers
	cfg.ParallelExecutionEnabled = !*skipParallel
	cfg.MinParallelBatchSize = *minBatchSize
	cfg.SimulateCompute = *simulateCompute
	cfg.ComputeComplexity = *computeComplexity
	
	// Apply test mode configuration
	switch *testMode {
	case "small":
		fmt.Println("Using configuration for small, simple transactions")
		cfg.ConfigureForSmallTransactions()
	case "medium":
		fmt.Println("Using configuration for medium-complexity transactions")
		cfg.ConfigureForMediumTransactions()
	case "complex":
		fmt.Println("Using configuration for complex, compute-intensive transactions")
		cfg.ConfigureForComplexTransactions()
	case "mixed":
		fmt.Println("Using mixed transaction complexity configuration")
		// Default configuration is already mixed
	default:
		fmt.Printf("Unknown test mode '%s', using default configuration\n", *testMode)
	}
	
	fmt.Printf("Configuration: %d transactions, %d per block, %d workers, min batch %d, parallel: %v\n",
		cfg.NumTransactions, cfg.BlockSize, cfg.ParallelWorkers, cfg.MinParallelBatchSize, cfg.ParallelExecutionEnabled)
	
	return cfg
}

// generateTransactions creates and submits random transactions with complexity based on test mode
func generateTransactions(bc *blockchain.Blockchain, numTransactions int, testMode string) {
	fmt.Printf("Generating %d transactions in '%s' mode...\n", numTransactions, testMode)
	
	// Create more user accounts for testing
	for i := 5; i < 20; i++ {
		address := fmt.Sprintf("user_%d", i)
		bc.State.CreateAccount(address, 100+i)
	}
	
	// Determine the complexity distribution
	var simplePct, mediumPct float64
	// Removed unused "complexPct" variable
	switch testMode {
	case "small":
		simplePct, mediumPct = 0.9, 0.1
	case "medium":
		simplePct, mediumPct = 0.3, 0.6
	case "complex":
		simplePct, mediumPct = 0.1, 0.3
	case "mixed":
		simplePct, mediumPct = 0.6, 0.3
	default:
		simplePct, mediumPct = 0.6, 0.3
	}
	
	// Calculate how many transactions of each type to generate
	simpleCount := int(float64(numTransactions) * simplePct)
	mediumCount := int(float64(numTransactions) * mediumPct)
	complexCount := numTransactions - simpleCount - mediumCount
	
	// Generate different transaction patterns
	
	// 1. Create independent transactions (different senders, different recipients)
	// These should be highly parallelizable
	fmt.Println("Generating independent transactions...")
	generateIndependentTransactions(bc, simpleCount)
	
	// 2. Create sequential transactions from the same sender
	// These should run sequentially due to nonce dependencies
	fmt.Println("Generating sequential transactions (nonce dependencies)...")
	generateSequentialTransactions(bc, mediumCount)
	
	// 3. Create more complex transaction patterns (circular, many-to-one, etc.)
	fmt.Println("Generating complex transaction patterns...")
	generateComplexTransactions(bc, complexCount)
	
	fmt.Printf("Finished generating %d transactions\n", numTransactions)
}

// generateIndependentTransactions creates transactions with few dependencies
func generateIndependentTransactions(bc *blockchain.Blockchain, count int) {
	for i := 0; i < count; i++ {
		senderIdx := i % 15
		senderAddr := fmt.Sprintf("user_%d", senderIdx)
		
		recipientIdx := (senderIdx + 5) % 20
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(senderAddr, recipientAddr, 1+i%5, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
	}
}

// generateSequentialTransactions creates transactions with strong nonce dependencies
func generateSequentialTransactions(bc *blockchain.Blockchain, count int) {
	// Use a small set of senders to create nonce dependencies
	senders := []string{"user_0", "user_1", "user_2"}
	
	for i := 0; i < count; i++ {
		senderAddr := senders[i%len(senders)]
		recipientIdx := 10 + (i % 10)
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(senderAddr, recipientAddr, 1, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
		
		// Small delay to ensure proper nonce ordering
		time.Sleep(5 * time.Millisecond)
	}
}

// generateComplexTransactions creates transactions with complex dependencies
func generateComplexTransactions(bc *blockchain.Blockchain, count int) {
	// Split into different patterns
	patternCount := count / 3
	
	// 1. Create circular transfers (A->B->C->A)
	for i := 0; i < patternCount; i++ {
		chainLength := 3 + (i % 3) // Chains of length 3-5
		baseIdx := i % 10
		
		// Create a circle of transactions
		for j := 0; j < chainLength; j++ {
			senderIdx := (baseIdx + j) % 20
			recipientIdx := (baseIdx + j + 1) % 20
			if j == chainLength-1 {
				recipientIdx = baseIdx // Close the circle
			}
			
			senderAddr := fmt.Sprintf("user_%d", senderIdx)
			recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
			
			senderAcc, _ := bc.State.GetAccount(senderAddr)
			privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			
			tx := transaction.New(senderAddr, recipientAddr, 1, senderAcc.Nonce, privKey)
			bc.SubmitTransaction(tx)
		}
	}
	
	// 2. Create many-to-one transactions (many senders, one recipient)
	for i := 0; i < patternCount; i++ {
		senderIdx := i % 19 + 1 // Skip user_0 (used in pattern 2)
		senderAddr := fmt.Sprintf("user_%d", senderIdx)
		recipientAddr := "user_0" // Same recipient
		
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(senderAddr, recipientAddr, 1, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
	}
	
	// 3. Create one-to-many transactions (one sender, many recipients)
	remainingCount := count - 2*patternCount
	centralSender := "user_1"
	for i := 0; i < remainingCount; i++ {
		recipientIdx := (i % 18) + 2 // Avoid sending to self or user_0
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		senderAcc, _ := bc.State.GetAccount(centralSender)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(centralSender, recipientAddr, 1, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
		
		// Small delay to ensure proper nonce ordering
		time.Sleep(5 * time.Millisecond)
	}
}

// runSimulation runs the main blockchain simulation loop
func runSimulation(bc *blockchain.Blockchain) {
	// Initial state
	fmt.Println("\n=== Initial Blockchain State ===")
	bc.PrintState()
	
	// Run performance comparison for sequential vs parallel execution
	fmt.Println("\n=== Running Performance Comparison (Sequential vs Parallel) ===")
	
	// Toggle parallel execution off for the first half of blocks to compare
	originalParallelSetting := bc.Config.ParallelExecutionEnabled
	bc.Config.ParallelExecutionEnabled = false
	
	// Number of rounds to simulate
	numRounds := 10
	blockBenchmarks := make([]performance.BenchmarkResults, 0)
	
	// Main simulation loop
	for round := 0; round < numRounds; round++ {
		fmt.Printf("\n--- Starting round %d for block %d ---\n", 
			round+1, bc.CurrentHeight+1)
		
		// Toggle parallel execution on for second half of blocks
		if round == numRounds/2 {
			fmt.Println("\n=== Switching to Parallel Execution ===")
			bc.Config.ParallelExecutionEnabled = originalParallelSetting
		}
		
		// Current leader proposes a block
		newBlock := bc.ProposeBlock()
		if newBlock == nil {
			fmt.Println("No block proposed, waiting...")
			time.Sleep(bc.Config.BlockInterval)
			continue
		}
		
		// Benchmark this block's execution if it has enough transactions
		if len(newBlock.Transactions) > 10 {
			// Run benchmark
			benchResult := performance.RunBlockBenchmark(
				newBlock, 
				bc.State.Copy(), 
				bc.Config.ParallelWorkers,
			)
			blockBenchmarks = append(blockBenchmarks, benchResult)
		}
		
		// Validators vote on the block
		quorumReached := bc.VoteOnBlock(newBlock)
		if quorumReached {
			// Add block to blockchain
			bc.AddBlock(newBlock)
			
			// Process pending executions
			bc.ProcessPendingExecutions()
		}
		
		// Rotate leader for next round
		bc.RotateLeader()
		
		// Print state every few rounds
		if (round+1) % 3 == 0 {
			fmt.Printf("\n=== Blockchain State after round %d ===\n", round+1)
			bc.PrintState()
		}
		
		// Wait for next block interval
		time.Sleep(bc.Config.BlockInterval)
	}
	
	// Restore original parallel setting
	bc.Config.ParallelExecutionEnabled = originalParallelSetting
	
	// Wait for final execution and verification
	time.Sleep(5 * time.Second)
	
	// Print final state
	fmt.Println("\n=== Final Blockchain State ===")
	bc.PrintState()
	
	// Create visualizer and print statistics
	visualizer := visualization.New(bc)
	visualizer.PrintBlockchainTimeline()
	visualizer.PrintTransactionFlow()
	visualizer.PrintValidatorStats()
	visualizer.PrintExecutionStats()
	
	// Print overall performance comparison
	fmt.Println("\n=== Overall Performance Comparison ===")
	fmt.Println("Block | Sequential Time | Parallel Time | Speedup | Conflicts")
	fmt.Println("------+----------------+--------------+--------+----------")
	
	totalSpeedup := 0.0
	totalBlocks := len(blockBenchmarks)
	
	for i, result := range blockBenchmarks {
		fmt.Printf(" %3d  | %14v | %12v | %6.2fx | %8d\n",
			i+1,
			result.SequentialStats.TotalTime,
			result.ParallelStats.TotalTime,
			result.Speedup,
			result.ParallelStats.Conflicts,
		)
		totalSpeedup += result.Speedup
	}
	
	if totalBlocks > 0 {
		avgSpeedup := totalSpeedup / float64(totalBlocks)
		fmt.Printf("\nAverage Speedup: %.2fx\n", avgSpeedup)
		
		if avgSpeedup > 1.5 {
			fmt.Println("Overall, parallel execution showed significant performance benefits.")
		} else if avgSpeedup > 1.0 {
			fmt.Println("Overall, parallel execution showed modest performance benefits.")
		} else {
			fmt.Println("Overall, parallel execution did not provide significant performance benefits " +
				"for this workload. This may be due to high transaction dependencies or conflicts.")
		}
	}
}
