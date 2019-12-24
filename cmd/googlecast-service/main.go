package main

import (
	"fmt"
	"os"

	// Frameworks
	"github.com/djthorpe/googlecast"
	"github.com/djthorpe/gopi"
)

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
	start <- gopi.DONE

	// If there is an argument, then this is the service to lookup
	events := cast.Subscribe()
FOR_LOOP:
	for {
		select {
		case evt := <-events:
			if evt_, ok := evt.(googlecast.Event); ok {
				fmt.Println(evt_)
			}
		case <-stop:
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

	// Run the command line tool
	os.Exit(gopi.CommandLineTool2(config, Main, Events))
}
