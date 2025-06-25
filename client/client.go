package client

import (
	"context"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/pkg/errors"
	pchannel "perun.network/go-perun/channel"
)

// ErrCouldNotDecodeTx is returned when the tx could not be decoded.
var ErrCouldNotDecodeTx = errors.New("could not decode tx output")

// SolanaClient provides functions to interact with the Solana blockchain.
// It includes methods for opening, aborting, funding, disputing, closing, and force closing channels.
type SolanaClient interface {
	Open(ctx context.Context, perunAddr solana.PublicKey, params *pchannel.Params, state *pchannel.State) error
	Abort(ctx context.Context) error
	Fund(ctx context.Context) error
	Dispute(ctx context.Context) error
	Close(ctx context.Context) error
	ForceClose(ctx context.Context) error
}

var _ SolanaClient = (*ContractBackend)(nil)

func (cb *ContractBackend) Open(ctx context.Context, perunAddr solana.PublicKey, params *pchannel.Params, state *pchannel.State) error {
	rpcClient := cb.signer.sender.GetRPCClient()
	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "Open: could not get latest blockhash")
	}

	openIx, err := cb.NewOpenInstruction(perunAddr, params, state)
	if err != nil {
		return errors.Wrap(err, "Open: could not create open instruction")
	}

	openTx, err := solana.NewTransaction(
		[]solana.Instruction{openIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "Open: could not create transaction")
	}
	_, err = cb.InvokeSignedTx(ctx, openTx)
	if err != nil {
		return errors.Wrap(err, "Open: could not invoke signed transaction")
	}
	return nil
}

func (cb *ContractBackend) Abort(ctx context.Context) error {
	return nil //TODO
}

func (cb *ContractBackend) Fund(ctx context.Context) error {
	return nil //TODO
}

func (cb *ContractBackend) Dispute(ctx context.Context) error {
	return nil //TODO
}

func (cb *ContractBackend) Close(ctx context.Context) error {
	return nil //TODO
}

func (cb *ContractBackend) ForceClose(ctx context.Context) error {
	return nil //TODO
}
