package adjudicator

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/perun-network/perun-solana-backend/channel/event"
	"github.com/perun-network/perun-solana-backend/client"
	"github.com/perun-network/perun-solana-backend/encoding"
	"perun.network/go-perun/channel"
	pchannel "perun.network/go-perun/channel"
	"perun.network/go-perun/log"
	pkgsync "polycry.pt/poly-go/sync"
)

const (
	DefaultBufferSize                  = 1024
	DefaultSubscriptionPollingInterval = time.Duration(15) * time.Second
)

type PollingSubscription struct {
	chanControl       encoding.Control
	challengeDuration *time.Duration
	cb                *client.ContractBackend
	cid               pchannel.ID
	perunAddr         solana.PublicKey
	assetAddrs        []solana.PublicKey
	events            chan event.PerunEvent
	subErrors         chan error
	err               error
	cancel            context.CancelFunc
	closer            *pkgsync.Closer
	pollInterval      time.Duration
	log               log.Embedding
}

func NewAdjudicatorSubFromChannelID(ctx context.Context, cid pchannel.ID, cb *client.ContractBackend, perunAddr solana.PublicKey, assetAddrs []solana.PublicKey, challengeDuration *time.Duration) *PollingSubscription {
	sub := &PollingSubscription{
		chanControl:       encoding.Control{},
		challengeDuration: challengeDuration,
		cb:                cb,
		cid:               cid,
		perunAddr:         perunAddr,
		assetAddrs:        assetAddrs,
		events:            make(chan event.PerunEvent, DefaultBufferSize),
		subErrors:         make(chan error, 1),
		pollInterval:      DefaultSubscriptionPollingInterval,
		closer:            new(pkgsync.Closer),
		log:               log.MakeEmbedding(log.Default()),
	}

	ctx, sub.cancel = context.WithCancel(ctx)
	go sub.run(ctx)
	return sub
}

func (p *PollingSubscription) run(ctx context.Context) {
	p.log.Log().Info("Listening for channel state changes")

	chanInfo, err := p.cb.GetChannelInfo(ctx, p.perunAddr, p.cid)
	if err != nil {
		p.subErrors <- err
	}

	p.chanControl = chanInfo.Control
	finish := func(err error) {
		p.err = err
		close(p.events)
	}
	var newChanControl encoding.Control
polling:
	for {
		p.log.Log().Debug("AdjudicatorSub is listening for Adjudicator Events")
		select {
		case err := <-p.subErrors:
			finish(err)
			return
		case <-ctx.Done():
			p.log.Log().Debug("Timeout during Adjudicator Subscription")
			finish(nil)
			return
		case <-time.After(p.pollInterval):
			log.Println("Polling for contract events...", p.cid)
			newChanInfo, err := p.cb.GetChannelInfo(ctx, p.perunAddr, p.cid)
			newChanControl = newChanInfo.Control

			if err != nil {
				p.subErrors <- err
			}
			adjEvent, err := DifferencesInControls(p.chanControl, newChanControl)
			if err != nil {
				p.subErrors <- err
			}

			if adjEvent == nil {
				p.chanControl = newChanControl
				p.log.Log().Debug("No events yet, continuing polling...")
				continue polling
			} else {
				p.log.Log().Debug("Contract event detected, evaluating...")
				p.log.Log().Debugf("Found contract event: %v", adjEvent)
				adjEvent.SetID(p.cid)
				p.events <- adjEvent
				etype, _ := adjEvent.GetType()
				if etype == event.EventTypeWithdrawn {
					log.Println("Withdrawn event detected, closing subscription")
					return
				}
			}
		}
	}
}

// DifferencesInControls checks the differences between two channel controls.
func DifferencesInControls(controlCurr, controlNext encoding.Control) (event.PerunEvent, error) {
	if controlCurr.FundedA != controlNext.FundedA {
		if controlCurr.FundedA {
			return nil, errors.New("channel cannot be unfunded A before withdrawal")
		}
	}

	if controlCurr.FundedB != controlNext.FundedB {
		if controlCurr.FundedB {
			return nil, errors.New("channel cannot be unfunded B before withdrawal")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &event.FundEvent{}, nil
		}
	}

	if controlCurr.Closed != controlNext.Closed {
		if controlCurr.Closed {
			return nil, errors.New("channel cannot be reopened after closing")
		}
		if !controlCurr.Closed && controlNext.Closed {
			return &event.CloseEvent{}, nil
		}
		return &event.CloseEvent{}, nil
	}

	if controlCurr.Closed && controlNext.Closed {
		return &event.CloseEvent{}, nil
	}

	if controlCurr.WithdrawnA != controlNext.WithdrawnA {
		if controlCurr.WithdrawnA {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &event.WithdrawnEvent{}, nil
		}
	}

	if controlCurr.WithdrawnB != controlNext.WithdrawnB {
		if controlCurr.WithdrawnB {
			return nil, errors.New("channel cannot be unwithdrawn")
		}
		if controlNext.WithdrawnA && controlNext.WithdrawnB {
			return &event.WithdrawnEvent{}, nil
		}
	}

	if controlCurr.Disputed != controlNext.Disputed {
		if controlCurr.Disputed {
			return nil, errors.New("channel cannot be undisputed")
		}
		return &event.DisputedEvent{}, nil
	}

	return nil, nil
}

func (p *PollingSubscription) Next() channel.AdjudicatorEvent {
	if p.closer.IsClosed() {
		return nil
	}

	if p.getEvents() == nil {
		return nil
	}
	select {
	case ev := <-p.getEvents():
		if ev == nil {
			return nil
		}

		switch e := ev.(type) {
		case *event.DisputedEvent:
			log.Println("DisputedEvent received - build RegisteredEvent")
			dispEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: event.MakeTimeout(*p.challengeDuration),
			}
			adjDispEvent := &pchannel.RegisteredEvent{AdjudicatorEventBase: dispEvent, State: nil, Sigs: nil}
			return adjDispEvent

		case *event.CloseEvent:

			log.Println("CloseEvent received - build ConcludedEvent, ", e.ID())

			conclEvent := pchannel.AdjudicatorEventBase{
				VersionV: e.Version(),
				IDV:      e.ID(),
				TimeoutV: event.MakeTimeout(*p.challengeDuration),
			}
			adjConclEvent := &pchannel.ConcludedEvent{AdjudicatorEventBase: conclEvent}
			return adjConclEvent

		default:
			log.Printf("Received an unknown event type: %v\n", reflect.TypeOf(e))
			return nil
		}

	case <-p.closer.Closed():
		return nil
	}
}

func (p *PollingSubscription) Err() error {
	return p.err
}

func (p *PollingSubscription) Close() error {
	p.closer.Close()
	return nil
}

func (p *PollingSubscription) getEvents() <-chan event.PerunEvent {
	return p.events
}
