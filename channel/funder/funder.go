package funder

import (
	"context"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/perun-network/perun-solana-backend/client"
	"github.com/perun-network/perun-solana-backend/encoding"

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
	time.Sleep(2 * f.pollingInterval) // Wait for the channel to be opened
	channelInfo, err := f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
	if err != nil {
		log.Println("Error while getting channel info: ", err)
		return err
	}
	log.Println("channel opened: ", channelInfo)

	return f.fundParty(ctx, req)
}

func (f *Funder) fundParty(ctx context.Context, req pchannel.FundingReq) error {
	party := getPartyByIndex(req.Idx)

	log.Printf("%s: Funding channel...", party)
	for i := 0; i < f.maxIters; i++ {
		select {
		case <-ctx.Done():
			timeoutErr := makeTimeoutErr([]pchannel.Index{req.Idx}, 0)
			errAbort := f.AbortChannel(ctx, req.State)
			log.Printf("%s: Aborting channel due to timeout...", party)
			if errAbort != nil {
				return errAbort
			}
			return timeoutErr

		case <-time.After(f.pollingInterval):

			log.Printf("%s: Polling for opened channel...", party)
			chanState, err := f.cb.GetChannelInfo(ctx, f.perunAddr, req.State.ID)
			if err != nil {
				log.Printf("%s: Error while polling for opened channel: %v", party, err)
				continue
			}

			log.Printf("%s: Found opened channel!", party)
			if chanState.Control.FundedA && chanState.Control.FundedB {
				return nil
			}

			if req.Idx == pchannel.Index(0) && !chanState.Control.FundedA { //nolint:nestif
				shouldFund := needFunding(req.State.Balances[0], req.State.Assets)
				if !shouldFund {
					log.Println("Party A does not need to fund")
					return nil
				}
				err := f.FundChannel(ctx, req.State, false)
				if err != nil {
					return err
				}
				bal0 := "bal0"
				bal1 := "bal1"
				t0, ok := req.State.Assets[0].(*channel.SolanaCrossAsset)
				if ok {
					aAdr0, err := channel.MakeAssetAddress(t0.Asset)
					if err != nil {
						return err
					}
					for {
						bal0, err = f.cb.GetBalance(aAdr0)
						if err != nil {
							log.Println("Error while getting balance: ", err)
						}
						if bal0 != "" {
							break
						}
						time.Sleep(1 * time.Second) // Wait for a second before retrying
					}
				}
				t1, ok := req.State.Assets[1].(*channel.SolanaCrossAsset)
				if ok {
					cAdr1, err := channel.MakeAssetAddress(t1.Asset)
					if err != nil {
						return err
					}
					bal1, err = f.cb.GetBalance(cAdr1)
					if err != nil {
						log.Println("Error while getting balance: ", err)
					}
				}
				log.Println("Balance A: ", bal0, bal1, " after funding amount: ", req.State.Balances, req.State.Assets)
				continue
			}
			//nolint:nestif
			if req.Idx == pchannel.Index(1) && !chanState.Control.FundedB && (chanState.Control.FundedA || !needFunding(req.State.Balances[0], req.State.Assets)) { // If party A has funded or does not need to fund, party B funds
				log.Println("Funding party B")
				shouldFund := needFunding(req.State.Balances[1], req.State.Assets)
				if !shouldFund {
					log.Println("Party B does not need to fund", req.State.Balances[1], req.State.Assets)
					return nil
				}
				err := f.FundChannel(ctx, req.State, true)
				if err != nil {
					return err
				}
				bal0 := "bal0"
				bal1 := "bal1"
				t0, ok := req.State.Assets[0].(*channel.SolanaCrossAsset)
				if ok {
					cAdr0, err := channel.MakeAssetAddress(t0.Asset)
					if err != nil {
						return err
					}
					bal0, err = f.cb.GetBalance(cAdr0)
					if err != nil {
						log.Println("Error while getting balance: ", err)
					}
				}
				t1, ok := req.State.Assets[1].(*channel.SolanaCrossAsset)
				if ok {
					cAdr1, err := channel.MakeAssetAddress(t1.Asset)
					if err != nil {
						return err
					}
					bal1, err = f.cb.GetBalance(cAdr1)
					if err != nil {
						log.Println("Error while getting balance: ", err)
					}
				}
				log.Println("Balance B: ", bal0, bal1, " after funding amount: ", req.State.Balances, req.State.Assets)
				continue
			}
		}
	}
	return f.AbortChannel(ctx, req.State)
}

// AbortChannel aborts the channel with the given state.
func (f *Funder) AbortChannel(ctx context.Context, state *pchannel.State) error {
	return f.cb.Abort(ctx)
}

func (f *Funder) openChannel(ctx context.Context, req pchannel.FundingReq) error {
	err := f.cb.Open(ctx, f.perunAddr, req.Params, req.State)
	if err != nil {
		return errors.Join(errors.New("error while opening channel in party A"), err)
	}

	return nil
}

// FundChannel funds the channel with the given state.
func (f *Funder) FundChannel(ctx context.Context, state *pchannel.State, funderIdx bool) error {
	balsSolana, err := encoding.MakeBalances(state.Allocation)
	if err != nil {
		return errors.New("error while making balances")
	}

	if !containsAllAssets(balsSolana.Tokens, f.assetAddrs) {
		return errors.New("asset address is not equal to the address stored in the state")
	}

	return f.cb.Fund(ctx, f.perunAddr, state.ID, funderIdx)
}

func getPartyByIndex(funderIdx pchannel.Index) string {
	if funderIdx == 1 {
		return "Party B"
	}
	return "Party A"
}

// makeTimeoutErr returns a FundingTimeoutError for a specific Asset for a specific Funder.
func makeTimeoutErr(remains []pchannel.Index, assetIdx int) error {
	indices := make([]pchannel.Index, 0, len(remains))

	indices = append(indices, remains...)

	return pchannel.NewFundingTimeoutError(
		[]*pchannel.AssetFundingError{{
			Asset:         pchannel.Index(assetIdx),
			TimedOutPeers: indices,
		}},
	)
}

// Function to check if all assets in state.Allocation are present in f.assetAddrs.
func containsAllAssets(stateAssets []encoding.CrossAsset, fAssets []solana.PublicKey) bool {
	fAssetSet := assetSliceToSet(fAssets)

	for _, asset := range stateAssets {
		assetVal := asset.SolanaAddress
		if _, found := fAssetSet[assetVal.String()]; found { // if just one Asset was found, we continue
			return true
		}
	}

	return false
}

// Helper function to convert a slice of Asset to a set (map for fast lookup).
func assetSliceToSet(assets []solana.PublicKey) map[string]struct{} {
	assetSet := make(map[string]struct{})
	for _, asset := range assets {
		assetSet[asset.String()] = struct{}{}
	}
	return assetSet
}

// needFunding checks if a participant needs to fund the channel.
func needFunding(balances []pchannel.Bal, assets []pchannel.Asset) bool {
	for i, bal := range balances {
		_, ok := assets[i].(*channel.SolanaCrossAsset)
		if bal.Cmp(big.NewInt(0)) != 0 && ok { // if balance is non 0 and asset is a solana asset, participant needs to fund
			return true
		}
	}
	return false
}
