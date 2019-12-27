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
	devices   map[string]*castdevice
	channels  map[*castchannel]*castdevice

	event.Publisher
	event.Tasks
	sync.Mutex
	sync.WaitGroup
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
	this.devices = make(map[string]*castdevice)
	this.channels = make(map[*castchannel]*castdevice)

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

	// Close channels
	for channel := range this.channels {
		errs.Add(this.Disconnect(channel))
	}

	// Wait for end of channel watching
	this.WaitGroup.Wait()

	// Stop background tasks
	if err := this.Tasks.Close(); err != nil {
		errs.Add(err)
	}

	// Unsubscribe
	this.Publisher.Close()

	// Release resources
	this.channels = nil
	this.devices = nil

	// Return any errors caught
	return errs.ErrorOrSelf()
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *cast) String() string {
	if len(this.devices) > 0 {
		return fmt.Sprintf("<googlecast>{ devices=%v }", this.devices)
	} else {
		return fmt.Sprintf("<googlecast>{ nil }")
	}
}

////////////////////////////////////////////////////////////////////////////////
// INTERFACE IMPLEMENTATION

func (this *cast) Devices() []googlecast.Device {
	this.Lock()
	defer this.Unlock()

	devices := make([]googlecast.Device, 0, len(this.devices))
	for _, device := range this.devices {
		devices = append(devices, device)
	}
	return devices
}

func (this *cast) Connect(device googlecast.Device, flag gopi.RPCFlag, timeout time.Duration) (googlecast.Channel, error) {
	this.log.Debug2("<googlecast.Connect>{ device=%v flag=%v timeout=%v }", device, flag, timeout)

	if device_, ok := device.(*castdevice); device_ == nil || ok == false {
		return nil, gopi.ErrBadParameter
	} else if ip, err := device_.addr(flag); err != nil {
		return nil, err
	} else if channel, err := gopi.Open(Channel{
		Addr:    ip.String(),
		Port:    uint16(device_.Port()),
		Timeout: timeout,
	}, this.log); err != nil {
		return nil, fmt.Errorf("Connect: %w", err)
	} else if channel_, ok := channel.(*castchannel); ok == false {
		return nil, gopi.ErrAppError
	} else if err := this.addChannel(device_, channel_); err != nil {
		return nil, err
	} else {
		// Watch channel for messages
		go this.WatchChannelEvents(device, channel_.Subscribe())

		// Return success
		return channel_, nil
	}
}

func (this *cast) Disconnect(channel googlecast.Channel) error {
	this.log.Debug2("<googlecast.Disconnect>{ channel=%v }", channel)

	if channel_, ok := channel.(*castchannel); channel_ == nil || ok == false {
		return gopi.ErrBadParameter
	} else if err := this.deleteChannel(channel_); err != nil {
		return err
	} else {
		return channel_.Close()
	}
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

func (this *cast) WatchChannelEvents(device googlecast.Device, evts <-chan gopi.Event) {
	this.WaitGroup.Add(1)
FOR_LOOP:
	for {
		select {
		case evt := <-evts:
			if evt == nil {
				continue
			} else if evt_, ok := evt.(*castevent); ok == false {
				continue
			} else if evt_.Type() == googlecast.CAST_EVENT_CHANNEL_DISCONNECT {
				break FOR_LOOP
			} else {
				// Append device
				evt_.device_ = device
				evt_.source_ = this
				this.Emit(evt_)
			}
		}
	}
	this.WaitGroup.Done()
}

////////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func (this *cast) WatchEvent(evt gopi.RPCEvent) error {
	if service := evt.ServiceRecord(); service == nil || service.Service() != SERVICE_TYPE_GOOGLECAST {
		return nil
	} else if device := NewDevice(service); device.Id() == "" {
		return nil
	} else if evt.Type() == gopi.RPC_EVENT_SERVICE_EXPIRED {
		this.Emit(&castevent{googlecast.CAST_EVENT_DEVICE_DELETED, this, device, nil, 0})
		this.deleteDevice(device)
	} else if evt.Type() == gopi.RPC_EVENT_SERVICE_ADDED || evt.Type() == gopi.RPC_EVENT_SERVICE_UPDATED {
		if device_, exists := this.devices[device.Id()]; device_ == nil || exists == false {
			this.addDevice(device)
			this.Emit(&castevent{googlecast.CAST_EVENT_DEVICE_ADDED, this, device, nil, 0})
		} else if device.Equals(device_) == false {
			this.addDevice(device)
			this.Emit(&castevent{googlecast.CAST_EVENT_DEVICE_UPDATED, this, device, nil, 0})
		}
	}
	// Success
	return nil
}

func NewDevice(srv gopi.RPCServiceRecord) *castdevice {
	return &castdevice{RPCServiceRecord: srv}
}

func (this *cast) addDevice(device *castdevice) {
	this.Lock()
	defer this.Unlock()
	this.devices[device.Id()] = device
}

func (this *cast) deleteDevice(device *castdevice) {
	this.Lock()
	defer this.Unlock()
	delete(this.devices, device.Id())
}

func (this *cast) addChannel(device *castdevice, channel *castchannel) error {
	this.Lock()
	defer this.Unlock()
	if _, exists := this.devices[device.Id()]; exists == false {
		return gopi.ErrNotFound
	} else {
		this.channels[channel] = device
		return nil
	}
}

func (this *cast) deleteChannel(channel *castchannel) error {
	this.Lock()
	defer this.Unlock()
	if _, exists := this.channels[channel]; exists {
		delete(this.channels, channel)
		return nil
	} else {
		return gopi.ErrNotFound
	}
}
