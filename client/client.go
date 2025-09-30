package client

import (
	"context"
	"log"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/perun-network/perun-solana-backend/channel/event"
	"github.com/perun-network/perun-solana-backend/encoding"
	"github.com/pkg/errors"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
)

// ErrCouldNotDecodeTx is returned when the tx could not be decoded.
var ErrCouldNotDecodeTx = errors.New("could not decode tx output")

// SolanaClient provides functions to interact with the Solana blockchain.
// It includes methods for opening, aborting, funding, disputing, closing, and force closing channels.
type SolanaClient interface {
	Open(ctx context.Context, perunAddr solana.PublicKey, params *pchannel.Params, state *pchannel.State) error
	Abort(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset) error
	Fund(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset, funderIdx bool) error
	Dispute(ctx context.Context, perunAddr solana.PublicKey, state *pchannel.State, sigs []pwallet.Sig) error
	Close(ctx context.Context, perunAddr solana.PublicKey, state *pchannel.State, sigs []pwallet.Sig) error
	ForceClose(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID) error
	GetChannelInfo(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID) (encoding.Channel, error)
}

var _ SolanaClient = (*ContractBackend)(nil)

func (cb *ContractBackend) Open(ctx context.Context, perunAddr solana.PublicKey, params *pchannel.Params, state *pchannel.State) error {
	log.Println("Open called by contract backend")
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
	_, err = cb.InvokeAndConfirmSignedTx(ctx, openTx)
	if err != nil {
		return errors.Wrap(err, "Open: could not invoke signed transaction")
	}
	return nil
}

func (cb *ContractBackend) Abort(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset) error {
	log.Println("Abort called by contract backend")
	rpcClient := cb.signer.sender.GetRPCClient()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "Abort: could not get latest blockhash")
	}

	channel, err := cb.GetChannelInfo(ctx, perunAddr, chanID)
	if err != nil {
		return errors.Wrap(err, "Abort: could not get channel info")
	}
	creator := solana.PublicKey(channel.Control.Creator)

	abortIx, err := cb.NewAbortInstruction(perunAddr, chanID, assets, creator)
	if err != nil {
		return errors.Wrap(err, "Abort: could not create abort instruction")
	}
	abortTx, err := solana.NewTransaction(
		[]solana.Instruction{abortIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "Abort: could not create transaction")
	}
	_, err = cb.InvokeAndConfirmSignedTx(ctx, abortTx)
	if err != nil {
		return errors.Wrap(err, "Abort: could not invoke signed transaction")
	}

	return nil
}

func (cb *ContractBackend) Fund(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset, funderIdx bool) error {
	log.Println("Fund called by contract backend")
	rpcClient := cb.signer.sender.GetRPCClient()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "Fund: could not get latest blockhash")
	}

	fundIx, err := cb.NewFundInstruction(perunAddr, chanID, assets, funderIdx)
	if err != nil {
		return errors.Wrap(err, "Fund: could not create fund instruction")
	}
	fundTx, err := solana.NewTransaction(
		[]solana.Instruction{fundIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "Fund: could not create transaction")
	}
	_, err = cb.InvokeAndConfirmSignedTx(ctx, fundTx)
	if err != nil {
		return errors.Wrap(err, "Fund: could not invoke signed transaction")
	}
	return nil
}

func (cb *ContractBackend) Dispute(ctx context.Context, perunAddr solana.PublicKey, state *pchannel.State, sigs []pwallet.Sig) error {
	log.Println("Dispute called by contract backend")

	rpcClient := cb.signer.sender.GetRPCClient()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "Dispute: could not get latest blockhash")
	}

	disputeIx, err := cb.NewDisputeInstruction(perunAddr, state, sigs)
	if err != nil {
		return errors.Wrap(err, "Dispute: could not create dispute instruction")
	}
	disputeTx, err := solana.NewTransaction(
		[]solana.Instruction{disputeIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "Dispute: could not create transaction")
	}
	_, err = cb.InvokeAndConfirmSignedTx(ctx, disputeTx)
	if err != nil {
		return errors.Wrap(err, "Dispute: could not invoke signed transaction")
	}

	return nil
}

func (cb *ContractBackend) Close(ctx context.Context, perunAddr solana.PublicKey, state *pchannel.State, sigs []pwallet.Sig) error {
	log.Println("Close called by contract backend")

	rpcClient := cb.signer.sender.GetRPCClient()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "Close: could not get latest blockhash")
	}

	closeIx, err := cb.NewCloseInstruction(perunAddr, state, sigs)
	if err != nil {
		return errors.Wrap(err, "Close: could not create close instruction")
	}
	closeTx, err := solana.NewTransaction(
		[]solana.Instruction{closeIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "Close: could not create transaction")
	}
	_, err = cb.InvokeAndConfirmSignedTx(ctx, closeTx)
	if err != nil {
		return errors.Wrap(err, "Close: could not invoke signed transaction")
	}

	return nil
}

