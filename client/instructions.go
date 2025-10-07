package client

import (
	"log"

	"github.com/gagliardetto/solana-go"
	system "github.com/gagliardetto/solana-go/programs/system"
	"github.com/perun-network/perun-solana-backend/channel"
	"github.com/perun-network/perun-solana-backend/encoding"
	"github.com/pkg/errors"
	pchannel "perun.network/go-perun/channel"
	pwallet "perun.network/go-perun/wallet"
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

func (cb *ContractBackend) NewFundInstruction(perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset, funderIdx bool) (solana.Instruction, error) {
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

	for _, asset := range assets {
		solAsset, ok := asset.(*channel.SolanaCrossAsset)
		if ok {
			if !solAsset.Asset.IsSOL {
				actorAta, err := cb.GetAssociatedTokenAccount(cb.signer.participant.SolanaAddress, *solAsset.Asset.Mint)
				if err != nil {
					return nil, errors.Wrap(err, "could not get associated token account for channel")
				}
				channelAta, err := cb.GetAssociatedTokenAccount(channelPDA, *solAsset.Asset.Mint)
				if err != nil {
					return nil, errors.Wrap(err, "could not get associated token account for channel")
				}
				accounts = append(accounts, solana.NewAccountMeta(*solAsset.Asset.Mint, true, false))                       // Mint address of the token
				accounts = append(accounts, solana.NewAccountMeta(actorAta, true, false))                                   // Signer's associated token account for the asset
				accounts = append(accounts, solana.NewAccountMeta(channelAta, true, false))                                 // Channel's associated token account for the asset
				accounts = append(accounts, solana.NewAccountMeta(solana.TokenProgramID, false, false))                     // SPL Token program account
				accounts = append(accounts, solana.NewAccountMeta(solana.SPLAssociatedTokenAccountProgramID, false, false)) // Associated Token program account
			}
		}
	}

	fundIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return fundIx, nil
}

func (cb *ContractBackend) NewAbortInstruction(perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset, creator solana.PublicKey) (solana.Instruction, error) {
	data, err := encoding.MakeAbortInstruction(chanID)
	if err != nil {
		return nil, errors.Wrap(err, "could not create abort instruction")
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
	}

	for _, asset := range assets {
		solAsset, ok := asset.(*channel.SolanaCrossAsset)
		if ok {
			if !solAsset.Asset.IsSOL {
				actorAta, err := cb.GetAssociatedTokenAccount(cb.signer.participant.SolanaAddress, *solAsset.Asset.Mint)
				if err != nil {
					return nil, errors.Wrap(err, "could not get associated token account for actor")
				}
				channelAta, err := cb.GetAssociatedTokenAccount(channelPDA, *solAsset.Asset.Mint)
				if err != nil {
					return nil, errors.Wrap(err, "could not get associated token account for channel")
				}
				accounts = append(accounts, solana.NewAccountMeta(*solAsset.Asset.Mint, true, false))                       // Mint address of the token
				accounts = append(accounts, solana.NewAccountMeta(actorAta, true, false))                                   // Signer's associated token account for the asset
				accounts = append(accounts, solana.NewAccountMeta(channelAta, true, false))                                 // Channel's associated token account for the asset
				accounts = append(accounts, solana.NewAccountMeta(solana.TokenProgramID, false, false))                     // SPL Token program account
				accounts = append(accounts, solana.NewAccountMeta(solana.SPLAssociatedTokenAccountProgramID, false, false)) // Associated Token program account
			}
		}
	}

	accounts = append(accounts, solana.NewAccountMeta(creator, true, false)) // Channel creator's account
	abortIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return abortIx, nil
}

func (cb *ContractBackend) NewCloseInstruction(perunAddr solana.PublicKey, state *pchannel.State, sigs []pwallet.Sig) (solana.Instruction, error) {
	if len(sigs) != 2 {
		return nil, errors.New("need exactly two signatures to close the channel")
	}
	if len(sigs[0]) != 65 || len(sigs[1]) != 65 {
		return nil, errors.New("signatures must be 65 bytes long")
	}
	var sigA [65]byte
	copy(sigA[:], sigs[0][:])
	log.Println("Signature A:", sigA)
	var sigB [65]byte
	copy(sigB[:], sigs[1][:])
	log.Println("Signature B:", sigB)

	data, err := encoding.MakeCloseInstruction(state, sigA, sigB)
	if err != nil {
		return nil, errors.Wrap(err, "could not create close instruction")
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
	}
	closeIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return closeIx, nil
}

func (cb *ContractBackend) NewForceCloseInstruction(perunAddr solana.PublicKey, chanID pchannel.ID) (solana.Instruction, error) {
	data, err := encoding.MakeForceCloseInstruction(chanID)
	if err != nil {
		return nil, errors.Wrap(err, "could not create force close instruction")
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
	}
	forceCloseIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return forceCloseIx, nil
}

func (cb *ContractBackend) NewWithdrawInstruction(perunAddr solana.PublicKey, chanID pchannel.ID, assets []pchannel.Asset, withdrawerIdx bool, oneWithdrawer bool, creator solana.PublicKey) (solana.Instruction, error) {
	data, err := encoding.MakeWithdrawInstruction(chanID, withdrawerIdx, oneWithdrawer)
	if err != nil {
		return nil, errors.Wrap(err, "could not create withdraw instruction")
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
	}

	for _, asset := range assets {
		solAsset, ok := asset.(*channel.SolanaCrossAsset)
		if ok {
			if !solAsset.Asset.IsSOL {
				actorAta, err := cb.GetAssociatedTokenAccount(cb.signer.participant.SolanaAddress, *solAsset.Asset.Mint)
				if err != nil {
					return nil, errors.Wrap(err, "could not get associated token account for channel")
				}
				channelAta, err := cb.GetAssociatedTokenAccount(channelPDA, *solAsset.Asset.Mint)
				if err != nil {
					return nil, errors.Wrap(err, "could not get associated token account for channel")
				}
				accounts = append(accounts, solana.NewAccountMeta(*solAsset.Asset.Mint, true, false))                       // Mint address of the token
				accounts = append(accounts, solana.NewAccountMeta(actorAta, true, false))                                   // Signer's associated token account for the asset
				accounts = append(accounts, solana.NewAccountMeta(channelAta, true, false))                                 // Channel's associated token account for the asset
				accounts = append(accounts, solana.NewAccountMeta(solana.SystemProgramID, false, false))                    // System program account
				accounts = append(accounts, solana.NewAccountMeta(solana.TokenProgramID, false, false))                     // SPL Token program account
				accounts = append(accounts, solana.NewAccountMeta(solana.SPLAssociatedTokenAccountProgramID, false, false)) // Associated Token program account
			} else {
				// If the asset is not a SolanaCrossAsset, we assume it's SOL and add the SystemProgramID
				accounts = append(accounts, solana.NewAccountMeta(solana.SystemProgramID, false, false)) // System program account
			}
		}
	}

	accounts = append(accounts, solana.NewAccountMeta(creator, true, false)) // Channel creator's account
	withdrawIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return withdrawIx, nil
}

func (cb *ContractBackend) NewDisputeInstruction(perunAddr solana.PublicKey, state *pchannel.State, sigs []pwallet.Sig) (solana.Instruction, error) {
	if len(sigs) != 2 {
		return nil, errors.New("need exactly two signatures to dispute the channel")
	}
	var sigA [65]byte
	copy(sigA[:], sigs[0][:])
	var sigB [65]byte
	copy(sigB[:], sigs[1][:])

	data, err := encoding.MakeDisputeInstruction(state, sigA, sigB)
	if err != nil {
		return nil, errors.Wrap(err, "could not create dispute instruction")
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
	}
	closeIx := solana.NewInstruction(
		perunAddr, // Program ID
		accounts,  // Accounts to be passed to the instruction
		data,      // Instruction data
	)
	return closeIx, nil
}
