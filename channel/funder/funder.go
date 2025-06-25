package funder

import (
	"time"

	"github.com/gagliardetto/solana-go"
	"perun.network/perun-stellar-backend/client"
)

const (
	MaxIterationsUntilAbort = 30
	DefaultPollingInterval  = time.Duration(4) * time.Second
)

// Funder is a struct that implements the Funder interface for Stellar.
type Funder struct {
	cb              *client.ContractBackend
	perunAddr       solana.PublicKey
	assetAddrs      []solana.PublicKey
	maxIters        int
	pollingInterval time.Duration
}

// NewFunder creates a new Funder instance with the given parameters.
func NewFunder(
	cb *client.ContractBackend,
	perunAddr solana.PublicKey,
	assetAddrs []solana.PublicKey,
	maxIters int,
	pollingInterval time.Duration,
) *Funder {
	if maxIters <= 0 {
		maxIters = MaxIterationsUntilAbort
	}
	if pollingInterval <= 0 {
		pollingInterval = DefaultPollingInterval
	}
	return &Funder{
		cb:              cb,
		perunAddr:       perunAddr,
		assetAddrs:      assetAddrs,
		maxIters:        maxIters,
		pollingInterval: pollingInterval,
	}
}

// GetPerunAddr returns the perun address of the funder.
func (f *Funder) GetPerunAddr() solana.PublicKey {
	return f.perunAddr
}

// GetAssetAddrs returns the asset addresses of the funder.
func (f *Funder) GetAssetAddrs() []solana.PublicKey {
	return f.assetAddrs
}
