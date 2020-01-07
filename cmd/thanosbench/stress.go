package main

import (
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerStress(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("stress", "Stress tests a remote StoreAPI.")
	workers := cmd.Flag("workers.num", "Number of go routines for stress testing.").Required().Int()
	target := cmd.Arg("target", "IP:PORT pair of the target to stress.").IP()

	// TODO(GiedriusS): send other requests like Info() as well.
	m["stress"] = func(g *run.Group, logger log.Logger) error {
		g.Add(func() error {
			var _ = workers
			var _ = target
			return nil
		}, func(error) {

		})
		return nil
	}
}
