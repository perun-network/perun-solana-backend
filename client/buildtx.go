package client

import (
	"context"
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

type Sender interface {
	SendTx(context.Context, *solana.Transaction) (solana.Signature, error)
	SendAndConfirmTx(context.Context, *solana.Transaction, *ws.Client) (solana.Signature, error)
	SetRPCClient(*rpc.Client) error
	GetRPCClient() *rpc.Client
}

type TxSender struct {
	rpcClient *rpc.Client // The RPC client used to send transactions.
}

func NewTxSender(rpcClient *rpc.Client) *TxSender {
	return &TxSender{
		rpcClient: rpcClient,
	}
}

func (s *TxSender) SetRPCClient(rpcClient *rpc.Client) error {
	if rpcClient == nil {
		return errors.New("RPC client cannot be nil")
	}
	s.rpcClient = rpcClient
	return nil
}

func (s *TxSender) GetRPCClient() *rpc.Client {
	if s.rpcClient == nil {
		s.rpcClient = rpc.New(rpc.LocalNet_RPC) // Default to LocalNet_RPC if no client is set.
	}
	return s.rpcClient
}

func (s *TxSender) SendTx(ctx context.Context, tx *solana.Transaction) (solana.Signature, error) {
	sig, err := s.rpcClient.SendTransaction(
		ctx,
		tx,
	)
	return sig, err
}

func (s *TxSender) SendAndConfirmTx(ctx context.Context, tx *solana.Transaction, wsClient *ws.Client) (solana.Signature, error) {
	sig, err := confirm.SendAndConfirmTransaction(
		ctx,
		s.rpcClient,
		wsClient,
		tx,
	)
	return sig, err
}
