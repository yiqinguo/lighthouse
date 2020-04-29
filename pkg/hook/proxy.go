package hook

import (
	"net/http"
	"net/http/httputil"

	"github.com/docker/go-connections/sockets"
	"k8s.io/klog"

	"github.com/mYmNeo/lighthouse/pkg/util"
)

type reverseProxy struct {
	proxy *httputil.ReverseProxy
}

func newReverseProxy(remoteEndpoint string) *reverseProxy {
	proto, addr, err := util.GetProtoAndAddress(remoteEndpoint)
	if err != nil {
		klog.Fatalf("can't parse remote endpoint %s, %v", remoteEndpoint, err)
	}

	tr := new(http.Transport)
	sockets.ConfigureTransport(tr, proto, addr)

	rp := &reverseProxy{
		proxy: &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = addr
				if _, ok := req.Header["User-Agent"]; !ok {
					// explicitly disable User-Agent so it's not set to default value
					req.Header.Set("User-Agent", "")
				}
			},
			Transport: tr,
		},
	}

	return rp
}

func (rp *reverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	klog.V(8).Infof("Serve request %s", req.URL.String())
	rp.proxy.ServeHTTP(w, req)
}
