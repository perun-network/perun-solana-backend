package event

import (
	"errors"

	"github.com/perun-network/perun-solana-backend/encoding"
	pchannel "perun.network/go-perun/channel"
)

type (
	Version   = uint64
	EventType int //nolint:golint
)

const (
	EventTypeOpen          EventType = iota
	EventTypeFundChannel             // participant/s funding channel
	EventTypeFundedChannel           // participants have funded channel
	EventTypeClosed                  // channel closed -> withdrawing enabled
	EventTypeWithdrawing             // participant/s withdrawing
	EventTypeWithdrawn               // participants have withdrawn
	EventTypeForceClose              // participant has force closed the channel
	EventTypeDisputed                // participant has disputed the channel
	EventTypeError                   // inconsistent event
)

var (
	ErrNoWithdrawEvent = errors.New("withdraw event not found")
)

const AssertPerunSymbol = "perun"

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

// SolanaEvent is a struct that represents a Solana event.
type SolanaEvent struct {
	Type         EventType
	ChannelState encoding.Channel
}

// GetType returns the type of the Solana event.
func (e *SolanaEvent) GetType() EventType {
	return e.Type
}

// GetChannel returns the channel of the Solana event.
func (e *SolanaEvent) GetChannel() encoding.Channel {
	return e.ChannelState
}

// GetType returns the type of the OpenEvent.
func (e *OpenEvent) GetType() (EventType, error) {
	return EventTypeOpen, nil
}

// ID returns the ID of the OpenEvent.
func (e *OpenEvent) ID() pchannel.ID {
	return e.idv
}

// Version returns the version of the OpenEvent.
func (e *OpenEvent) Version() Version {
	return e.versionV
}

// Timeout returns the timeout of the OpenEvent.
func (e *OpenEvent) Timeout() pchannel.Timeout {
	return e.timeout
}

// SetID sets the ID of the OpenEvent.
func (e *OpenEvent) SetID(id pchannel.ID) {
	e.idv = id
}

// GetChannel returns the channel of the WithdrawnEvent.
func (e *WithdrawnEvent) GetChannel() encoding.Channel {
	return e.channel
}

// GetType returns the type of the WithdrawnEvent.
func (e *WithdrawnEvent) GetType() (EventType, error) {
	withdrawnA := e.channel.Control.WithdrawnA
	withdrawnB := e.channel.Control.WithdrawnB

	if withdrawnA && withdrawnB {
		return EventTypeWithdrawn, nil
	} else if withdrawnA != withdrawnB {
		return EventTypeWithdrawing, nil
	}
	return EventTypeError, errors.New("withdraw event has no consistent type: not withdrawn")
}

// ID returns the ID of the WithdrawnEvent.
func (e *WithdrawnEvent) ID() pchannel.ID {
	return e.idv
}

// Version returns the version of the WithdrawnEvent.
func (e *WithdrawnEvent) Version() Version {
	return e.versionV
}

// Timeout returns the timeout of the WithdrawnEvent.
func (e *WithdrawnEvent) Timeout() pchannel.Timeout {
	return e.timeout
}

// SetID sets the ID of the WithdrawnEvent.
func (e *WithdrawnEvent) SetID(id pchannel.ID) {
	e.idv = id
}

// GetChannel returns the channel of the CloseEvent.
func (e *CloseEvent) GetChannel() encoding.Channel {
	return e.channel
}

// GetType returns the type of the CloseEvent.
func (e *CloseEvent) GetType() (EventType, error) {
	return EventTypeClosed, nil
}

// ID returns the ID of the CloseEvent.
func (e *CloseEvent) ID() pchannel.ID {
	return e.idv
}

// Version returns the version of the CloseEvent.
func (e *CloseEvent) Version() Version {
	return e.versionV
}

// Timeout returns the timeout of the CloseEvent.
func (e *CloseEvent) Timeout() pchannel.Timeout {
	return e.timeout
}

// SetID sets the ID of the CloseEvent.
func (e *CloseEvent) SetID(id pchannel.ID) {
	e.idv = id
}

// GetChannel returns the channel of the FundEvent.
func (e *FundEvent) GetChannel() encoding.Channel {
	return e.channel
}

// GetType returns the type of the FundEvent.
func (e *FundEvent) GetType() (EventType, error) {
	fundedA := e.channel.Control.FundedA
	fundedB := e.channel.Control.FundedB

	if fundedA && fundedB {
		return EventTypeFundedChannel, nil
	} else if fundedA != fundedB {
		return EventTypeFundChannel, nil
	}
	return EventTypeError, errors.New("funding event has no consistent type: not funded")
}

// ID returns the ID of the FundEvent.
func (e *FundEvent) ID() pchannel.ID {
	return e.idv
}

// Version returns the version of the FundEvent.
func (e *FundEvent) Version() Version {
	return e.versionV
}

// Timeout returns the timeout of the FundEvent.
func (e *FundEvent) Timeout() pchannel.Timeout {
	return e.timeout
}

// SetID sets the ID of the FundEvent.
func (e *FundEvent) SetID(id pchannel.ID) {
	e.idv = id
}

// ID returns the id of the DisputedEvent.
func (e *DisputedEvent) ID() pchannel.ID {
	return e.idv
}

// GetChannel returns the channel of the DisputedEvent.
func (e *DisputedEvent) GetChannel() encoding.Channel {
	return e.channel
}

// Version returns the version of the DisputedEvent.
func (e *DisputedEvent) Version() Version {
	return e.versionV
}

// Timeout returns the timeout of the DisputedEvent.
func (e *DisputedEvent) Timeout() pchannel.Timeout {
	return e.timeout
}

// GetType returns the type of the DisputedEvent.
func (e *DisputedEvent) GetType() (EventType, error) {
	return EventTypeDisputed, nil
}

// SetID sets the ID of the DisputedEvent.
func (e *DisputedEvent) SetID(id pchannel.ID) {
	e.idv = id
}
