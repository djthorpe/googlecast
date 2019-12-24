/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2019
  All Rights Reserved
  Documentation http://djthorpe.github.io/gopi/
  For Licensing and Usage information, please see LICENSE.md
*/

package googlecast

import (
	"context"
	"fmt"
	"sync"
	"time"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
	errors "github.com/djthorpe/gopi/util/errors"
	event "github.com/djthorpe/gopi/util/event"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type Cast struct {
	Discovery gopi.RPCServiceDiscovery
}

type cast struct {
	log       gopi.Logger
	discovery gopi.RPCServiceDiscovery
	devices   map[string]*device

	event.Publisher
	event.Tasks
	sync.Mutex
}

////////////////////////////////////////////////////////////////////////////////
// COMSTANTS

const (
	SERVICE_TYPE_GOOGLECAST = "_googlecast._tcp"
	DELTA_LOOKUP_TIME       = 60 * time.Second
)

////////////////////////////////////////////////////////////////////////////////
// OPEN AND CLOSE

func (config Cast) Open(logger gopi.Logger) (gopi.Driver, error) {
	logger.Debug("<googlecast.Open>{ discovery=%v }", config.Discovery)

	this := new(cast)
	this.log = logger
	this.discovery = config.Discovery
	this.devices = make(map[string]*device)

	if this.discovery == nil {
		return nil, gopi.ErrBadParameter
	}

	// Run background tasks
	this.Tasks.Start(this.Watch, this.Lookup)

	// Success
	return this, nil
}

func (this *cast) Close() error {
	this.log.Debug("<googlecast.Close>{ }")

	errs := errors.CompoundError{}

	// Stop background tasks
	if err := this.Tasks.Close(); err != nil {
		errs.Add(err)
	}

	// Unsubscribe
	this.Publisher.Close()

	// Release resources
	this.devices = nil

	// Return any errors caught
	return errs.ErrorOrSelf()
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *cast) String() string {
	return fmt.Sprintf("<googlecast>{ }")
}

////////////////////////////////////////////////////////////////////////////////
// INTERFACE IMPLEMENTATION

func (this *cast) Devices() []googlecast.Device {
	devices := make([]googlecast.Device, 0, len(this.devices))
	for _, device := range this.devices {
		devices = append(devices, device)
	}
	return devices
}

////////////////////////////////////////////////////////////////////////////////
// BACKGROUND TASKS

func (this *cast) Lookup(start chan<- event.Signal, stop <-chan event.Signal) error {
	this.log.Debug("<googlecast.Lookup> Started")
	start <- gopi.DONE

	// Periodically lookup Googlecast devices
	timer := time.NewTimer(100 * time.Millisecond)
FOR_LOOP:
	for {
		select {
		case <-timer.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			this.discovery.Lookup(ctx, SERVICE_TYPE_GOOGLECAST)
			cancel()
			timer.Reset(DELTA_LOOKUP_TIME)
		case <-stop:
			break FOR_LOOP
		}
	}
	this.log.Debug("<googlecast.Lookup> Stopped")
	return nil
}

func (this *cast) Watch(start chan<- event.Signal, stop <-chan event.Signal) error {
	this.log.Debug("<googlecast.Watch> Started")
	start <- gopi.DONE

	events := this.discovery.Subscribe()
FOR_LOOP:
	for {
		select {
		case evt := <-events:
			if evt_, ok := evt.(gopi.RPCEvent); ok {
				this.WatchEvent(evt_)
			}
		case <-stop:
			break FOR_LOOP
		}
	}
	this.discovery.Unsubscribe(events)
	this.log.Debug("<googlecast.Watch> Stopped")
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func (this *cast) WatchEvent(evt gopi.RPCEvent) error {
	this.Lock()
	defer this.Unlock()
	if service := evt.ServiceRecord(); service == nil || service.Service() != SERVICE_TYPE_GOOGLECAST {
		return nil
	} else if device := NewDevice(service); device.Id() == "" {
		return nil
	} else if evt.Type() == gopi.RPC_EVENT_SERVICE_EXPIRED {
		this.Emit(&castevent{ googlecast.CAST_EVENT_DEVICE_DELETED, this, device })
		delete(this.devices, device.Id())
	} else if evt.Type() == gopi.RPC_EVENT_SERVICE_ADDED || evt.Type() == gopi.RPC_EVENT_SERVICE_UPDATED {
		if device_, exists := this.devices[device.Id()]; device_ == nil || exists == false {
			this.devices[device.Id()] = device
			this.Emit(&castevent{ googlecast.CAST_EVENT_DEVICE_ADDED, this, device })
		} else if device.Equals(device_) == false {
			this.devices[device.Id()] = device
			this.Emit(&castevent{ googlecast.CAST_EVENT_DEVICE_UPDATED, this, device })
		}
	}
	// Success
	return nil
}

func NewDevice(srv gopi.RPCServiceRecord) *device {
	return &device{RPCServiceRecord: srv}
}
