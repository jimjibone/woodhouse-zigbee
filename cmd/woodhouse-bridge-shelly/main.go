package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	api "github.com/jimjibone/woodhouse-4/api/go"
	"github.com/jimjibone/woodhouse-4/apitools"
	"github.com/jimjibone/woodhouse-4/log"
	"github.com/jimjibone/woodhouse-4/wh"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                 "woodhouse-bridge-shelly",
		Usage:                "Bridges Shelly devices into Woodhouse.",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Usage: "woodhouse-core server address (disables automatic discovery)",
			},
			&cli.StringFlag{
				Name:  "id",
				Usage: "ID used by this bridge",
				Value: "shelly",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Name used by this bridge",
				Value: "Shelly",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Usage:   "Enable debug logging",
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
				Description: "Shelly device support.",
				BootTime:    apitools.TimeToTimestamp(time.Now()),
			})

			// Only do regular device updates when connected to woodhouse.
			doUpdates := make(chan bool, 1)
			bridge.OnConnected = func() { doUpdates <- true }
			bridge.OnDisconnected = func() { doUpdates <- false }

			// Collect errors from goroutines.
			var wg sync.WaitGroup
			errs := make(chan error, 1)

			// Run the shelly stuff.
			wg.Add(1)
			go func() {
				err := shellyStuff(ctx, bridge, doUpdates)
				if err != nil {
					errs <- err
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
