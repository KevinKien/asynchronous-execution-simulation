flowchart TD
    subgraph "Transaction Processing"
        A1[User Creates Transaction] --> A2[User Signs Transaction]
        A2 --> A3[Transaction Sent to Network]
        A3 --> A4[Transaction Enters Mempool]
    end
    
    subgraph "Block Creation & Consensus"
        B1[Leader Selects Transactions] --> B2[Leader Creates Block Proposal]
        B2 --> B3[Block Contains:\n- Header\n- Transactions\n- Delayed Merkle Root]
        B3 --> B4[Leader Signs Block]
        B4 --> B5[Block Proposed]
        B5 --> B6{Validators Vote}
        B6 -->|Quorum Reached| B7[Block Voted]
        B6 -->|Insufficient Votes| B1
        B7 --> B8[Previous Block Finalized]
    end
    
    subgraph "Asynchronous Execution"
        C1[Finalized Block Enter Execution Queue] --> C2{Speculative Execution?}
        C2 -->|Yes| C3[Execute Block Before Finalization]
        C2 -->|No| C4[Execute Block After Finalization]
        C3 --> C5[Store Speculative Results]
        C4 --> C6[Update State]
        C5 -->|Block Finalized| C6
        C6 --> C7[Calculate New State Root]
        C7 --> C8[Store State Root]
    end
    
    subgraph "Verification"
        D1[Block Height + D Reached] --> D2[Verify Block]
        D2 --> D3[Compare State Root with Delayed Root]
        D3 -->|Match| D4[Block Verified]
        D3 -->|Mismatch| D5[Rollback and Re-execute]
        D5 --> C4
    end
    
    A4 --> B1
    B8 --> C1
    C8 --> D1
    
    classDef consensus fill:#f9d5e5,stroke:#333,stroke-width:1px
    classDef transaction fill:#eeeeee,stroke:#333,stroke-width:1px
    classDef execution fill:#e3f0f9,stroke:#333,stroke-width:1px
    classDef verification fill:#d5f5e3,stroke:#333,stroke-width:1px
    
    class A1,A2,A3,A4 transaction
    class B1,B2,B3,B4,B5,B6,B7,B8 consensus
    class C1,C2,C3,C4,C5,C6,C7,C8 execution
    class D1,D2,D3,D4,D5 verification
