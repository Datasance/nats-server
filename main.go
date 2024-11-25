package main

import (
	"errors"
	"log"
	"os"

	sdk "github.com/datasance/iofog-go-sdk/v3/pkg/microservices"
	nats "github.com/datasance/nats-server/internal/nats"
)

var (
	natsServer *nats.Server
)

func init() {
	natsServer = new(nats.Server)
	natsServer.Config = new(nats.Config)
}

func main() {
	ioFogClient, clientError := sdk.NewDefaultIoFogClient()
	if clientError != nil {
		log.Fatalln(clientError.Error())
	}

	// Update initial configuration
	if err := updateConfig(ioFogClient, natsServer.Config); err != nil {
		log.Fatalln(err.Error())
	}

	// Establish WebSocket connection for configuration updates
	confChannel := ioFogClient.EstablishControlWsConnection(0)

	// Channel for server exit handling
	exitChannel := make(chan error)

	// Start NATS server in a goroutine
	go natsServer.StartServer(natsServer.Config, exitChannel)

	// Main loop to handle configuration updates
	for {
		select {
		case <-exitChannel:
			os.Exit(0)
		case <-confChannel:
			newConfig := new(nats.Config)
			if err := updateConfig(ioFogClient, newConfig); err != nil {
				log.Fatal(err)
			} else {
				natsServer.UpdateServer(newConfig)
			}
		}
	}
}

func updateConfig(ioFogClient *sdk.IoFogClient, config interface{}) error {
	attemptLimit := 5
	var err error

	for err = ioFogClient.GetConfigIntoStruct(config); err != nil && attemptLimit > 0; attemptLimit-- {
		return err
	}

	if attemptLimit == 0 {
		return errors.New("Update config failed")
	}

	return nil
}
