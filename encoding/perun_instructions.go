package encoding

import (
	"bytes"

	bin "github.com/gagliardetto/binary"
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
		Enum: bin.BorshEnum(1),
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
