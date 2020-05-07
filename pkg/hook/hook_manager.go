package hook

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	systemd "github.com/coreos/go-systemd/v22/daemon"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
	"github.com/mYmNeo/lighthouse/pkg/util"
)

type hookManager struct {
	timeout       time.Duration
	listenAddress string
	mux           *mux.Router
	backend       http.Handler
}

func NewHookManager() *hookManager {
	hm := &hookManager{
		mux: mux.NewRouter(),
	}

	return hm
}

func (hm *hookManager) Run(stop <-chan struct{}) error {
	proto, addr, err := util.GetProtoAndAddress(hm.listenAddress)
	if err != nil {
		return err
	}

	/** Abstract unix socket is not supported */
	if proto == util.UnixProto {
		if strings.HasPrefix(addr, "@") {
			klog.Fatalf("can't use abstract unix socket %s", addr)
		}
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	l, err := net.Listen(proto, addr)
	if err != nil {
		return err
	}

	ready := make(chan struct{})
	ch := make(chan error)
	go func() {
		close(ready)
		ch <- http.Serve(l, hm)
	}()

	<-ready
	klog.Infof("Hook manager is running")

	sent, err := systemd.SdNotify(true, "READY=1\n")
	if err != nil {
		klog.Warningf("Unable to send systemd daemon successful start message: %v\n", err)
	}

	if !sent {
		klog.Warningf("Unable to send systemd daemon Type=notify in systemd service file?")
	}

	select {
	case <-stop:
	case e := <-ch:
		return e
	}

	return nil
}

func (hm *hookManager) InitFromConfig(config *componentconfig.HookConfiguration) error {
	klog.Infof("Hook timeout: %d seconds", config.Timeout)
	hm.timeout = config.Timeout * time.Second
	hm.backend = newReverseProxy(config.RemoteEndpoint)
	hm.listenAddress = config.ListenAddress

	patternMap := make(map[componentconfig.HookStage][]HookHandler)

	for _, r := range config.WebHooks {
		klog.Infof("Register hook %s, endpoint %s", r.Name, r.Endpoint)
		hc := newHookConnector(r.Name, r.Endpoint, r.FailurePolicy)
		for _, fp := range r.Stages {
			klog.Infof("Register %s %s with %s", fp.Method, fp.URLPattern, hc.endpoint)
			handlers, found := patternMap[fp]
			if !found {
				handlers = make([]HookHandler, 0)
			}
			handlers = append(handlers, hc)
			patternMap[fp] = handlers
		}
	}

	for st, handlers := range patternMap {
		klog.V(2).Infof("Build router: %s %s", st.Method, st.URLPattern)
		chainHandler := hm.buildHookHandlerFunc(handlers)
		hm.mux.Methods(st.Method).Path(st.URLPattern).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := chainHandler(w, r); err != nil {
				return
			}
			klog.V(4).Infof("Send data to backend path %s", r.URL.Path)
			hm.backend.ServeHTTP(w, r)
			klog.V(4).Infof("Finish backend path %s", r.URL.Path)
		})
	}

	return nil
}

func (hm *hookManager) buildHookHandlerFunc(handlers []HookHandler) func(w http.ResponseWriter, r *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
		defer cancel()

		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			klog.Errorf("can't read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return err
		}

		klog.V(4).Infof("PreHook request %s, body: %s", r.URL.Path, string(bodyBytes))
		for idx, h := range handlers {
			hookErr := func() error {
				klog.V(4).Infof("Send to hook handler %d", idx)
				patch := &PatchData{}
				if err := h.Hook(ctx, patch, r.URL.Path, bodyBytes); err != nil {
					klog.Errorf("hook failed, %v", err)
					return err
				}

				if patch.PatchData == nil {
					return nil
				}

				switch types.PatchType(patch.PatchType) {
				case types.JSONPatchType:
					p, err := jsonpatch.DecodePatch(patch.PatchData)
					if err != nil {
						klog.Errorf("can't decode patch, %v", err)
						return err
					}
					bodyBytes, err = p.Apply(bodyBytes)
					if err != nil {
						klog.Errorf("can't apply patch, %v", err)
						return err
					}
				case types.MergePatchType:
					bodyBytes, err = jsonpatch.MergePatch(bodyBytes, patch.PatchData)
					if err != nil {
						klog.Errorf("can't merge patch, %v", err)
						return err
					}
				default:
					return fmt.Errorf("unknown patch type: %s", patch.PatchType)
				}

				return nil
			}()

			if hookErr == nil {
				continue
			}

			klog.Errorf("can't perform hook, %v", hookErr)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(hookErr.Error()))
			return hookErr
		}

		klog.V(4).Infof("After preHook request %s, body: %s", r.URL.Path, string(bodyBytes))
		newBody := bytes.NewBuffer(bodyBytes)
		r.Body = ioutil.NopCloser(newBody)
		r.ContentLength = int64(newBody.Len())
		return nil
	}
}

func (hm *hookManager) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var match mux.RouteMatch
	if hm.mux.Match(req, &match) {
		hm.mux.ServeHTTP(w, req)
		return
	}
	hm.backend.ServeHTTP(w, req)
}
