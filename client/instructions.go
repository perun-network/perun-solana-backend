package client

import (
	"github.com/gagliardetto/solana-go"
	system "github.com/gagliardetto/solana-go/programs/system"
	"github.com/perun-network/perun-solana-backend/encoding"
	"github.com/pkg/errors"
	pchannel "perun.network/go-perun/channel"
)

// ChannelPDA computes the Program Derived Address (PDA) for a Perun channel on Solana.
func ChannelPDA(channelID [32]byte, perunAddr solana.PublicKey) (solana.PublicKey, error) {
	pda, _, err := solana.FindProgramAddress([][]byte{
		[]byte("channel"),
		channelID[:],
	}, perunAddr)
	if err != nil {
		return solana.PublicKey{}, errors.Wrap(err, "could not find program address for channel")
	}
	return pda, nil
}

// NewOpenInstruction creates a new Open instruction for the Perun channel.
func (cb *ContractBackend) NewOpenInstruction(perunAddr solana.PublicKey, params *pchannel.Params, state *pchannel.State) (solana.Instruction, error) {
	perunID := perunAddr // Perun program address, should be set to the actual Perun program address on Solana

	data, err := encoding.MakeOpenInstruction(params, state)
	if err != nil {
		return nil, errors.Wrap(err, "could not create open instruction")
	}

	var channelID [32]byte
	copy(channelID[:], state.ID[:])
	channelPDA, err := ChannelPDA(channelID, perunAddr)
	if err != nil {
		return nil, errors.Wrap(err, "could not get channel PDA")
	}

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(channelPDA, true, false),                         // Program account derived from channel ID
		solana.NewAccountMeta(cb.signer.participant.SolanaAddress, true, true), // Participant's account
		solana.NewAccountMeta(system.ProgramID, false, false),                  // System program account
	}

	openIx := solana.NewInstruction(
		perunID,  // Program ID
		accounts, // Accounts to be passed to the instruction
		data,     // Instruction data
	)
	return openIx, nil
}

func (cb *ContractBackend) NewFundInstruction(perunAddr solana.PublicKey, chanID pchannel.ID, funderIdx bool) (solana.Instruction, error) {
	data, err := encoding.MakeFundInstruction(chanID, funderIdx)
	if err != nil {
		return nil, errors.Wrap(err, "could not create open instruction")
	}
	var channelID [32]byte
	copy(channelID[:], chanID[:])
	channelPDA, err := ChannelPDA(channelID, perunAddr)
	if err != nil {
		return nil, errors.Wrap(err, "could not get channel PDA")
	}

	accounts := []*solana.AccountMeta{
		solana.NewAccountMeta(channelPDA, true, false),                         // Program account derived from channel ID
		solana.NewAccountMeta(cb.signer.participant.SolanaAddress, true, true), // Participant's account
		solana.NewAccountMeta(system.ProgramID, false, false),                  // System program account
	}
	fundIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return fundIx, nil
}
