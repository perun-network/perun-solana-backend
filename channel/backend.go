package channel

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"perun.network/go-perun/channel"
	"perun.network/go-perun/wallet"
)

const (
	BackendID    = 6
	ethBackendID = 1
)

type backend struct{}

var Backend = backend{}

func init() {
	channel.SetBackend(Backend, BackendID)
}

// CalcID calculates the channel ID from the channel parameters.
func (b backend) CalcID(params *channel.Params) (channel.ID, error) {
	p, err := ToEthParams(params)
	if err != nil {
		return channel.ID{}, errors.WithMessage(err, "could not convert params")
	}
	bytes, err := EncodeChannelParams(&p)
	if err != nil {
		return channel.ID{}, errors.WithMessage(err, "could not encode params")
	}
	// Hash encoded params.
	return crypto.Keccak256Hash(bytes), nil
}

// NewAppID creates a new Solana app ID.
func (b backend) NewAppID() (channel.AppID, error) {
	panic("no app support for Solana backend")
}

// NewAsset creates a new Solana asset.
func (b backend) NewAsset() channel.Asset {
	return NewSOLAsset()
}

// Sign signs the channel state with the account.
func (b backend) Sign(account wallet.Account, state *channel.State) (wallet.Sig, error) {
	if err := checkBackends(state.Allocation.Backends); err != nil {
		return nil, errors.New("invalid backends in state allocation: " + err.Error())
	}

	ethState := ToEthState(state)

	bytes, err := EncodeEthState(&ethState)
	if err != nil {
		return nil, err
	}
	sig, err := account.SignData(bytes)
	if err != nil {
		return nil, err
	}
	return sig, err
}

// Verify verifies the signature of the channel state.
func (b backend) Verify(addr wallet.Address, state *channel.State, sig wallet.Sig) (bool, error) {
	ethState := ToEthState(state)
	bytes, err := EncodeEthState(&ethState)
	if err != nil {
		return false, err
	}
	return wallet.VerifySignature(bytes, sig, addr)
}

func checkBackends(backends []wallet.BackendID) error {
	if len(backends) == 0 {
		return errors.New("backends slice is empty")
	}

	hasSolanaBackend := false

	for _, backend := range backends {
		if backend == BackendID {
			hasSolanaBackend = true
		}
	}

	if !hasSolanaBackend {
		return errors.New("SolanaBackendID not found in backends")
	}

	return nil
}
