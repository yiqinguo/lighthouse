package main

import (
	goflag "flag"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"github.com/mYmNeo/lighthouse/cmd/lighthouse/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	command := app.NewLighthouseCommand()

	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
