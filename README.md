# Blockchain Proof of Authority with Asynchronous Execution

This Go project simulates a blockchain with Proof of Authority (PoA) consensus mechanism and asynchronous execution, following the process described in the design document. The simulation captures the entire workflow from transaction submission to state verification.

## Project Structure

```
├── block/
│   └── block.go           # Block data structure and operations
├── blockchain/
│   └── blockchain.go      # Main blockchain engine
├── config/
│   └── config.go          # Configuration parameters
├── state/
│   └── state.go           # State management
├── transaction/
│   └── transaction.go     # Transaction operations
├── validator/
│   └── validator.go       # Validator operations
├── visualization/
│   └── visualization.go   # Visualization utilities
└── main.go                # Entry point for simulation
```

## Key Features

1. **PoA Consensus**:
   - Validators take turns becoming leaders in a round-robin fashion
   - Leaders propose blocks containing transactions
   - Validators vote on proposed blocks without executing transactions
   - Blocks are finalized when they receive a quorum of votes

2. **Asynchronous Execution**:
   - Consensus and execution are decoupled processes
   - Blocks are executed after finalization (or speculatively)
   - State roots are calculated after execution
   - State roots from D blocks ago are included in new blocks for verification

3. **Transaction Flow**:
   - Transactions are created, signed, and submitted to the mempool
   - Transactions are propagated to validators
   - Leaders select and order transactions into blocks
   - Execution follows the order determined by consensus

4. **Speculative Execution**:
   - Optional feature to execute blocks before finalization
   - Results are stored and used if the block gets finalized
   - Improves throughput by parallelizing consensus and execution

## How to Run

1. Make sure you have Go installed (version 1.16 or higher recommended)

2. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/blockchain-poa.git
   cd blockchain-poa
   ```

3. Build and run the simulation:
   ```bash
   go build
   ./blockchain-poa
   ```

## Configuration Parameters

You can adjust the following parameters in the `config/config.go` file:

- `NumValidators`: Number of validators in the network
- `NumTransactions`: Number of transactions to generate
- `BlockSize`: Maximum number of transactions per block
- `ExecutionDelay`: Number of blocks (D) between execution and verification
- `BlockInterval`: Time between block proposals
- `QuorumPercentage`: Required percentage of validators for consensus
- `SpeculativeEnabled`: Whether to enable speculative execution

## Simulation Output

The simulation outputs several visualizations:

1. **Blockchain State**: Current height, account balances, and block status
2. **Blockchain Timeline**: Timestamps for each block's state transitions
3. **Transaction Flow**: Distribution of transactions across blocks
4. **Validator Statistics**: Block proposals and voting activity
5. **Execution Statistics**: Delays and lags between consensus and execution

## Implementation Details

### Block Lifecycle

Each block goes through the following states:

1. **Proposed**: Block is created and proposed by the current leader
2. **Voted**: Block received enough votes from validators (quorum)
3. **Finalized**: Block cannot be reverted (happens when next block is voted)
4. **Verified**: Execution is complete and state is updated

### Asynchronous Execution Process

1. When a block is finalized, it enters the execution queue
2. Transactions in the block are executed against the state
3. A new state root is calculated and stored
4. The block is verified once execution is complete
5. Future blocks include the delayed state root for verification

## Notes

This is a simplified simulation designed for educational purposes. In a production environment, you would need to implement:

- Network communication between nodes
- Persistence for blockchain and state
- More sophisticated mempool management
- Advanced error handling and recovery mechanisms
- Cryptographic operations with proper security measures

## License

MIT License
