package channel

import (
	"bytes"

	"github.com/gagliardetto/solana-go"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
	pmulti "perun.network/go-perun/channel/multi"
	"perun.network/go-perun/wire/perunio"
)

var (
	SOLMagic         byte = 0x00
	SPLMagic         byte = 0x01
	SolanaContractID      = "6"
)

type (
	SolanaCrossAsset struct {
		id    CCID
		Asset SolanaAsset
	}

	SolanaAsset struct {
		IsSOL bool
		Mint  *solana.PublicKey // If IsSOL is true, this is the SOL asset; otherwise, it's a token asset.
	}

	// CCID is a unique identifier for a channel asset.
	CCID struct {
		backendID uint32
		ledgerID  ContractLID
	}

	// ContractLID represents the ID of a contract on a specific chain.
	ContractLID struct{ string }
)

var _ pmulti.Asset = (*SolanaCrossAsset)(nil)

func (sa SolanaAsset) Address() []byte {
	if sa.IsSOL {
		return nil // SOL does not have a mint address in the same way tokens do.
	}
	if sa.Mint == nil {
		return nil
	}
	return sa.Mint.Bytes()
}

func (sa SolanaAsset) MarshalBinary() ([]byte, error) {
	if sa.IsSOL {
		return []byte{SOLMagic}, nil // SOL does not have a mint address, so we return an
	}
	e := sa.Mint.Bytes()
	return append([]byte{SPLMagic}, e...), nil // SPL token asset, return mint address prefixed with SPLMagic.
}

func (sa *SolanaAsset) UnmarshalBinary(data []byte) error {
	// Implement binary unmarshalling logic if needed.
	if len(data) < 1 {
		return errors.New("asset invalid: empty")
	}
	switch data[0] {
	case SOLMagic:
		sa.IsSOL = true
		sa.Mint = nil // SOL does not have a mint address.
		return nil
	case SPLMagic:
		sa.IsSOL = false
		mint := solana.PublicKeyFromBytes(data[1:])
		sa.Mint = &mint
		return nil
	default:
		return errors.Errorf("asset invalid: unknown magic byte %x", data[0])
	}
}

func (sa SolanaAsset) IsInvalid() bool {
	return (!sa.IsSOL && sa.Mint == nil) || (sa.IsSOL && sa.Mint != nil)
}

func (sa SolanaAsset) Equal(asset pchannel.Asset) bool {
	other, ok := asset.(*SolanaAsset)
	if !ok {
		return false
	}
	if sa.IsSOL != other.IsSOL {
		return false
	}
	if sa.IsSOL {
		return other.Mint == nil // Both are SOL, so they are equal if both have no mint address.
	}
	return sa.Mint.Equals(*other.Mint) // Both are token assets, compare mint addresses.
}

// IsCompatibleAsset returns the Asset if the asset is compatible with the CKB backend.
func IsCompatibleAsset(asset pchannel.Asset) (*SolanaAsset, error) {
	a, ok := asset.(*SolanaCrossAsset)
	if !ok {
		b, ok := asset.(*SolanaAsset)
		if !ok {
			return nil, errors.New("asset is not of type Asset")
		} else {
			return b, nil
		}
	}
	return &a.Asset, nil
}

func (c SolanaCrossAsset) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := perunio.Encode(&buf, c.id.ledgerID, c.id.backendID, c.Asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode SolanaCrossAsset")
	}
	return buf.Bytes(), nil
}

func (c *SolanaCrossAsset) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)
	return perunio.Decode(buf, &c.id.ledgerID, &c.id.backendID, &c.Asset)
}

func (c SolanaCrossAsset) ID() CCID {
	return c.id
}

func (c SolanaCrossAsset) Address() []byte {
	return c.Asset.Address()
}

func (c SolanaCrossAsset) Equal(asset pchannel.Asset) bool {
	_, ok := asset.(*SolanaCrossAsset)
	return ok
}

func (c SolanaCrossAsset) LedgerBackendID() pmulti.LedgerBackendID {
	return c.id
}

func ToSolanaCrossAsset(asset pchannel.Asset) (*SolanaCrossAsset, error) {
	crossAsset, ok := asset.(*SolanaCrossAsset)
	if !ok {
		return nil, errors.New("asset is not of type SolanaCrossAsset")
	}
	return crossAsset, nil
}

func NewSOLAsset() *SolanaAsset {
	return &SolanaAsset{
		IsSOL: true,
		Mint:  nil, // SOL does not have a mint address in the same way tokens do.
	}
}

func NewTokenAsset(mintAddr *solana.PublicKey) SolanaAsset {
	return SolanaAsset{
		IsSOL: false,
		Mint:  mintAddr,
	}
}

func NewSOLSolanaCrossAsset() *SolanaCrossAsset {
	solAsset := NewSOLAsset()

	return &SolanaCrossAsset{
		id: CCID{
			backendID: BackendID,
			ledgerID:  MakeContractID(SolanaContractID),
		},
		Asset: *solAsset,
	}
}

func NewTokenSolanaCrossAsset(mintAddr *solana.PublicKey, contractID ContractLID) SolanaCrossAsset {
	return SolanaCrossAsset{
		id:    MakeCCID(contractID),
		Asset: NewTokenAsset(mintAddr),
	}
}

// MakeCCID makes a CCID for the given id.
func MakeCCID(contractID ContractLID) CCID {
	return CCID{BackendID, contractID}
}

// MakeContractID makes a ChainID for the given id.
func MakeContractID(id string) ContractLID {
	return ContractLID{id}
}

// MarshalBinary encodes the ContractLID into a binary format.
func (cid ContractLID) MarshalBinary() ([]byte, error) {
	if cid.string == "" {
		return nil, errors.New("contract ID is empty")
	}
	return base58.Decode(cid.string)
}

// UnmarshalBinary decodes the binary data into a ContractLID.
func (cid *ContractLID) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		return errors.New("contract ID data is empty")
	}
	cid.string = base58.Encode(data)
	return nil
}

// BackendID returns the backend ID of the asset.
func (c CCID) BackendID() uint32 {
	return c.backendID
}

// LedgerID returns the ledger ID of the asset.
func (c CCID) LedgerID() multi.LedgerID {
	return c.ledgerID
}

// MapKey returns the asset's map key representation.
func (id ContractLID) MapKey() multi.LedgerIDMapKey {
	if id.string == "" {
		return ""
	}
	return multi.LedgerIDMapKey(id.string)
}
