package client

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/pkg/errors"

	pwallet "perun.network/go-perun/wallet"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/perun-network/perun-solana-backend/wallet"
)

const (
	defaultSolanaRPC  = rpc.LocalNet_RPC // Default Solana RPC endpoint.
	defaultSolanaWS   = rpc.LocalNet_WS  // Default Solana WebSocket endpoint.
	defaultCommitment = rpc.CommitmentFinalized
)

type SolanaSigner struct {
	privateKey  *solana.PrivateKey // The private key of the account that will be used to sign transactions.
	participant *wallet.Participant
	account     pwallet.Account // The account associated with the participant.
	sender      Sender
}

type SignerConfig struct {
	privateKey  *solana.PrivateKey
	participant *wallet.Participant
	account     pwallet.Account
	sender      Sender
	rpcURL      string // The RPC URL to connect to the Solana network.
}

func NewSignerConfig(
	privateKey *solana.PrivateKey,
	participant *wallet.Participant,
	account pwallet.Account,
	sender Sender,
	rpcURL string,
) *SignerConfig {
	if privateKey.PublicKey() != participant.SolanaAddress {
		panic("private key's public key does not match the participant's Solana address")
	}
	signerConfig := &SignerConfig{
		privateKey:  privateKey,
		participant: participant,
		account:     account,
		sender:      sender,
		rpcURL:      rpcURL,
	}
	return signerConfig
}

func NewRandomConfig(rng *rand.Rand) *SignerConfig {
	signerConfig := &SignerConfig{}

	// Generate a random account and participant.
	acc, kp, err := wallet.NewRandomAccount(rng)
	if err != nil {
		panic(err)
	}
	signerConfig.account = acc
	signerConfig.participant = acc.Participant()
	signerConfig.privateKey = kp
	// Set the default RPC URL.
	signerConfig.rpcURL = defaultSolanaRPC
	// Create a new TxSender with the default RPC URL.
	signerConfig.sender = NewTxSender(rpc.New(defaultSolanaRPC))
	return signerConfig
}

// NewSolanaSigner creates a new SolanaSigner with the provided configuration.
func NewSolanaSigner(cfg SignerConfig) *SolanaSigner {
	ss := &SolanaSigner{}

	if cfg.privateKey != nil {
		ss.privateKey = cfg.privateKey
	}
	if cfg.participant != nil {
		ss.participant = cfg.participant
	}
	if cfg.account != nil {
		ss.account = cfg.account
	}

	if cfg.sender != nil {
		ss.sender = cfg.sender
	} else {
		if cfg.rpcURL == "" {
			cfg.rpcURL = defaultSolanaRPC // Use the default RPC URL if none is provided.
		}
		ss.sender = NewTxSender(rpc.New(cfg.rpcURL))
	}

	return ss
}

// ContractBackend provides a backend for interacting with the Solana blockchain.
type ContractBackend struct {
	signer  SolanaSigner
	chainID int
	cbMutex sync.Mutex
}

// NewRandomDefaultContractBackend creates a new ContractBackend with a random signer configuration and the default chain ID.
func NewRandomDefaultContractBackend() *ContractBackend {
	rng := rand.New(rand.NewSource(rand.Int63()))
	return NewContractBackend(*NewRandomConfig(rng), channel.BackendID)
}

// NewContractBackend creates a new ContractBackend with the given signer configuration and chain ID.
func NewContractBackend(scfg SignerConfig, chainID int) *ContractBackend {
	cb := &ContractBackend{
		signer:  *NewSolanaSigner(scfg),
		chainID: chainID,
		cbMutex: sync.Mutex{},
	}

	return cb
}

func (cb *ContractBackend) InvokeSignedTx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	cb.cbMutex.Lock()
	defer cb.cbMutex.Unlock()

	_, err := tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if cb.signer.privateKey.PublicKey() == key {
				return cb.signer.privateKey
			}
			return nil
		},
	)
	if err != nil {
		return solana.Signature{}, errors.Wrap(err, "InvokeTx: could not sign transaction")
	}

	return cb.signer.sender.SendTx(ctx, tx)
}

func (cb *ContractBackend) InvokeAndConfirmSignedTx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	cb.cbMutex.Lock()
	defer cb.cbMutex.Unlock()
	wsClient, err := ws.Connect(ctx, rpc.LocalNet_WS)
	if err != nil {
		return solana.Signature{}, errors.Wrap(err, "InvokeAndConfirmTx: could not connect to WebSocket client")
	}
	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if cb.signer.privateKey.PublicKey() == key {
				return cb.signer.privateKey
			}
			return nil
		},
	)
	if err != nil {
		return solana.Signature{}, errors.Wrap(err, "InvokeAndConfirmTx: could not sign transaction")
	}

	return cb.signer.sender.SendAndConfirmTx(ctx, tx, wsClient)
}

// GetBalance returns the balance of the given asset mint.
// If the mint is the zero pubkey, it returns the SOL balance.
func (cb *ContractBackend) GetBalance(mint solana.PublicKey) (string, error) {
	ctx := context.Background()
	client := cb.signer.sender.GetRPCClient()

	// Check if mint is zero => SOL balance
	if mint.IsZero() {
		acctInfo, err := client.GetAccountInfo(ctx, cb.signer.participant.SolanaAddress)
		if err != nil {
			return "", fmt.Errorf("failed to get SOL account info: %w", err)
		}
		if acctInfo == nil || acctInfo.Value == nil {
			return "", fmt.Errorf("no SOL account data found for %s", cb.signer.participant.SolanaAddress)
		}
		return fmt.Sprintf("%d", acctInfo.Value.Lamports), nil
	}

	// Otherwise, treat it as an SPL token and get ATA balance
	ata, _, err := solana.FindAssociatedTokenAddress(cb.signer.participant.SolanaAddress, mint)
	if err != nil {
		return "", fmt.Errorf("failed to derive ATA: %w", err)
	}

	res, err := client.GetTokenAccountBalance(ctx, ata, rpc.CommitmentFinalized)
	if err != nil {
		return "", fmt.Errorf("failed to get SPL token balance: %w", err)
	}
	if res == nil || res.Value == nil {
		return "", fmt.Errorf("no SPL balance info for ATA %s", ata)
	}

	return res.Value.Amount, nil // raw string amount
}
