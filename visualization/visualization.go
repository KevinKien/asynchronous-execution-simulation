// visualization/visualization.go
package visualization

import (
	"fmt"
	"strings"
	"time"

	"asynchronous-execution-simulation/block"
	"asynchronous-execution-simulation/blockchain"
)

// BlockchainVisualizer provides visualization utilities for the blockchain
type BlockchainVisualizer struct {
	bc *blockchain.Blockchain
}

// New creates a new blockchain visualizer
func New(bc *blockchain.Blockchain) *BlockchainVisualizer {
	return &BlockchainVisualizer{
		bc: bc,
	}
}

// PrintBlockchainTimeline prints a visual timeline of blocks and their state transitions
func (v *BlockchainVisualizer) PrintBlockchainTimeline() {
	fmt.Println("\n=== Blockchain Timeline ===")
	fmt.Println("Block | Proposed | Voted | Finalized | Verified | Transactions | State Root")
	fmt.Println("----------------------------------------------------------------------")
	
	for _, b := range v.bc.Blocks {
		proposedTime := formatTime(b.ProposedAt)
		votedTime := formatTime(b.VotedAt)
		finalizedTime := formatTime(b.FinalizedAt)
		verifiedTime := formatTime(b.VerifiedAt)
		
		fmt.Printf(" %3d  | %8s | %4s | %8s | %7s | %11d | %s\n",
			b.Height,
			proposedTime,
			votedTime,
			finalizedTime,
			verifiedTime,
			len(b.Transactions),
			shortHash(b.StateRoot),
		)
	}
}

// PrintTransactionFlow prints transaction flow from mempool to execution
func (v *BlockchainVisualizer) PrintTransactionFlow() {
	fmt.Println("\n=== Transaction Flow ===")
	fmt.Println("Mempool Size: ", len(v.bc.Mempool))
	
	// Count transactions per block
	txCountMap := make(map[int]int)
	for _, b := range v.bc.Blocks {
		txCountMap[b.Height] = len(b.Transactions)
	}
	
	// Print a bar chart of transactions per block
	fmt.Println("\nTransactions per Block:")
	for i := 0; i <= v.bc.CurrentHeight; i++ {
		count, exists := txCountMap[i]
		if !exists {
			count = 0
		}
		
		fmt.Printf("Block %2d: %s (%d)\n", i, strings.Repeat("■", count), count)
	}
}

// PrintValidatorStats prints statistics about validators
func (v *BlockchainVisualizer) PrintValidatorStats() {
	fmt.Println("\n=== Validator Statistics ===")
	
	// Count blocks proposed by each validator
	proposalCountMap := make(map[int]int)
	for _, b := range v.bc.Blocks {
		proposalCountMap[b.Proposer]++
	}
	
	// Print statistics
	fmt.Println("Validator | Is Leader | Proposed Blocks | Mempool Size")
	fmt.Println("--------------------------------------------------")
	for _, validator := range v.bc.Validators {
		proposedBlocks := proposalCountMap[validator.ID]
		leader := " "
		if validator.IsLeader {
			leader = "✓"
		}
		
		fmt.Printf(" %8d | %8s | %14d | %11d\n",
			validator.ID,
			leader,
			proposedBlocks,
			len(validator.Mempool),
		)
	}
}

// PrintExecutionStats prints statistics about execution vs consensus
func (v *BlockchainVisualizer) PrintExecutionStats() {
	fmt.Println("\n=== Execution vs Consensus ===")
	
	totalBlocks := v.bc.CurrentHeight + 1
	finalizedBlocks := 0
	verifiedBlocks := 0
	executionDelayTotal := 0
	verificationDelayTotal := 0
	
	for _, b := range v.bc.Blocks {
		if b.Status == block.Finalized || b.Status == block.Verified {
			finalizedBlocks++
			
			// Calculate finalization delay
			if !b.ProposedAt.IsZero() && !b.FinalizedAt.IsZero() {
				delay := b.FinalizedAt.Sub(b.ProposedAt)
				executionDelayTotal += int(delay.Milliseconds())
			}
		}
		
		if b.Status == block.Verified {
			verifiedBlocks++
			
			// Calculate verification delay
			if !b.ProposedAt.IsZero() && !b.VerifiedAt.IsZero() {
				delay := b.VerifiedAt.Sub(b.ProposedAt)
				verificationDelayTotal += int(delay.Milliseconds())
			}
		}
	}
	
	// Calculate averages
	var avgExecutionDelay float64
	if finalizedBlocks > 0 {
		avgExecutionDelay = float64(executionDelayTotal) / float64(finalizedBlocks)
	}
	
	var avgVerificationDelay float64
	if verifiedBlocks > 0 {
		avgVerificationDelay = float64(verificationDelayTotal) / float64(verifiedBlocks)
	}
	
	fmt.Printf("Total Blocks: %d\n", totalBlocks)
	fmt.Printf("Finalized Blocks: %d (%.1f%%)\n", 
		finalizedBlocks, float64(finalizedBlocks)/float64(totalBlocks)*100)
	fmt.Printf("Verified Blocks: %d (%.1f%%)\n", 
		verifiedBlocks, float64(verifiedBlocks)/float64(totalBlocks)*100)
	fmt.Printf("Average Execution Delay: %.2f ms\n", avgExecutionDelay)
	fmt.Printf("Average Verification Delay: %.2f ms\n", avgVerificationDelay)
	fmt.Printf("Execution Lag: %d blocks\n", totalBlocks-v.bc.LastExecutedHeight-1)
	fmt.Printf("Verification Lag: %d blocks\n", totalBlocks-v.bc.LastVerifiedHeight-1)
}

// Helper functions
func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05")
}

func shortHash(hash string) string {
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8] + "..."
}
