package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/wh"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse-bridge-zigbee",
		Usage:                "Bridges zigbee devices from zigbee2mqtt into Woodhouse.",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Usage: "woodhouse-core server address (disables automatic discovery)",
			},
			&cli.StringFlag{
				Name:  "id",
				Usage: "ID used by this bridge",
				Value: "zigbee",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Name used by this bridge",
				Value: "Zigbee",
			},
			&cli.StringFlag{
				Name:  "ws-addr",
				Usage: "websocket server address",
				Value: "localhost:8080",
			},
			&cli.BoolFlag{
				Name:  "use-mqtt",
				Usage: "use mqtt connection instead of websockets",
			},
			&cli.StringFlag{
				Name:  "mqtt-server",
				Usage: "mqtt server address",
				Value: "mqtt://localhost:1883",
			},
			&cli.StringFlag{
				Name:  "mqtt-topic",
				Usage: "mqtt root topic",
				Value: "zigbee2mqtt",
			},
		},
		Action: func(args *cli.Context) error {
			// Create the main app context.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Cancel the main app context if an interrupt is received.
			var sig = make(chan os.Signal, 2)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
			go func() {
				select {
				case <-ctx.Done():
				case <-sig:
					cancel()
				}
			}()

			// Create the bridge.
			bridge := wh.NewBridge(&api.BridgeInfo{
				BridgeId:    args.String("id"),
				Name:        args.String("name"),
				Description: "Zigbee device support via zigbee2mqtt.",
				BootTime:    apitools.TimeToTimestamp(time.Now()),
			})

			// Collect errors from goroutines.
			var wg sync.WaitGroup
			errs := make(chan error, 1)

			// Run the zigbee stuff.
			wg.Add(1)
			go func() {
				if args.Bool("use-mqtt") {
					// Use MQTT for zigbee network data and requests.
					// Note: The websocket connection has a more reliable API
					// formatting and reports the state of all devices
					// regardless of configuration. MQTT data on the other hand
					// can vary depending on user preference.
					zigbee := ZigbeeMQTT{
						MqttAddr:  args.String("mqtt-server"),
						RootTopic: args.String("mqtt-topic"),
					}
					err := zigbee.Run(ctx, bridge)
					if err != nil {
						errs <- err
					}
				} else {
					// Use websockets for zigbee network data and requests.
					zigbee := ZigbeeWebsockets{
						Addr: args.String("ws-addr"),
					}
					err := zigbee.Run(ctx, bridge)
					if err != nil {
						errs <- err
					}
				}
				wg.Done()
			}()

			// Run the bridge stuff.
			wg.Add(1)
			go func() {
				connector := wh.NewConnector(bridge.Run)
				err := connector.Run(ctx)
				if err != nil {
					errs <- err
				}
				wg.Done()
			}()

			// Wait for program exit...
			select {
			case <-ctx.Done():
			case err := <-errs:
				return err
			}
			cancel()
			wg.Wait()
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
