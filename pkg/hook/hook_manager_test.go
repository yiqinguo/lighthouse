package hook

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/types"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
	"github.com/mYmNeo/lighthouse/pkg/test"
)

func TestHookManagerPreHook(t *testing.T) {
	flag.Set("v", "4")
	flag.Parse()

	testUnits := []*testHookManagerUnit{
		{
			path:     fmt.Sprintf("/container/%s/create", uuid.New().String()),
			pattern:  "/container/{id:[-a-z0-9]+}/create",
			payload:  `{"foo":"bar"}`,
			expected: `{"a":"b"}`,
			patches: []*PatchData{
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"replace","path":"/foo", "value":"1"}]`),
				},
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"remove","path":"/foo"}]`),
				},
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"add","path":"/a","value":"b"}]`),
				},
			},
		},
		{
			path:     fmt.Sprintf("/container/%s/create", uuid.New().String()),
			pattern:  "/container/{id:[0-9]+}/create}",
			payload:  `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
			patches: []*PatchData{
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"replace","path":"/foo", "value":"1"}]`),
				},
			},
		},
	}

	for i := range testUnits {
		func() {
			u := testUnits[i]
			backendServer := createTestServerBundle(1)
			hookServer := createTestServerBundle(len(u.patches))

			for j := range u.patches {
				p := u.patches[j]
				hookServer.servers[j].RegisterHandler(HookPath(componentconfig.PreHookType, u.path), func(w http.ResponseWriter,
					r *http.Request) {
					json.NewEncoder(w).Encode(p)
				})
			}

			backendServer.servers[0].RegisterHandler(u.path, func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("can't get body: %v", err)
					return
				}

				if string(bodyBytes) != u.expected {
					t.Errorf("%d expected: %s to be %s", i, string(bodyBytes), u.expected)
					return
				}
				w.WriteHeader(http.StatusOK)
			})

			cfg := &componentconfig.HookConfiguration{
				Timeout:        10,
				ListenAddress:  fmt.Sprintf("unix://@%s", uuid.New().String()),
				RemoteEndpoint: backendServer.servers[0].GetAddress(),
				WebHooks:       make(componentconfig.HookConfigurationList, len(u.patches)),
			}

			for j := range u.patches {
				webhook := &cfg.WebHooks[j]

				webhook.Endpoint = hookServer.servers[j].GetAddress()
				webhook.Name = fmt.Sprintf("hook-%d", j)
				webhook.FailurePolicy = componentconfig.PolicyFail
				webhook.Stages = append(webhook.Stages, componentconfig.HookStage{
					Method:     http.MethodPost,
					URLPattern: u.pattern,
					Type:       componentconfig.PreHookType,
				})
			}

			hm := NewHookManager()
			if err := hm.InitFromConfig(cfg); err != nil {
				t.Errorf("can't init hook manager: %v", err)
				return
			}

			totalServerNum := len(backendServer.servers) + len(hookServer.servers)
			readyCh := make(chan bool, totalServerNum)
			go func() {
				backendServer.Start(readyCh)
			}()

			go func() {
				hookServer.Start(readyCh)
			}()

			defer func() {
				backendServer.Stop()
				hookServer.Stop()
			}()

			for len(readyCh) != totalServerNum {
				time.Sleep(time.Second)
			}

			ans := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s", cfg.ListenAddress),
				bytes.NewBuffer([]byte(u.payload)))
			if err != nil {
				t.Errorf("can't create HTTP request: %v", err)
				return
			}
			req.URL.Path = u.path

			hm.ServeHTTP(ans, req)

			resp := ans.Result()
			if resp == nil {
				t.Errorf("resp is nil")
				return
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected %d to be %d", resp.StatusCode, http.StatusOK)
				return
			}
		}()
	}
}

func TestHookManagerPostHook(t *testing.T) {
	flag.Set("v", "4")
	flag.Parse()

	testUnits := []*testHookManagerUnit{
		{
			path:     fmt.Sprintf("/container/%s/create", uuid.New().String()),
			pattern:  "/container/{id:[-a-z0-9]+}/create",
			payload:  `{"foo":"bar"}`,
			expected: `{"a":"b"}`,
			patches: []*PatchData{
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"replace","path":"/body/foo", "value":"1"}]`),
				},
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"remove","path":"/body/foo"}]`),
				},
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"add","path":"/body/a","value":"b"}]`),
				},
			},
		},
		{
			path:     fmt.Sprintf("/container/%s/create", uuid.New().String()),
			pattern:  "/container/{id:[0-9]+}/create}",
			payload:  `{"foo":"bar"}`,
			expected: `{"foo":"bar"}`,
			patches: []*PatchData{
				{
					PatchType: string(types.JSONPatchType),
					PatchData: []byte(`[{"op":"replace","path":"/body/foo", "value":"1"}]`),
				},
			},
		},
	}

	for i := range testUnits {
		func() {
			u := testUnits[i]
			backendServer := createTestServerBundle(1)
			hookServer := createTestServerBundle(len(u.patches))

			for j := range u.patches {
				p := u.patches[j]
				hookServer.servers[j].RegisterHandler(HookPath(componentconfig.PostHookType, u.path), func(w http.ResponseWriter,
					r *http.Request) {
					json.NewEncoder(w).Encode(p)
				})
			}

			backendServer.servers[0].RegisterHandler(u.path, func(w http.ResponseWriter, r *http.Request) {
				bodyBytes, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Errorf("can't get body: %v", err)
					return
				}

				if string(bodyBytes) != u.payload {
					t.Errorf("expected: %s to be %s", string(bodyBytes), u.expected)
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write(bodyBytes)
			})

			cfg := &componentconfig.HookConfiguration{
				Timeout:        10,
				ListenAddress:  fmt.Sprintf("unix://@%s", uuid.New().String()),
				RemoteEndpoint: backendServer.servers[0].GetAddress(),
				WebHooks:       make(componentconfig.HookConfigurationList, len(u.patches)),
			}

			for j := range u.patches {
				webhook := &cfg.WebHooks[j]

				webhook.Endpoint = hookServer.servers[j].GetAddress()
				webhook.Name = fmt.Sprintf("hook-%d", j)
				webhook.FailurePolicy = componentconfig.PolicyFail
				webhook.Stages = append(webhook.Stages, componentconfig.HookStage{
					Method:     http.MethodPost,
					URLPattern: u.pattern,
					Type:       componentconfig.PostHookType,
				})
			}

			hm := NewHookManager()
			if err := hm.InitFromConfig(cfg); err != nil {
				t.Errorf("can't init hook manager: %v", err)
				return
			}

			totalServerNum := len(backendServer.servers) + len(hookServer.servers)
			readyCh := make(chan bool, totalServerNum)
			go func() {
				backendServer.Start(readyCh)
			}()

			go func() {
				hookServer.Start(readyCh)
			}()

			defer func() {
				backendServer.Stop()
				hookServer.Stop()
			}()

			for len(readyCh) != totalServerNum {
				time.Sleep(time.Second)
			}

			ans := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s", cfg.ListenAddress),
				bytes.NewBuffer([]byte(u.payload)))
			if err != nil {
				t.Errorf("can't create HTTP request: %v", err)
				return
			}
			req.URL.Path = u.path

			hm.ServeHTTP(ans, req)

			resp := ans.Result()
			if resp == nil {
				t.Errorf("resp is nil")
				return
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected %d to be %d", resp.StatusCode, http.StatusOK)
				return
			}

			body, err := ioutil.ReadAll(resp.Body)
			if string(body) != u.expected {
				t.Errorf("%d expect body %s to be %s", i, string(body), u.expected)
				return
			}
		}()
	}
}

type testServerBundle struct {
	servers []*test.UnixSocketServer
}

func createTestServerBundle(num int) *testServerBundle {
	b := &testServerBundle{
		servers: make([]*test.UnixSocketServer, num),
	}

	for i := 0; i < num; i++ {
		b.servers[i] = test.NewUnixSocketServer()
	}

	return b
}

func (b *testServerBundle) Start(ready chan bool) error {
	ch := make(chan error, len(b.servers))

	for i := range b.servers {
		s := b.servers[i]
		go func() {
			ready <- true
			ch <- s.Start()
		}()
	}

	select {
	case e := <-ch:
		return e
	}
}

func (b *testServerBundle) Stop() {
	for _, s := range b.servers {
		s.Stop()
	}
}

type testHookManagerUnit struct {
	path     string
	pattern  string
	payload  string
	expected string
	patches  []*PatchData
}
