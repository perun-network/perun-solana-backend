package funder

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/perun-network/perun-solana-backend/client"

	pchannel "perun.network/go-perun/channel"
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
) *Funder {
	return &Funder{
		cb:              cb,
		perunAddr:       perunAddr,
		assetAddrs:      assetAddrs,
		maxIters:        MaxIterationsUntilAbort,
		pollingInterval: DefaultPollingInterval,
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

// Fund first calls open if the channel is not opened and then funds the channel.
func (f *Funder) Fund(ctx context.Context, req pchannel.FundingReq) error {
	log.Println("Fund called")

	if req.Idx != 0 && req.Idx != 1 {
		return errors.New("req.Idx must be 0 or 1")
	}

	if req.Idx == pchannel.Index(0) {
		err := f.openChannel(ctx, req)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *Funder) openChannel(ctx context.Context, req pchannel.FundingReq) error {
	err := f.cb.Open(ctx, f.perunAddr, req.Params, req.State)
	if err != nil {
		return errors.Join(errors.New("error while opening channel in party A"), err)
	}

	time.Sleep(f.pollingInterval) // Wait for the channel to be opened
	channelInfo, err := f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
	if err != nil {
		log.Println("Error while getting channel info: ", err)
		return err
	}
	log.Println("channel opened: ", channelInfo)
	return nil
}
