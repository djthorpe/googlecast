package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
	rpc "github.com/djthorpe/gopi-rpc"
	tablewriter "github.com/olekukonko/tablewriter"
)

////////////////////////////////////////////////////////////////////////////////

func PrintDevices(writer io.Writer, devices []googlecast.Device) {
	table := tablewriter.NewWriter(writer)
	for _, device := range devices {
		table.Append([]string{
			device.Id(),
			device.Name(),
			device.Model(),
			device.Service(),
			fmt.Sprint(device.State()),
		})
	}
	table.Render()
}

func WatchEvents(ctx context.Context, client googlecast.Client) error {
	// Receive error from StreamEvents in the background
	errs := make(chan error)
	go func() {
		errs <- client.StreamEvents(ctx)
	}()
	// subscribe to events
	evts := client.Subscribe()
	for {
		select {
		case evt_ := <-evts:
			if evt, ok := evt_.(googlecast.Event); evt != nil && ok && evt.Type() != googlecast.CAST_EVENT_NONE {
				fmt.Println("EVENT:", evt)
			}
		case err := <-errs:
			close(errs)
			client.Unsubscribe(evts)
			return err
		}
	}
	// Success
	return nil
}

func Main(app *gopi.AppInstance, services []gopi.RPCServiceRecord, done chan<- struct{}) error {
	if len(services) == 0 {
		return fmt.Errorf("Service not found")
	} else if client_, err := app.ClientPool.NewClientEx("gopi.GoogleCast", services, 0); err != nil {
		return err
	} else if client := client_.(googlecast.Client); client == nil {
		return gopi.ErrAppError
	} else if err := client.Ping(); err != nil {
		return err
	} else if devices, err := client.Devices(); err != nil {
		return err
	} else {
		PrintDevices(os.Stdout, devices)
		// Watch in background, wait for signal
		ctx, cancel := context.WithCancel(context.Background())
		go WatchEvents(ctx, client)
		app.WaitForSignal()
		cancel()
	}

	// Success
	done <- gopi.DONE
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func main() {
	// Create the configuration
	config := gopi.NewAppConfig("googlecast:client")

	// Run the command line tool
	os.Exit(rpc.Client(config, time.Second, Main))
}
