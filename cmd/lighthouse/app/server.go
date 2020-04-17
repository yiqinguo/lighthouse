package app

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/mYmNeo/version/verflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig/v1alpha1"
	"github.com/mYmNeo/lighthouse/pkg/hook"
	"github.com/mYmNeo/lighthouse/pkg/lighthouse/scheme"
	"github.com/mYmNeo/lighthouse/pkg/util"
)

type Options struct {
	ConfigFile string
	config     *componentconfig.HookConfiguration
}

func NewLighthouseCommand() *cobra.Command {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use: "lighthouse",
		Long: "The lighthouse runs on each node. This is a preHook framework to modify request body for any matched rules in the" +
			" configuration. It is an enhancement for kubelet to run a container",
		Run: func(cmd *cobra.Command, args []string) {
			verflag.PrintAndExitIfRequested()
			util.PrintFlags(cmd.Flags())

			if err := opts.Complete(); err != nil {
				klog.Fatalf("failed complete: %v", err)
			}

			if err := opts.Run(); err != nil {
				klog.Exit(err)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func NewOptions() *Options {
	return &Options{
		config: &componentconfig.HookConfiguration{},
	}
}

func (o *Options) Run() error {
	hookServer := hook.NewHookManager()

	if err := hookServer.InitFromConfig(o.config); err != nil {
		return err
	}

	return hookServer.Run(context.Background().Done())
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigFile, "config", o.ConfigFile, "The path to the configuration file")
}

func (o *Options) Complete() error {
	if len(o.ConfigFile) > 0 {
		cfgData, err := ioutil.ReadFile(o.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to read hook configuration file %q, %v", o.ConfigFile, err)
		}

		versioned := &v1alpha1.HookConfiguration{}
		v1alpha1.SetObjectDefaults_HookConfiguration(versioned)
		decoder := scheme.Codecs.UniversalDecoder(v1alpha1.SchemeGroupVersion)
		if err := runtime.DecodeInto(decoder, cfgData, versioned); err != nil {
			return fmt.Errorf("failed to decode hook configuration file %q, %v", o.ConfigFile, err)
		}

		if err := scheme.Scheme.Convert(versioned, o.config, nil); err != nil {
			return fmt.Errorf("failed to convert versioned hook configurtion to internal version, %v", err)
		}
	}
	return nil
}