func (cb *ContractBackend) ForceClose(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID) error {
	log.Println("ForceClose called by contract backend")

	rpcClient := cb.signer.sender.GetRPCClient()
	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "ForceClose: could not get latest blockhash")
	}

	forceCloseIx, err := cb.NewForceCloseInstruction(perunAddr, chanID)
	if err != nil {
		return errors.Wrap(err, "ForceClose: could not create force close instruction")
	}
	forceCloseTx, err := solana.NewTransaction(
		[]solana.Instruction{forceCloseIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "ForceClose: could not create transaction")
	}
	_, err = cb.InvokeAndConfirmSignedTx(ctx, forceCloseTx)
	if err != nil {
		return errors.Wrap(err, "ForceClose: could not invoke signed transaction")
	}

	return nil
}

// Withdraw withdraws the funds from the channel.
//
//nolint:funlen
func (cb *ContractBackend) Withdraw(ctx context.Context, perunAddr solana.PublicKey, req pchannel.AdjudicatorReq, withdrawerIdx bool, oneWithdrawer bool) error {
	log.Println("Withdraw called by contract backend")

	rpcClient := cb.signer.sender.GetRPCClient()

	recent, err := rpcClient.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return errors.Wrap(err, "Withdraw: could not get latest blockhash")
	}

	chanID := req.Tx.State.ID
	chanInfo, err := cb.GetChannelInfo(ctx, perunAddr, chanID)
	if err != nil {
		return errors.Wrap(err, "Withdraw: could not get channel info")
	}
	creator := solana.PublicKey(chanInfo.Control.Creator)
	assets := req.Tx.State.Assets
	withdrawIx, err := cb.NewWithdrawInstruction(perunAddr, chanID, assets, withdrawerIdx, oneWithdrawer, creator)
	if err != nil {
		return errors.Wrap(err, "Withdraw: could not create withdraw instruction")
	}

	withdrawTx, err := solana.NewTransaction(
		[]solana.Instruction{withdrawIx},
		recent.Value.Blockhash,
		solana.TransactionPayer(cb.signer.privateKey.PublicKey()),
	)
	if err != nil {
		return errors.Wrap(err, "Withdraw: could not create transaction")
	}
	_, err = cb.InvokeAndConfirmSignedTx(ctx, withdrawTx)
	if err != nil {
		return errors.Wrap(err, "Withdraw: could not invoke signed transaction")
	}

	// Check after withdraw.
	for _, asset := range assets {
		if solanaAsset, ok := asset.(*channel.SolanaCrossAsset); ok {
			// If asset is a solana asset, check balance after withdraw.
			mint, err := encoding.MakeAddress(solanaAsset)
			if err != nil {
				return errors.Wrap(err, "Withdraw: could not make address for asset")
			}

			bal, err := cb.GetBalance(mint)
			if err != nil {
				return errors.Wrap(err, "Withdraw: could not fetch balance after withdraw")
			}
			log.Println("Balance: ", bal, " after withdrawing: ", cb.signer.participant.SolanaAddress, solanaAsset)
		}
	}

	chanInfoAfterWithdrawn, err := cb.GetChannelInfo(ctx, perunAddr, chanID)
	if err != nil {
		return errors.Wrap(err, "Withdraw: could not get channel info after withdraw")
	}
	if (withdrawerIdx && chanInfoAfterWithdrawn.Control.WithdrawnB) || (!withdrawerIdx && chanInfoAfterWithdrawn.Control.WithdrawnA) {
		return nil
	}
	return event.ErrNoWithdrawEvent
}

func (cb *ContractBackend) GetChannelInfo(ctx context.Context, perunAddr solana.PublicKey, chanID pchannel.ID) (encoding.Channel, error) {
	channelPDA, err := ChannelPDA(chanID, perunAddr)
	if err != nil {
		return encoding.Channel{}, errors.Wrap(err, "GetChannelInfo: could not get channel PDA")
	}
	rpcClient := cb.signer.sender.GetRPCClient()
	accountInfo, err := rpcClient.GetAccountInfoWithOpts(
		ctx,
		channelPDA,
		&rpc.GetAccountInfoOpts{
			Commitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		return encoding.Channel{}, errors.Wrap(err, "GetChannelInfo: could not get account info")
	}
	if accountInfo == nil {
		return encoding.Channel{}, errors.New("GetChannelInfo: account info is nil")
	}
	borshDec := bin.NewBorshDecoder(accountInfo.Value.Data.GetBinary())
	var channel encoding.Channel
	if err := borshDec.Decode(&channel); err != nil {
		return encoding.Channel{}, errors.Wrap(ErrCouldNotDecodeTx, "GetChannelInfo: could not decode channel data")
	}
	return channel, nil
}
