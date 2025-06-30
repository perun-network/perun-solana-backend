package adjudicator

import (
	"context"
	"time"

	"perun.network/go-perun/channel"
)

const (
	DefaultBufferSize                  = 3
	DefaultSubscriptionPollingInterval = time.Duration(4) * time.Second
)

type PollingSubscription struct{} //TODO

func NewAdjudicatorSubFromChannelID(ctx context.Context, id channel.ID) *PollingSubscription {
	return &PollingSubscription{}
}

func (p *PollingSubscription) Next() channel.AdjudicatorEvent {
	//TODO: implement Next in adjudicator subscription
	return nil
}

func (p *PollingSubscription) Err() error {
	return nil //TODO: implement Err in adjudicator subscription
}

func (p *PollingSubscription) Close() error {
	return nil //TODO: implement Close in adjudicator subscription
}
