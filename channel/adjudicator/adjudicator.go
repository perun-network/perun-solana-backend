package adjudicator

import (
	"context"

	"perun.network/go-perun/channel"
)

type Adjudicator struct{}

func NewAdjudicator() *Adjudicator {
	return &Adjudicator{}
}

func (a Adjudicator) Register(ctx context.Context, req channel.AdjudicatorReq, subChannels []channel.SignedState) error {
	panic("TODO: implement Register in adjudicator")
}

func (a Adjudicator) Withdraw(ctx context.Context, req channel.AdjudicatorReq, stateMap channel.StateMap) error {
	panic("TODO: implement Withdraw in adjudicator")
}

func (a Adjudicator) Progress(ctx context.Context, req channel.ProgressReq) error {
	return nil // Only used in AppChannel
}

func (a Adjudicator) Subscribe(ctx context.Context, id channel.ID) (channel.AdjudicatorSubscription, error) {
	return NewAdjudicatorSubFromChannelID(ctx, id), nil
}
