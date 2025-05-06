// main.go - Entry point for the blockchain simulation
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/yourusername/blockchain-poa/blockchain"
	"github.com/yourusername/blockchain-poa/config"
	"github.com/yourusername/blockchain-poa/transaction"
	"github.com/yourusername/blockchain-poa/visualization"
)

func main() {
	fmt.Println("Starting PoA Blockchain with Asynchronous Execution Simulation")
	
	// Create configuration
	cfg := config.DefaultConfig()
	
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
	for i := 0; i < numTransactions; i++ {
		// Random sender (among user accounts)
		senderIdx := i % 5
		senderAddr := fmt.Sprintf("user_%d", senderIdx)
		
		// Get the sender's current nonce
		senderAcc, _ := bc.State.GetAccount(senderAddr)
		
		// Random recipient (different from sender)
		recipientIdx := (senderIdx + 1 + i%4) % 5
		recipientAddr := fmt.Sprintf("user_%d", recipientIdx)
		
		// Create a private key for signing (in real world, user would have this)
		privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		
		// Random amount between 1 and 10
		amount := 1 + i%10
		
		// Create transaction
		tx := transaction.New(senderAddr, recipientAddr, amount, senderAcc.Nonce, privKey)
		
		// Submit to blockchain
		bc.SubmitTransaction(tx)
		
		// Wait a bit before next transaction
		time.Sleep(200 * time.Millisecond)
	}
}

// runSimulation runs the main blockchain simulation loop
func runSimulation(bc *blockchain.Blockchain) {
	// Initial state
	fmt.Println("\n=== Initial Blockchain State ===")
	bc.PrintState()
	
	// Number of rounds to simulate
	numRounds := 10
	
	// Main simulation loop
	for round := 0; round < numRounds; round++ {
		fmt.Printf("\n--- Starting round %d for block %d ---\n", 
			round+1, bc.CurrentHeight+1)
		
		// Current leader proposes a block
		newBlock := bc.ProposeBlock()
		if newBlock == nil {
			fmt.Println("No block proposed, waiting...")
			time.Sleep(cfg.BlockInterval)
			continue
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
}
