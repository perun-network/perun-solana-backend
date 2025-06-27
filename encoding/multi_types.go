package encoding

import (
	"crypto/ecdsa"
	"math/big"
	"strconv"

	"github.com/gagliardetto/solana-go"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/pkg/errors"

	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/channel/multi"
)

// CrossAsset represents an on-chain asset on Solana.
type CrossAsset struct {
	Chain         Chain
	SolanaAddress solana.PublicKey
	EthAddress    [20]byte
}

// MakeTokens converts a slice of pchannel.Asset to a slice of CrossAsset.
func MakeTokens(assets []pchannel.Asset) ([]CrossAsset, error) {
	tokens := make([]CrossAsset, len(assets))
	for i, ast := range assets {

		multiAsset, ok := ast.(multi.Asset)
		if !ok {
			return nil, errors.New("asset is not a multi.Asset")
		}
		id := multiAsset.LedgerBackendID().LedgerID()
		if id == nil {
			return nil, errors.New("asset does not have a LedgerID")
		}
		lidMapKey := id.MapKey()
		lidval, err := strconv.ParseUint(string(lidMapKey), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse LedgerID %s", lidMapKey)
		}
		var tokenSolanaAddrVal solana.PublicKey
		var tokenEthAddrVal [20]byte
		switch asset := ast.(type) {
		case *channel.SolanaCrossAsset:
			sa, err := channel.ToSolanaCrossAsset(asset)
			if err != nil {
				return nil, err
			}

			tokenSolanaAddrVal, err = MakeAddress(sa)
			if err != nil {
				return nil, err
			}
			defAddr := [20]byte{} //nolint:gomnd
			tokenEthAddrVal = defAddr

		case *channel.EthAsset:
			addrBytes, err := asset.AssetHolder.MarshalBinary()
			if err != nil {
				return nil, err
			}
			if len(addrBytes) != 20 {
				return nil, errors.New("invalid AssetHolder address length")
			}
			copy(tokenEthAddrVal[:], addrBytes)

			tokenSolanaAddrVal, err = randomAddress()
			if err != nil {
				return nil, err
			}

		default:
			// Assume that Asset it an ethereum asset
			ethAddress := asset.Address()
			// Check if the string is a valid length (20 byte)
			if len(ethAddress) != 20 { //nolint:gomnd
				return nil, errors.New("unexpected asset type")
			}
			copy(tokenEthAddrVal[:], ethAddress)
			tokenSolanaAddrVal, err = randomAddress()
			if err != nil {
				return nil, err
			}
		}

		tokens[i] = CrossAsset{
			Chain:         Chain(lidval),
			SolanaAddress: tokenSolanaAddrVal,
			EthAddress:    tokenEthAddrVal,
		}
	}
	return tokens, nil
}

func MakeAddress(asset *channel.SolanaCrossAsset) (solana.PublicKey, error) {
	if asset == nil {
		return solana.PublicKey{}, errors.New("asset is nil")
	}
	if asset.Asset.IsInvalid() {
		return solana.PublicKey{}, errors.New("asset is invalid")
	}
	if asset.Asset.IsSOL {
		return solana.PublicKey{}, nil // SOL does not have a mint address.
	}
	return *asset.Asset.Mint, nil
}

// Chain represents a chain identifier.
type Chain uint64

// PublicKeyToBytes convert ECDSA public key to bytes.
func PublicKeyToBytes(pubKey *ecdsa.PublicKey) []byte {
	// Get the X and Y coordinates
	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Calculate the byte lengths for fixed-width representation.
	curveBits := pubKey.Curve.Params().BitSize
	curveByteSize := (curveBits + 7) / 8 //nolint:gomnd

	// Create fixed-size byte slices for X and Y.
	xPadded := make([]byte, curveByteSize)
	yPadded := make([]byte, curveByteSize)
	copy(xPadded[curveByteSize-len(xBytes):], xBytes)
	copy(yPadded[curveByteSize-len(yBytes):], yBytes)

	// Concatenate the X and Y coordinates
	pubKeyBytes := make([]byte, 0, 65)      //nolint:gomnd
	pubKeyBytes = append(pubKeyBytes, 0x04) //nolint:gomnd
	pubKeyBytes = append(pubKeyBytes, xPadded...)
	pubKeyBytes = append(pubKeyBytes, yPadded...)
	return pubKeyBytes
}

// BytesToPublicKey convert bytes back to ECDSA public key.
func BytesToPublicKey(data []byte) (*big.Int, *big.Int, error) {
	if len(data) != 65 || data[0] != 0x04 {
		return nil, nil, errors.New("invalid public key")
	}
	// Split data into X and Y
	x := new(big.Int).SetBytes(data[1:33])
	y := new(big.Int).SetBytes(data[33:])

	// Return the public key
	return x, y, nil
}

func randomAddress() (solana.PublicKey, error) {
	// Generate a random Solana address (public key)
	privKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return solana.PublicKey{}, errors.Wrap(err, "failed to generate random Solana address")
	}
	return privKey.PublicKey(), nil
}
