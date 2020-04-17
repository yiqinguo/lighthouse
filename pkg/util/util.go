package util

import (
	"fmt"
	"net/http"
	"strings"
	"syscall"

	"github.com/docker/go-connections/sockets"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

const (
	UnixProto = "unix"
)

func BuildClientOrDie(endpoint string) *http.Client {
	proto, addr, err := GetProtoAndAddress(endpoint)
	if err != nil {
		klog.Fatalf("can't parse endpoint %s, %v", endpoint, err)
	}

	if proto != UnixProto {
		klog.Fatalf("only support unix socket")
	}

	tr := new(http.Transport)
	sockets.ConfigureTransport(tr, proto, addr)
	return &http.Client{
		Transport: tr,
	}
}

func GetProtoAndAddress(endpoint string) (string, string, error) {
	seps := strings.SplitN(endpoint, "://", 2)
	if len(seps) != 2 {
		return "", "", fmt.Errorf("malformed unix socket")
	}

	if len(seps[1]) > len(syscall.RawSockaddrUnix{}.Path) {
		return "", "", fmt.Errorf("unix socket path %q is too long", seps[1])
	}

	return seps[0], seps[1], nil
}

func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
