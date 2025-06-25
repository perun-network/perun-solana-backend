package wallet

import (
	"crypto/ecdsa"
	"math/rand"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gagliardetto/solana-go"
	"github.com/pkg/errors"
	"perun.network/go-perun/wallet"
)

const (
	CCAddressLength = 20 // Length of a cross-chain address in bytes.
)

type Account struct {
	// privateKey is the private key of the account.
	privateKey ecdsa.PrivateKey
	// ParticipantAddress references the Public Key of the Participant this account belongs to.
	ParticipantAddress solana.PublicKey
	// CCAddr is the cross-chain address of the participant.
	CCAddr [CCAddressLength]byte
}

// NewAccount creates a new account with the given private key and addresses.
func NewAccount(privateKey string, addr solana.PublicKey, ccAddresses [CCAddressLength]byte) (*Account, error) {
	// Decode the private key from the string.
	privateKeyECDSA, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(errors.Wrap(err, "NewAccount"))
	}

	return &Account{
		privateKey:         *privateKeyECDSA,
		ParticipantAddress: addr,
		CCAddr:             ccAddresses,
	}, nil
}

// NewRandomAccountWithAddress creates a new account with a random private key and the given address as
// Account.ParticipantAddress.
func NewRandomAccountWithAddress(rng *rand.Rand, addr solana.PublicKey) (*Account, error) {
	s, err := ecdsa.GenerateKey(secp256k1.S256(), rng)
	if err != nil {
		return nil, err
	}
	return &Account{privateKey: *s, ParticipantAddress: addr}, nil
}

// NewRandomAccount creates a new account with a random private key. It also creates a random key pair, using its
// address as the account'privateKey Account.ParticipantAddress.
func NewRandomAccount(rng *rand.Rand) (*Account, *solana.PrivateKey, error) {
	privKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, nil, errors.Wrap(err, "NewRandomAccount")
	}
	acc, err := NewRandomAccountWithAddress(rng, privKey.PublicKey())
	if err != nil {
		return nil, nil, errors.Wrap(err, "NewRandomAccountWithAddress")
	}
	return acc, &privKey, nil

}

// Address returns the Participant this account belongs to.
func (a Account) Address() wallet.Address {
	pubKey, ok := a.privateKey.Public().(*ecdsa.PublicKey) // Ensure correct type
	if !ok {
		panic("unexpected type for ecdsa.PublicKey")
	}
	return NewParticipant(a.ParticipantAddress, pubKey, a.CCAddr)
}

// Participant returns the Participant this account belongs to.
func (a Account) Participant() *Participant {
	return NewParticipant(a.ParticipantAddress, a.privateKey.Public().(*ecdsa.PublicKey), a.CCAddr)
}

// SignData signs the given data with the account's private key.
func (a Account) SignData(data []byte) ([]byte, error) {
	hash := crypto.Keccak256(data)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	phash := crypto.Keccak256(prefix, hash)

	sig, err := crypto.Sign(phash, &a.privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "SignHash")
	}
	sig[64] += 27
	return sig, nil
}
