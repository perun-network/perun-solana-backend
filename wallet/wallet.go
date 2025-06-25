package wallet

import (
	"math/rand"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"
)

// EphemeralWallet is a wallet that stores accounts in memory.
type EphemeralWallet struct {
	lock     sync.Mutex
	accounts map[string]*Account
}

// Unlock unlocks the account associated with the given address.
func (e *EphemeralWallet) Unlock(a wallet.Address) (wallet.Account, error) {
	addr, ok := a.(*Participant)
	if !ok {
		return nil, errors.New("incorrect Participant type")
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	account, ok := e.accounts[addr.String()]
	if !ok {
		return nil, errors.New("account not found")
	}
	return account, nil
}

// LockAll locks all accounts.
func (e *EphemeralWallet) LockAll() {}

// IncrementUsage increments the usage counter of the account associated with the given address.
func (e *EphemeralWallet) IncrementUsage(address wallet.Address) {}

// DecrementUsage decrements the usage counter of the account associated with the given address.
func (e *EphemeralWallet) DecrementUsage(address wallet.Address) {}

// AddNewAccount generates a new account and adds it to the wallet.
func (e *EphemeralWallet) AddNewAccount(rng *rand.Rand) (*Account, *solana.PrivateKey, error) {
	acc, kp, err := NewRandomAccount(rng)
	if err != nil {
		return nil, nil, err
	}
	return acc, kp, e.AddAccount(acc)
}

// AddAccount adds the given account to the wallet.
func (e *EphemeralWallet) AddAccount(acc *Account) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	k := AsParticipant(acc.Address()).String()
	if _, ok := e.accounts[k]; ok {
		return errors.New("account already exists")
	}
	e.accounts[k] = acc
	return nil
}

// NewEphemeralWallet creates a new EphemeralWallet instance.
func NewEphemeralWallet() *EphemeralWallet {
	return &EphemeralWallet{
		accounts: make(map[string]*Account),
	}
}
