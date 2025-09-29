package adjudicator

import (
	"context"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/perun-network/perun-solana-backend/client"
	pchannel "perun.network/go-perun/channel"
)

var ErrChannelAlreadyClosed = errors.New("channel is already closed")

var DefaultChallengeDuration = time.Duration(20) * time.Second //nolint:gomnd

const (
	MaxIterationsUntilAbort = 30
	DefaultPollingInterval  = time.Duration(4) * time.Second
)

type Adjudicator struct {
	cb                *client.ContractBackend
	perunAddr         solana.PublicKey
	assetAddrs        []solana.PublicKey
	challengeDuration *time.Duration
	maxIters          int
	pollingInterval   time.Duration
	oneWithdrawer     bool
}

func NewAdjudicator(cb *client.ContractBackend, perunAddr solana.PublicKey, assetAddrs []solana.PublicKey, oneWithdrawer bool) *Adjudicator {
	return &Adjudicator{
		cb:                cb,
		perunAddr:         perunAddr,
		assetAddrs:        assetAddrs,
		challengeDuration: &DefaultChallengeDuration,
		maxIters:          MaxIterationsUntilAbort,
		pollingInterval:   DefaultPollingInterval,
		oneWithdrawer:     oneWithdrawer,
	}
}

func (a Adjudicator) Register(ctx context.Context, req pchannel.AdjudicatorReq, subChannels []pchannel.SignedState) error {
	panic("TODO: implement Register in adjudicator")
}

func (a Adjudicator) Withdraw(ctx context.Context, req pchannel.AdjudicatorReq, stateMap pchannel.StateMap) error {
	log.Println("Withdraw called by Adjudicator")

	chanControl, errChanState := a.cb.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
	if errChanState != nil {
		return errChanState
	}

	if chanControl.Control.Closed {
		log.Println("Channel is already closed")
		if ((chanControl.Control.WithdrawnA || a.oneWithdrawer) && req.Idx == 0) || (req.Idx == 1 && chanControl.Control.WithdrawnB) {
			log.Println("Channel is already withdrawn")
			return nil
		}
		return a.handleWithdrawal(ctx, req)
	}

	//nolint:nestif
	if req.Tx.State.IsFinal {
		log.Println("Channel is final, closing now")
		withdrawSelf := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][req.Idx], req.Tx.State.Balances[1][req.Idx]}, req.Tx.State.Assets)
		withdrawOther := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][1-req.Idx], req.Tx.State.Balances[1][1-req.Idx]}, req.Tx.State.Assets)
		if req.Idx == 0 && a.oneWithdrawer && (!withdrawSelf || !withdrawOther) { // If one participant does not need to withdraw, the swap is cross-chain which means A does not need to close
			log.Println("A only closes when A & B have to withdraw")
			return nil
		}
		err := a.cb.Close(ctx, a.perunAddr, req.Tx.State, req.Tx.Sigs)
		if err != nil {
			chanControl, errChanState = a.cb.GetChannelInfo(ctx, a.perunAddr, req.Tx.State.ID)
			if errChanState != nil {
				log.Println("Error getting channel info: ", errChanState)
				return errChanState
			}

			if chanControl.Control.Closed {
				if a.oneWithdrawer && req.Idx == 0 {
					log.Println("Channel is already closed, A returns nil")
					return nil
				}
				return a.handleWithdrawal(ctx, req)
			}
			log.Println("Error closing channel: ", err)
			return err
		}
		log.Println("closed channel, ", err)
		return err
	}

	if err := a.cb.ForceClose(ctx, a.perunAddr, req.Tx.State, req.Tx.Sigs); err != nil {
		log.Println("ForceClose called")
		if errors.Is(err, ErrChannelAlreadyClosed) {
			return a.handleWithdrawal(ctx, req)
		}
		return err
	}

	log.Println("ForceClose called")
	return a.handleWithdrawal(ctx, req)
}

func (a Adjudicator) Progress(ctx context.Context, req pchannel.ProgressReq) error {
	return nil // Only used in AppChannel
}

func (a Adjudicator) Subscribe(ctx context.Context, id pchannel.ID) (pchannel.AdjudicatorSubscription, error) {
	return NewAdjudicatorSubFromChannelID(ctx, id), nil
}

func (a *Adjudicator) handleWithdrawal(ctx context.Context, req pchannel.AdjudicatorReq) error {
	withdrawOther := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][1-req.Idx], req.Tx.State.Balances[1][1-req.Idx]}, req.Tx.State.Assets)
	if a.oneWithdrawer && withdrawOther {
		log.Println("Withdrawing other", req.Idx)
		if err := a.withdrawOther(ctx, req); err != nil {
			log.Println("Error withdrawing other: ", err)
			return a.withdraw(ctx, req)
		}
	}
	withdrawSelf := needWithdraw([]pchannel.Bal{req.Tx.State.Balances[0][req.Idx], req.Tx.State.Balances[1][req.Idx]}, req.Tx.State.Assets)
	if withdrawSelf {
		log.Println("Withdrawing self", req.Idx)
		return a.withdraw(ctx, req)
	}
	return nil
}

func (a *Adjudicator) withdraw(ctx context.Context, req pchannel.AdjudicatorReq) error {
	perunAddress := a.perunAddr

	withdrawerIdx := req.Idx == 1

	return a.cb.Withdraw(ctx, perunAddress, req, withdrawerIdx, a.oneWithdrawer)
}

func (a *Adjudicator) withdrawOther(ctx context.Context, req pchannel.AdjudicatorReq) error {
	perunAddress := a.perunAddr

	return a.cb.Withdraw(ctx, perunAddress, req, false, true)
}

func needWithdraw(balances []pchannel.Bal, assets []pchannel.Asset) bool {
	for i, bal := range balances {
		_, ok := assets[i].(*channel.SolanaCrossAsset)
		if bal.Cmp(big.NewInt(0)) != 0 && ok { // if balance is larger than 0 and asset is a solana asset, participant needs to withdraw
			return true
		}
	}
	return false
}
