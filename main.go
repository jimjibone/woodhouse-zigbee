package main

import (
	"context"
	"os"
	"sync"

	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/shared/stores"
	wh "github.com/jimjibone/woodhouse-4/wh/v1"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "woodhouse-zigbee",
		Usage: "Woodhouse client for zigbee devices (via zigbee2mqtt).",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "store",
				Usage:    "path to config storage location",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "woodhouse-core server address",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "id",
				Usage: "ID used by this bridge",
				Value: "zigbee",
			},
			&cli.StringFlag{
				Name:     "web-addr",
				Usage:    "external web server address",
				Required: true,
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
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Usage:   "enable debug logging",
			},
		},
		Before: func(args *cli.Context) error {
			// Setup logging.
			if args.Bool("debug") {
				log.SetOptions(log.WithMinLevel(log.DebugLevel))
			} else {
				log.SetOptions(log.WithMinLevel(log.InfoLevel))
			}
			return nil
		},
		Action: func(args *cli.Context) error {
			// Create the store.
			store := stores.NewFSStore(args.String("store"))

			// Create the client.
			client := wh.NewClient(store, args.String("addr"), wh.WithClientID(args.String("id")))

			// Start the zigbee goroutine.
			wg := &sync.WaitGroup{}
			defer wg.Wait()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wg.Add(1)
			go func() {
				if args.Bool("use-mqtt") {
					// Use MQTT for zigbee network data and requests.
					// Note: The websocket connection has a more reliable API
					// formatting and reports the state of all devices
					// regardless of configuration. MQTT data on the other hand
					// can vary depending on user preference.
					zigbee := ZigbeeMQTT{
						WebAddr:   args.String("web-addr"),
						MqttAddr:  args.String("mqtt-server"),
						RootTopic: args.String("mqtt-topic"),
					}
					err := zigbee.Run(ctx, client)
					if err != nil {
						log.Errorf("failed to run: %s", err)
					}
				} else {
					// Use websockets for zigbee network data and requests.
					zigbee := ZigbeeWebsockets{
						FS:      store,
						WebAddr: args.String("web-addr"),
						WsAddr:  args.String("ws-addr"),
					}
					err := zigbee.Run(ctx, client)
					if err != nil {
						log.Errorf("failed to run: %s", err)
					}
				}
				wg.Done()
			}()

			// Run the client.
			err := client.Run()
			if err != nil {
				return err
			}

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}
