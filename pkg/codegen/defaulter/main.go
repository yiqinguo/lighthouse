package main

import (
	"flag"
	"os"
	"path/filepath"

	generatorargs "k8s.io/code-generator/cmd/defaulter-gen/args"
	"k8s.io/gengo/examples/defaulter-gen/generators"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()

	baseDir := os.Getenv("BASE_DIR")
	packageName := os.Getenv("PACKAGE")
	outputDir := os.Getenv("OUTPUT_PATH")
	genericArgs, customArgs := generatorargs.NewDefaults()
	genericArgs.GoHeaderFilePath = filepath.Join(baseDir, "hack/boilerplate.go.txt")

	genericArgs.InputDirs = []string{
		filepath.Join(packageName, "pkg/apis/componentconfig/v1alpha1"),
	}
	genericArgs.OutputBase = outputDir

	customArgs.ExtraPeerDirs = []string{
		filepath.Join(packageName, "pkg/apis/componentconfig/v1alpha1"),
	}
	genericArgs.OutputFileBaseName = "zz_generated.defaults"

	if err := genericArgs.Execute(
		generators.NameSystems(),
		generators.DefaultNameSystem(),
		generators.Packages,
	); err != nil {
		klog.Fatalf("Error: %v", err)
	}
	klog.Infof("Defaulter completed successfully.")
}
