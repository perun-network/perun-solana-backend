package encoding

import (
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	pchannel "perun.network/go-perun/channel"
	"perun.network/perun-solana-backend/wallet"
)

const solanaBackendID = 6

// Control is a struct that represents the control state of a channel.
type Control struct {
	FundedA    bool
	FundedB    bool
	Closed     bool
	WithdrawnA bool
	WithdrawnB bool
	Disputed   bool
	Timestamp  uint64
}

// Channel represents a channel on the solana block-chain.
type Channel struct {
	Params  Params
	State   ChannelState
	Control Control
}

// Participant represents a participant in a Perun channel on Solana.
type Participant struct {
	SolanaAddress solana.PublicKey
	CcAddress     [20]byte
	L2Pubkey      [65]byte
}

// MakeParticipant creates a Participant from a types.Participant.
func MakeParticipant(participant wallet.Participant) (Participant, error) {
	if participant.PubKey == nil {
		return Participant{}, errors.New("invalid Stellar public key length")
	}

	if !participant.PubKey.Curve.IsOnCurve(participant.PubKey.X, participant.PubKey.Y) {
		return Participant{}, errors.New("stellar public key is not on the curve")
	}
	pk := PublicKeyToBytes(participant.PubKey)
	if len(pk) != 65 {
		return Participant{}, errors.New("invalid Stellar public key length")
	}

	var l2Pubkey [65]byte
	copy(l2Pubkey[:], pk)

	return Participant{
		SolanaAddress: participant.SolanaAddress,
		CcAddress:     participant.CCAddr,
		L2Pubkey:      l2Pubkey,
	}, nil
}

// ChannelID represents the ID of a channel on-chain.
type ChannelID struct {
	ID [32]byte
}

// Params represents the parameters of a channel on-chain.
type Params struct {
	A                 Participant
	B                 Participant
	Nonce             [32]byte
	ChallengeDuration uint64
}

// MakeParams converts a pchannel.Params to a Params.
func MakeParams(params pchannel.Params) (Params, error) {
	if !params.LedgerChannel {
		return Params{}, errors.New("expected ledger channel")
	}
	if params.VirtualChannel {
		return Params{}, errors.New("expected non-virtual channel")
	}
	if !pchannel.IsNoApp(params.App) {
		return Params{}, errors.New("expected no app")
	}
	if len(params.Parts) != 2 { //nolint:gomnd
		return Params{}, errors.New("expected exactly two participants")
	}

	participantA, err := wallet.ToParticipant(params.Parts[0][solanaBackendID])
	if err != nil {
		return Params{}, err
	}
	a, err := MakeParticipant(*participantA)
	if err != nil {
		return Params{}, err
	}

	participantB, err := wallet.ToParticipant(params.Parts[1][solanaBackendID])
	if err != nil {
		return Params{}, err
	}
	b, err := MakeParticipant(*participantB)
	if err != nil {
		return Params{}, err
	}
	nonce := MakeNonce(params.Nonce)
	return Params{
		A:                 a,
		B:                 b,
		Nonce:             nonce,
		ChallengeDuration: params.ChallengeDuration,
	}, nil
}

// ChannelState represents the state of a channel on-chain.
type ChannelState struct {
	ChannelID [32]byte
	Balances  Balances
	Version   uint64
	Finalized bool
}

// MakeChannelState converts a pchannel.State to a ChannelState.
func MakeChannelState(state pchannel.State) (ChannelState, error) {
	if err := state.Valid(); err != nil {
		return ChannelState{}, err
	}
	if !channel.IsNoApp(state.App) {
		return ChannelState{}, errors.New("expected NoApp")
	}
	if !channel.IsNoData(state.Data) {
		return ChannelState{}, errors.New("expected NoData")
	}
	balances, err := MakeBalances(state.Allocation)
	if err != nil {
		return ChannelState{}, err
	}
	var channelID [32]byte
	copy(channelID[:], state.ID[:])
	return ChannelState{
		ChannelID: channelID,
		Balances:  balances,
		Version:   state.Version,
		Finalized: state.IsFinal,
	}, nil
}

// Balances represents the balances of the channel on-chain.
type Balances struct {
	Tokens []CrossAsset
	BalA   []uint64
	BalB   []uint64
}

// MakeBalances converts a pchannel.Allocation to Balances.
func MakeBalances(alloc pchannel.Allocation) (Balances, error) {
	if err := alloc.Valid(); err != nil {
		return Balances{}, err
	}
	if len(alloc.Locked) != 0 {
		return Balances{}, errors.New("expected no locked funds")
	}
	assets := alloc.Assets
	tokens, err := MakeTokens(assets)
	if err != nil {
		return Balances{}, err
	}

	numParts := alloc.NumParts()
	if numParts < 2 { //nolint:gomnd
		return Balances{}, errors.New("expected at least two parts")
	}
	bals := alloc.Balances

	balPartVecs := make([][]uint64, numParts)

	for _, balsAsset := range bals {
		for j, val := range balsAsset {
			balVal, err := MakeUint64(val)
			if err != nil {
				return Balances{}, err
			}

			if j < numParts {
				balPartVecs[j] = append(balPartVecs[j], balVal)
			} else {
				return Balances{}, errors.New("unexpected number of parts in balance asset")
			}
		}
	}

	// Assign the first two parts to BalA and BalB for backward compatibility
	var balAPartVec, balBPartVec []uint64
	if numParts > 0 {
		balAPartVec = balPartVecs[0]
	}
	if numParts > 1 {
		balBPartVec = balPartVecs[1]
	}

	return Balances{
		BalA:   balAPartVec,
		BalB:   balBPartVec,
		Tokens: tokens,
	}, nil
}

// MakeNonce converts a pchannel.Nonce to a [32]byte.
func MakeNonce(nonce pchannel.Nonce) [32]byte {
	var b [32]byte
	nonce.FillBytes(b[:])
	return b
}

func MakeUint64(i *big.Int) (uint64, error) {
	if i.Sign() < 0 {
		return 0, errors.New("expected non-negative balance")
	}
	if !i.IsUint64() {
		return 0, errors.New("balance too large for uint64")
	}
	return i.Uint64(), nil
}
