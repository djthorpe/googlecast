package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////

func HandleEvent(cast googlecast.Cast, evt googlecast.Event) error {
	switch evt.Type() {
	case googlecast.CAST_EVENT_DEVICE_ADDED:
		if evt.Device() != nil {
			if _, err := cast.Connect(evt.Device(), gopi.RPC_FLAG_INET_V4|gopi.RPC_FLAG_INET_V6, 0); err != nil {
				return err
			}
		}
	}
	event_type := strings.TrimPrefix(fmt.Sprint(evt.Type()), "CAST_EVENT_")
	fmt.Printf("%-20s %s\n", event_type, evt.Device().Name())
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func Main(app *gopi.AppInstance, done chan<- struct{}) error {

	app.Logger.Info("Waiting for CTRL+C")
	app.WaitForSignal()
	done <- gopi.DONE

	// Success
	return nil
}

func Events(app *gopi.AppInstance, start chan<- struct{}, stop <-chan struct{}) error {
	cast := app.ModuleInstance("googlecast").(googlecast.Cast)
	timeout, _ := app.AppFlags.GetDuration("timeout")
	timer := time.NewTimer(timeout)
	start <- gopi.DONE

	// If there is an argument, then this is the service to lookup
	events := cast.Subscribe()
FOR_LOOP:
	for {
		select {
		case <-timer.C:
			// Quit
			app.SendSignal()
		case evt := <-events:
			if evt_, ok := evt.(googlecast.Event); ok {
				if err := HandleEvent(cast, evt_); err != nil {
					app.Logger.Error("Error: %v", err)
				}
			}
		case <-stop:
			timer.Stop()
			break FOR_LOOP
		}
	}
	cast.Unsubscribe(events)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func main() {
	// Create the configuration
	config := gopi.NewAppConfig("googlecast", "discovery")

	// Set timeout flag
	config.AppFlags.FlagDuration("timeout", time.Second*2, "Timeout for discovery")

	// Run the command line tool
	os.Exit(gopi.CommandLineTool2(config, Main, Events))
}
