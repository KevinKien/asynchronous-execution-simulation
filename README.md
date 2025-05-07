# Blockchain PoA with Asynchronous & Parallel Execution

This project is a blockchain simulation using Proof of Authority (PoA) consensus mechanism combined with asynchronous and parallel execution. This simulation fully demonstrates the process from when a user submits a transaction until the state is updated in the database, exactly as described in the design document.

## Project Structure

```
├── block/
│   └── block.go           # Block data structure and operations
├── blockchain/
│   └── blockchain.go      # Main blockchain engine
├── config/
│   └── config.go          # Configuration parameters
├── execution/
│   └── parallel_execution.go # Quản lý thực thi song song
├── performance/
│   └── benchmarks.go      # Benchmark và phân tích hiệu suất
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

## Operation Process

1. **User Submits Transaction**
- Transaction is signed with the user's private key
- Transaction is sent to the network and enters the mempool

2. **Transaction Propagation**
- Transaction is propagated to validators
- Validators perform preliminary checks (signature, format, balance)

3. **Block Creation and Consensus Process**
- Leader selects transactions from the mempool and creates a new block proposal
- Block contains reference to previous block, transaction list, delayed merkle root
- Other validators verify the block and vote without executing
- Block is finalized when it receives sufficient votes

4. **Parallel Execution**
- Transactions are analyzed to identify dependencies
- Transactions are executed in parallel across multiple threads
- System tracks inputs/outputs and detects conflicts
- Conflicting transactions are re-executed sequentially

5. **State Update**
- Execution results are merged into the common state
- New merkle root is calculated and stored
- Block becomes "Verified" when execution is complete

## Key Features

1. **Proof of Authority (PoA) Consensus Mechanism**
- Authorized validators take turns becoming the leader (proposer) according to a schedule
- The leader selects and arranges transactions to include in the new block
- Validators verify and vote for the block without executing transactions
- A block is finalized when it receives sufficient votes (typically 2/3 of validators)

2. **Asynchronous Execution**
- Consensus and execution processes are separated
- Blocks are executed after being finalized (or speculatively executed)
- Merkle root from D previous blocks is included in the new block for verification
- Allows consensus to continue while blocks are being executed

3. **Parallel Execution**
- Executes multiple transactions simultaneously on different processing threads
- Detects and resolves conflicts between transactions
- Analyzes dependencies to optimize transaction ordering
- Compares performance between sequential and parallel execution

4. **Speculative Execution**
- Executes blocks before they are finalized
- Stores temporary results and applies them if the block is finalized
- Increases performance by parallelizing consensus and execution

## How to Run

1. Make sure you have Go installed (version 1.16 or higher recommended)

2. Clone the repository:
   ```bash
   git clone https://github.com/KevinKien/asynchronous-execution-simulation.git
   cd asynchronous-execution-simulation
   ```

3. Build and run the simulation:
   ```bash
   go build
   ./asynchronous-execution-simulation
   ```

## Configuration Customization
Open the config/config.go file to adjust parameters:
- `NumValidators`: Number of validators in the network
- `NumTransactions`: Number of transactions to create
- `BlockSize`: Maximum transactions in a block
- `ExecutionDelay`: Block delay D between execution and verification
- `ParallelExecutionEnabled`: Enable/disable parallel execution
- `ParallelWorkers`: Number of parallel threads
- `OptimizeTransactionOrder`: Enable/disable transaction order optimization

## Performance Measurement
The project includes benchmarking tools to compare performance between sequential and parallel execution:
- `Execution Time`: Compares total time and average time per transaction
- `Conflict Detection`: Number and percentage of conflicts between transactions
- `Speedup Ratio`: Performance improvement level of parallel execution

Results are displayed after each block and in a final summary report.

## Sample Transactions for Testing
The simulation generates various transaction patterns to test parallel execution performance:
- `Independent Transactions`: Transactions between unrelated accounts
- `Sequential Transactions`: Multiple transactions from the same sender
- `Circular Transactions`: Chain of transactions forming a circle (A->B->C->A)
- `Many-to-One Transactions`: Multiple senders, one receiver
- `One-to-Many Transactions`: One sender, multiple receivers

## Results and Analysis
After running the simulation, you will receive:
- `Blockchain State`: Current state, account balances, and block status
- `Blockchain Timeline`: Timeline for each block
- `Transaction Flow`: Distribution of transactions across blocks
- `Validator Statistics`: Statistics on validator activities
- `Execution Statistics`: Execution time and delay statistics
- `Performance Comparison`: Sequential vs parallel performance comparison

## Block Lifecycle
Each block goes through the following states:
- `Proposed`: Block is created and proposed by the leader
- `Voted`: Block receives sufficient votes from validators
- `Finalized`: Block cannot be reversed (occurs when the next block is voted)
- `Verified`: Execution is complete and state is updated

## Synchronization and Error Handling

- If a node detects its merkle root doesn't match the consensus merkle root after D blocks, it will rollback to the last verified state and re-execute
- In case of conflicting transactions during parallel execution, the system will re-execute affected transactions sequentially

## Compatibility with Design Document
This simulation is built to fully demonstrate the process described in the design document, including:
- PoA consensus mechanism
- Asynchronous execution with D blocks delay
- Delayed merkle root for state verification
- Parallel execution with conflict detection

## Notes
This is a simplified simulation to illustrate concepts. In a real production environment, you would need:

- Network communication between nodes
- Blockchain and state persistence
- Improved mempool management
- Advanced error handling and recovery mechanisms
- Enhanced cryptographic security measures

