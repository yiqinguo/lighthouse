package hook

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
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

type hookHandleKey struct {
	Method     string
	URLPattern string
}

type hookHandleData struct {
	preHooks  []HookHandler
	postHooks []HookHandler
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

	hooksMap := make(map[hookHandleKey]*hookHandleData)
	for _, r := range config.WebHooks {
		klog.Infof("Register hook %s, endpoint %s", r.Name, r.Endpoint)
		hc := newHookConnector(r.Name, r.Endpoint, r.FailurePolicy)
		for _, fp := range r.Stages {
			klog.Infof("Register %s %s %s with %s", fp.Type, fp.Method, fp.URLPattern, hc.endpoint)
			key := hookHandleKey{
				Method:     fp.Method,
				URLPattern: fp.URLPattern,
			}

			hookData, found := hooksMap[key]
			if !found {
				hookData = &hookHandleData{
					preHooks:  make([]HookHandler, 0),
					postHooks: make([]HookHandler, 0),
				}
			}

			switch fp.Type {
			case componentconfig.PreHookType:
				hookData.preHooks = append(hookData.preHooks, hc)
				hooksMap[key] = hookData
			case componentconfig.PostHookType:
				hookData.postHooks = append(hookData.postHooks, hc)
				hooksMap[key] = hookData
			}
		}
	}

	for k, v := range hooksMap {
		klog.V(2).Infof("Build router: %s %s", k.Method, k.URLPattern)
		preHookChainHandler := hm.buildPreHookHandlerFunc(v.preHooks)
		postHookChainHandler := hm.buildPostHookHandlerFunc(v.postHooks)

		hm.mux.Methods(k.Method).Path(k.URLPattern).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := preHookChainHandler(w, r); err != nil {
				return
			}

			klog.V(4).Infof("Send data to backend path %s", r.URL.Path)
			recorder := httptest.NewRecorder()
			hm.backend.ServeHTTP(recorder, r)
			klog.V(4).Infof("Finish backend path %s", r.URL.Path)

			postHookChainHandler(recorder, r)
			for k, vs := range recorder.Header() {
				for _, v := range vs {
					w.Header().Set(k, v)
				}
			}
			w.WriteHeader(recorder.Code)
			w.Write(recorder.Body.Bytes())
		})
	}

	return nil
}

func (hm *hookManager) applyHook(ctx context.Context, handlers []HookHandler, hookType componentconfig.HookType, path string,
	body *[]byte) error {
	var err error

	for idx, h := range handlers {
		hookErr := func() error {
			klog.V(4).Infof("Send to %s handler %d", hookType, idx)
			patch := &PatchData{}

			switch hookType {
			case componentconfig.PreHookType:
				if err := h.PreHook(ctx, patch, path, *body); err != nil {
					klog.Errorf("preHook failed, %v", err)
					return err
				}
			case componentconfig.PostHookType:
				if err := h.PostHook(ctx, patch, path, *body); err != nil {
					klog.Errorf("preHook failed, %v", err)
					return err
				}
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
				*body, err = p.Apply(*body)
				if err != nil {
					klog.Errorf("can't apply patch, %v", err)
					return err
				}
			case types.MergePatchType:
				*body, err = jsonpatch.MergePatch(*body, patch.PatchData)
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

		klog.Errorf("can't perform %s, %v", hookType, hookErr)
		return hookErr
	}

	return nil
}

func (hm *hookManager) buildPostHookHandlerFunc(handlers []HookHandler) PostHookFunc {
	return func(w *httptest.ResponseRecorder, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
		defer cancel()

		data := &PostHookData{
			StatusCode: w.Code,
			Body:       w.Body.Bytes(),
		}

		w.Body.Reset()

		bodyBytes, err := json.Marshal(data)
		if err != nil {
			klog.Errorf("can't marshal post hook data, %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		klog.V(4).Infof("PostHook request %s, body: %s", r.URL.Path, w.Body.String())
		if err := hm.applyHook(ctx, handlers, componentconfig.PostHookType, r.URL.Path, &bodyBytes); err != nil {
			klog.Errorf("can't perform postHook, %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if err := json.Unmarshal(bodyBytes, data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		klog.V(4).Infof("After postHook request %s, body: %s", r.URL.Path, string(data.Body))
		w.Write(data.Body)
	}
}

func (hm *hookManager) buildPreHookHandlerFunc(handlers []HookHandler) PreHookFunc {
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
		if err := hm.applyHook(ctx, handlers, componentconfig.PreHookType, r.URL.Path, &bodyBytes); err != nil {
			klog.Errorf("can't perform postHook, %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return err
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
