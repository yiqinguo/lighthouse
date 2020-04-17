package main

import (
	"os"
	"path/filepath"
	"strings"

	"k8s.io/klog"
)

func main() {
	baseDir := os.Getenv("BASE_DIR")

	for _, dir := range []string{"pkg/apis"} {
		walkErr := filepath.Walk(filepath.Join(baseDir, dir), func(path string,
			info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.Contains(path, "vendor") {
				return filepath.SkipDir
			}

			if strings.HasPrefix(info.Name(), "zz_generated") {
				klog.Infof("Removing %s", path)
				if err := os.Remove(path); err != nil {
					return err
				}
			}

			return nil
		})

		if walkErr != nil {
			os.Exit(1)
		}
	}
}
