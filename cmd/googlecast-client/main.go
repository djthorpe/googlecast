package main

import (
	"fmt"
	"os"
	"time"

	// Frameworks
	gopi "github.com/djthorpe/gopi"
	rpc "github.com/djthorpe/gopi-rpc"
)

////////////////////////////////////////////////////////////////////////////////

func Main(app *gopi.AppInstance, services []gopi.RPCServiceRecord, done chan<- struct{}) error {
	if len(services) == 0 {
		return fmt.Errorf("Service not found")
	} else if client, err := app.ClientPool.NewClientEx("gopi.GoogleCast", services, 0); err != nil {
		return err
	} else {
		fmt.Println(client)
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
