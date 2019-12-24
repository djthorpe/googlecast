package main

import (
	"os"

	// Frameworks
	gopi "github.com/djthorpe/gopi"
	rpc "github.com/djthorpe/gopi-rpc"
)

////////////////////////////////////////////////////////////////////////////////

func main() {
	// Create the configuration
	config := gopi.NewAppConfig("googlecast:service")

	// Run the command line tool
	os.Exit(rpc.Server(config))
}
