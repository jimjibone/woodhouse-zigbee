package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jimjibone/log"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name: "Tool for testing mqtt/websocket connections with zigbee2mqtt.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "ws-addr",
				Usage: "websocket server address",
				Value: "localhost:8080",
			},
			&cli.StringFlag{
				Name:  "web-addr",
				Usage: "external web server address",
				Value: "localhost:8080",
			},
			// &cli.BoolFlag{
			// 	Name:  "use-mqtt",
			// 	Usage: "use mqtt connection instead of websockets",
			// },
			// &cli.StringFlag{
			// 	Name:  "mqtt-server",
			// 	Usage: "mqtt server address",
			// 	Value: "mqtt://localhost:1883",
			// },
			// &cli.StringFlag{
			// 	Name:  "mqtt-topic",
			// 	Usage: "mqtt root topic",
			// 	Value: "zigbee2mqtt",
			// },
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
			zigbee := WsConn{
				WebAddr: args.String("web-addr"),
				WsAddr:  args.String("ws-addr"),
			}
			err := zigbee.Run(context.Background())
			if err != nil {
				return fmt.Errorf("failed to run: %s", err)
			}
			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}
