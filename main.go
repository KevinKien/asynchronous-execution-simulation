// main.go - Entry point for the blockchain simulation
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/KevinKien/asynchronous-execution-simulation/blockchain"
	"github.com/KevinKien/asynchronous-execution-simulation/config"
	"github.com/KevinKien/asynchronous-execution-simulation/transaction"
	"github.com/KevinKien/asynchronous-execution-simulation/visualization"
)

func main() {
	fmt.Println("Starting PoA Blockchain with Asynchronous Execution Simulation")
	
	// Create configuration with many transactions to show parallel execution benefits
	cfg := config.DefaultConfig()
	cfg.NumTransactions = 500     // Increase number of transactions
	cfg.BlockSize = 50           // Increase block size
	cfg.ParallelWorkers = runtime.NumCPU() // Use all available CPU cores
	
	// Initialize blockchain
	bc, err := blockchain.New(cfg)
	if err != nil {
		fmt.Printf("Error initializing blockchain: %v\n", err)
		return
	}
	
	// Start submitting transactions
	go generateTransactions(bc, cfg.NumTransactions)
	
	// Run the blockchain simulation
	runSimulation(bc)
}

// generateTransactions creates and submits random transactions
func generateTransactions(bc *blockchain.Blockchain, numTransactions int) {
	fmt.Printf("Generating %d transactions...\n", numTransactions)
	
	// Create more user accounts for testing
	for i := 5; i < 20; i++ {
		address := fmt.Sprintf("user_%d", i)
		bc.State.CreateAccount(address, 100+i)
	}
	
	// Create different transaction patterns to test parallel execution
	
	// 1. Create a batch of independent transactions (different senders, different recipients)
	// These should be highly parallelizable
	fmt.Println("Generating independent transactions...")
	for i := 0; i < numTransactions/5; i++ {
		senderIdx := i % 15
		senderAddr := fmt.Sprintf("user_%d", senderIdx)
		
		recipientIdx := (senderIdx + 5) % 20
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(senderAddr, recipientAddr, 1+i%5, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
	}
	
	// 2. Create sequential transactions from the same sender
	// These should run sequentially due to nonce dependencies
	fmt.Println("Generating sequential transactions from the same sender...")
	for i := 0; i < numTransactions/5; i++ {
		senderAddr := "user_0" // Same sender for all
		recipientIdx := 1 + (i % 15)
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(senderAddr, recipientAddr, 1, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
		
		// Small delay to ensure proper nonce ordering
		time.Sleep(10 * time.Millisecond)
	}
	
	// 3. Create circular transfers (A->B->C->A)
	// These have complex dependencies
	fmt.Println("Generating circular transactions...")
	for i := 0; i < numTransactions/5; i++ {
		numAccounts := 5
		baseIdx := i % 15
		
		// Create a circle of transactions
		for j := 0; j < numAccounts; j++ {
			senderIdx := (baseIdx + j) % 20
			recipientIdx := (baseIdx + j + 1) % 20
			
			senderAddr := fmt.Sprintf("user_%d", senderIdx)
			recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
			
			senderAcc, _ := bc.State.GetAccount(senderAddr)
			privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			
			tx := transaction.New(senderAddr, recipientAddr, 1, senderAcc.Nonce, privKey)
			bc.SubmitTransaction(tx)
		}
	}
	
	// 4. Create many-to-one transactions (many senders, one recipient)
	fmt.Println("Generating many-to-one transactions...")
	for i := 0; i < numTransactions/5; i++ {
		senderIdx := i % 19 + 1 // Skip user_0 (used in pattern 2)
		senderAddr := fmt.Sprintf("user_%d", senderIdx)
		recipientAddr := "user_0" // Same recipient
		
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(senderAddr, recipientAddr, 1, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
	}
	
	// 5. Create one-to-many transactions (one sender, many recipients)
	fmt.Println("Generating one-to-many transactions...")
	centralSender := "user_1"
	for i := 0; i < numTransactions/5; i++ {
		recipientIdx := (i % 18) + 2 // Avoid sending to self or user_0
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		senderAcc, _ := bc.State.GetAccount(centralSender)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		tx := transaction.New(centralSender, recipientAddr, 1, senderAcc.Nonce, privKey)
		bc.SubmitTransaction(tx)
		
		// Small delay to ensure proper nonce ordering
		time.Sleep(10 * time.Millisecond)
	}
	
	fmt.Printf("Finished generating %d transactions\n", numTransactions)
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
			bc.Config.ParallelExecutionEnabled = true
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
