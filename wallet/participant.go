package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gagliardetto/solana-go"
	"github.com/perun-network/perun-solana-backend/channel"
	"perun.network/go-perun/wallet"
)

// Participant is the backend's version of the on-chain participant in the Perun smart contract on Solana.
type Participant struct {
	// SolanaAddress is the on-chain Solana address of the participant.
	SolanaAddress solana.PublicKey
	// PublicKey is the public key of the participant, which is used to verify signatures on channel state.
	PubKey *ecdsa.PublicKey
	// CCAddr is the cross-chain address of the participant.
	CCAddr [CCAddressLength]byte
}

// NewParticipant creates a new participant with the given Stellar address, public key, and cross-chain address.
func NewParticipant(addr solana.PublicKey, pk *ecdsa.PublicKey, ccAddr [CCAddressLength]byte) *Participant {
	return &Participant{
		SolanaAddress: addr,
		PubKey:        pk,
		CCAddr:        ccAddr,
	}
}

// MarshalBinary encodes the participant into binary form.
func (p Participant) MarshalBinary() (data []byte, err error) {
	// Marshal the Stellar public key using secp256k1's raw byte format (uncompressed)
	//nolint:staticcheck
	pubKeyBytes := elliptic.Marshal(p.PubKey.Curve, p.PubKey.X, p.PubKey.Y)

	// Allocate and assemble buffer: [pubkey(65)] [sol_addr_len(1)] [sol_addr(n)] [ccaddr(fixed)]
	res := make([]byte, 0, 65+32+CCAddressLength)
	res = append(res, pubKeyBytes...)
	res = append(res, p.SolanaAddress[:]...) // Fixed-length Solana address
	res = append(res, p.CCAddr[:]...)        // Fixed-length CCAddr

	return res, nil
}

// UnmarshalBinary decodes the participant from binary form.
func (p *Participant) UnmarshalBinary(data []byte) error {
	if len(data) != 65+32+CCAddressLength {
		return fmt.Errorf("data too short to contain participant")
	}

	// Parse ECDSA public key
	x, y := elliptic.Unmarshal(secp256k1.S256(), data[:65])
	if x == nil || y == nil {
		return fmt.Errorf("failed to unmarshal ECDSA public key")
	}
	p.PubKey = &ecdsa.PublicKey{
		Curve: secp256k1.S256(),
		X:     x,
		Y:     y,
	}

	// Parse Solana address
	copy(p.SolanaAddress[:], data[65:97]) // 32 bytes for Solana address

	// Parse CCAddr
	copy(p.CCAddr[:], data[97:117]) // CCAddr is fixed length, 20 bytes

	return nil
}

// String returns the string representation of the participant as [ParticipantAddress string]:[public key hex].
func (p Participant) String() string {
	return p.AddressString() // + ":" + p.PublicKeyString()
}

// AddressString returns the Stellar address as a string.
func (p Participant) AddressString() string {
	return p.SolanaAddress.String()
}

// BackendID returns the Stellar backend ID.
func (p Participant) BackendID() wallet.BackendID {
	return channel.BackendID
}

// Equal checks if the given address is equal to the participant.
func (p Participant) Equal(other wallet.Address) bool {
	otherAddress, ok := other.(*Participant)
	if !ok {
		return false
	}
	return p.SolanaAddress == otherAddress.SolanaAddress && p.PubKey.Equal(otherAddress.PubKey) && p.CCAddr == otherAddress.CCAddr
}

// AsParticipant casts the given address to a participant.
func AsParticipant(address wallet.Address) *Participant {
	p, ok := address.(*Participant)
	if !ok {
		panic("ParticipantAddress has invalid type")
	}
	return p
}

// ToParticipant casts the given address to a participant.
func ToParticipant(address wallet.Address) (*Participant, error) {
	p, ok := address.(*Participant)
	if !ok {
		return nil, fmt.Errorf("address has invalid type")
	}
	return p, nil
}
