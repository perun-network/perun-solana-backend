package event

import (
	"github.com/perun-network/perun-solana-backend/encoding"
	pchannel "perun.network/go-perun/channel"
)

type (
	Version = uint64
)

type (
	// PerunEvent is an interface for all events that can be emitted by the Perun Solana Program.
	PerunEvent interface {
		ID() pchannel.ID
		GetChannel() encoding.Channel
		Version() Version
		GetType() (EventType, error)
		Timeout() pchannel.Timeout
		SetID(id pchannel.ID)
	}

	// OpenEvent is emitted when a channel is opened.
	OpenEvent struct {
		channel  encoding.Channel
		idv      pchannel.ID
		versionV Version
		timeout  pchannel.Timeout
	}

	// FundEvent is emitted when a channel is funded.
	FundEvent struct {
		channel  encoding.Channel
		idv      pchannel.ID
		versionV Version
		timeout  pchannel.Timeout
	}

	// CloseEvent is emitted when a channel is closed.
	CloseEvent struct {
		channel  encoding.Channel
		idv      pchannel.ID
		versionV Version
		timeout  pchannel.Timeout
	}

	// WithdrawnEvent is emitted when a channel is withdrawn.
	WithdrawnEvent struct {
		channel  encoding.Channel
		idv      pchannel.ID
		versionV Version
		timeout  pchannel.Timeout
	}

	// DisputedEvent is emitted when a channel is disputed.
	DisputedEvent struct {
		channel  encoding.Channel
		idv      pchannel.ID
		versionV Version
		timeout  pchannel.Timeout
	}
)
