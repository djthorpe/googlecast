package main

import (
	"fmt"
	"os"
	"time"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
	rpc "github.com/djthorpe/gopi-rpc"
)

////////////////////////////////////////////////////////////////////////////////

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
		fmt.Println(devices)
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
