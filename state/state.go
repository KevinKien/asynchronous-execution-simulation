// state/state.go
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	"github.com/KevinKien/asynchronous-execution-simulation/transaction"
)

// Account represents a user account in the state
type Account struct {
	Address string
	Balance int
	Nonce   int
}

// State manages the global state of the blockchain
type State struct {
	Accounts    map[string]*Account
	StateRoot   string
	StateMutex  sync.RWMutex
}

// New creates a new state
func New() *State {
	state := &State{
		Accounts:  make(map[string]*Account),
		StateRoot: "",
	}
	
	// Calculate initial state root
	state.CalculateStateRoot()
	
	return state
}

// Copy creates a deep copy of the state
func (s *State) Copy() *State {
	s.StateMutex.RLock()
	defer s.StateMutex.RUnlock()
	
	newState := &State{
		Accounts:  make(map[string]*Account),
		StateRoot: s.StateRoot,
	}
	
	for addr, acc := range s.Accounts {
		newState.Accounts[addr] = &Account{
			Address: acc.Address,
			Balance: acc.Balance,
			Nonce:   acc.Nonce,
		}
	}
	
	return newState
}

// CreateAccount creates a new account in the state
func (s *State) CreateAccount(address string, balance int) {
	s.StateMutex.Lock()
	defer s.StateMutex.Unlock()
	
	s.Accounts[address] = &Account{
		Address: address,
		Balance: balance,
		Nonce:   0,
	}
	
	s.CalculateStateRoot()
}

// GetAccount gets an account by address
func (s *State) GetAccount(address string) (*Account, bool) {
	s.StateMutex.RLock()
	defer s.StateMutex.RUnlock()
	
	acc, exists := s.Accounts[address]
	return acc, exists
}

// ExecuteTransaction executes a single transaction against the state
func (s *State) ExecuteTransaction(tx *transaction.Transaction) error {
	s.StateMutex.Lock()
	defer s.StateMutex.Unlock()
	
	// Get sender account
	sender, exists := s.Accounts[tx.Sender]
	if !exists {
		return fmt.Errorf("sender account %s not found", tx.Sender)
	}
	
	// Verify nonce
	if tx.Nonce != sender.Nonce {
		return fmt.Errorf("invalid nonce for tx %s: expected %d, got %d", 
			tx.ID, sender.Nonce, tx.Nonce)
	}
	
	// Verify balance
	if sender.Balance < tx.Amount {
		return fmt.Errorf("insufficient balance for tx %s", tx.ID)
	}
	
	// Get or create recipient account
	recipient, exists := s.Accounts[tx.Recipient]
	if !exists {
		recipient = &Account{
			Address: tx.Recipient,
			Balance: 0,
			Nonce:   0,
		}
		s.Accounts[tx.Recipient] = recipient
	}
	
	// Execute the transaction
	sender.Balance -= tx.Amount
	recipient.Balance += tx.Amount
	sender.Nonce++
	
	// Update state root
	s.CalculateStateRoot()
	
	return nil
}

// CalculateStateRoot calculates the Merkle root of the current state
func (s *State) CalculateStateRoot() {
	// Sort accounts by address for deterministic root
	addresses := make([]string, 0, len(s.Accounts))
	for addr := range s.Accounts {
		addresses = append(addresses, addr)
	}
	sort.Strings(addresses)
	
	// Create a string representation of the state
	stateStr := ""
	for _, addr := range addresses {
		acc := s.Accounts[addr]
		stateStr += fmt.Sprintf("%s:%d:%d_", addr, acc.Balance, acc.Nonce)
	}
	
	// Calculate hash
	hash := sha256.Sum256([]byte(stateStr))
	s.StateRoot = hex.EncodeToString(hash[:])
}

// String returns a string representation of the state
func (s *State) String() string {
	s.StateMutex.RLock()
	defer s.StateMutex.RUnlock()
	
	result := fmt.Sprintf("State [Root: %s]\n", s.StateRoot)
	for _, acc := range s.Accounts {
		result += fmt.Sprintf("  %s: Balance=%d, Nonce=%d\n", 
			acc.Address, acc.Balance, acc.Nonce)
	}
	
	return result
}
