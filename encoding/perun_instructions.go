package encoding

import (
	"bytes"

	"github.com/ethereum/go-ethereum/crypto"
	bin "github.com/gagliardetto/binary"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/pkg/errors"
	pchannel "perun.network/go-perun/channel"
)

type PerunInstruction struct {
	Enum         bin.BorshEnum `borsh_enum:"true"`
	Open         OpenInstruction
	Fund         FundInstruction
	Close        CloseInstruction
	ForceClose   ForceCloseInstruction
	Dispute      DisputeInstruction
	Withdraw     WithdrawInstruction
	AbortFunding AbortFundingInstruction
}

type OpenInstruction struct {
	Params Params
	State  ChannelState
}

type FundInstruction struct {
	ChannelID [32]byte
	PartyIdx  bool
}

type CloseInstruction struct {
	State ChannelState
	Hash  [32]byte // Hash of the state being closed
	SigA  [65]byte
	SigB  [65]byte
}

type ForceCloseInstruction struct {
	ChannelID [32]byte
}

type DisputeInstruction struct {
	State ChannelState
	SigA  [65]byte
	SigB  [65]byte
}

type WithdrawInstruction struct {
	ChannelID     [32]byte
	PartyIdx      bool
	OneWithdrawer bool
}

type AbortFundingInstruction struct {
	ChannelID [32]byte
}

func MakeOpenInstruction(params *pchannel.Params, state *pchannel.State) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	bParams, err := MakeParams(*params) // convert go-perun Params to encoding Params
	if err != nil {
		return nil, errors.Wrap(err, "failed to make params")
	}

	bState, err := MakeChannelState(*state) // convert go-perun State to encoding ChannelState
	if err != nil {
		return nil, errors.Wrap(err, "failed to make channel state")
	}

	instr := PerunInstruction{
		Enum: bin.BorshEnum(0),
		Open: OpenInstruction{
			Params: bParams,
			State:  bState,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode open instruction")
	}

	return buf.Bytes(), nil
}

func MakeFundInstruction(channelID [32]byte, partyIdx bool) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	instr := PerunInstruction{
		Enum: 1,
		Fund: FundInstruction{
			ChannelID: channelID,
			PartyIdx:  partyIdx,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode fund instruction")
	}

	return buf.Bytes(), nil
}

func MakeAbortInstruction(channelID [32]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	instr := PerunInstruction{
		Enum: 6,
		AbortFunding: AbortFundingInstruction{
			ChannelID: channelID,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode abort instruction")
	}

	return buf.Bytes(), nil
}

func MakeCloseInstruction(state *pchannel.State, sigA, sigB [65]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	bState, err := MakeChannelState(*state) // convert go-perun State to encoding ChannelState
	if err != nil {
		return nil, errors.Wrap(err, "failed to make channel state")
	}

	ethState := channel.ToEthState(state)
	bytes, err := channel.EncodeEthState(&ethState)
	if err != nil {
		return nil, err
	}
	hash := crypto.Keccak256(bytes)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	hash = crypto.Keccak256(prefix, hash)
	var hashArr [32]byte
	copy(hashArr[:], hash)

	instr := PerunInstruction{
		Enum: 2,
		Close: CloseInstruction{
			State: bState,
			Hash:  hashArr,
			SigA:  sigA,
			SigB:  sigB,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode close instruction")
	}

	return buf.Bytes(), nil
}

func MakeForceCloseInstruction(channelID [32]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	instr := PerunInstruction{
		Enum: 3,
		ForceClose: ForceCloseInstruction{
			ChannelID: channelID,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode force close instruction")
	}

	return buf.Bytes(), nil
}

func MakeDisputeInstruction(state *pchannel.State, sigA, sigB [65]byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	bState, err := MakeChannelState(*state)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make channel state")
	}

	instr := PerunInstruction{
		Enum: 4,
		Dispute: DisputeInstruction{
			State: bState,
			SigA:  sigA,
			SigB:  sigB,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode dispute instruction")
	}

	return buf.Bytes(), nil
}

func MakeWithdrawInstruction(channelID [32]byte, partyIdx, oneWithdrawer bool) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := bin.NewBorshEncoder(buf)

	instr := PerunInstruction{
		Enum: 5,
		Withdraw: WithdrawInstruction{
			ChannelID:     channelID,
			PartyIdx:      partyIdx,
			OneWithdrawer: oneWithdrawer,
		},
	}
	if err := enc.Encode(&instr); err != nil {
		return nil, errors.Wrap(err, "failed to encode withdraw instruction")
	}

	return buf.Bytes(), nil
}
