package iracing

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type Suspension struct {
	LFShockDef float32
	LFShockVel float32

	RFShockDef float32
	RFShockVel float32

	LRShockDef float32
	LRShockVel float32

	RRShockDef float32
	RRShockVel float32
}

type Output struct {
	out chan *Suspension
}

// for now this will receive variables and output them somewhere
func (o *Output) OutputChannel() chan *Suspension {
	if o.out == nil {
		// Connect to a server
		nc, _ := nats.Connect(nats.DefaultURL)
		o.out = make(chan *Suspension, 5)
		go func() {
			for {
				s := <-o.out
				msg, err := json.Marshal(s)
				if err != nil {
					// noop
				}
				nc.Publish("Suspension", msg)
			}
		}()
	}
	return o.out
}
