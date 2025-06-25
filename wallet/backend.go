package wallet

import (
	"io"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/crypto"
	"perun.network/go-perun/wallet"
	"perun.network/go-perun/wire/perunio"
	"perun.network/perun-solana-backend/channel"
)

// SignatureLength is the length of a signature in bytes.
const SignatureLength = 64

type backend struct{}

var Backend = backend{}

func init() {
	wallet.SetBackend(Backend, channel.BackendID)
}

// NewAddress creates a new address.
func (b backend) NewAddress() wallet.Address {
	return &Participant{}
}

// DecodeSig decodes a signature of length SignatureLength from the reader.
func (b backend) DecodeSig(reader io.Reader) (wallet.Sig, error) {
	buf := make(wallet.Sig, 65) //nolint:gomnd
	return buf, perunio.Decode(reader, &buf)
}

// VerifySignature verifies the signature of a message.
func (b backend) VerifySignature(msg []byte, sig wallet.Sig, a wallet.Address) (bool, error) {
	p, ok := a.(*Participant)
	if !ok {
		return false, errors.New("participant has invalid type")
	}
	hash := crypto.Keccak256(msg)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	hash = crypto.Keccak256(prefix, hash)
	sigCopy := make([]byte, 65) //nolint:gomnd
	copy(sigCopy, sig)
	if len(sigCopy) == 65 && (sigCopy[65-1] >= 27) { //nolint:gomnd
		sigCopy[65-1] -= 27
	}
	pk, err := crypto.SigToPub(hash, sigCopy)
	if err != nil {
		return false, errors.WithStack(err)
	}
	return pk.X.Cmp(p.PubKey.X) == 0 && pk.Y.Cmp(p.PubKey.Y) == 0 && pk.Curve == p.PubKey.Curve, nil
}
