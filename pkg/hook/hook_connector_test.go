package hook

import (
	"context"
	"flag"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
	"github.com/mYmNeo/lighthouse/pkg/test"
)

func init() {
	klog.InitFlags(nil)
}

func TestHookConnectorPreHook(t *testing.T) {
	flag.Set("v", "4")
	flag.Parse()

	testUnits := []*testConnectorUnit{
		{
			patch: &PatchData{
				PatchType: string(types.JSONPatchType),
				PatchData: []byte(`[{"op":"replace", "path":"/foo", "value":"bar"}]`),
			},
			payload: `{"foo":1}`,
			path:    "/replace",
		},
		{
			patch: &PatchData{
				PatchType: string(types.MergePatchType),
				PatchData: []byte(`{}`),
			},
			payload: `{"foo":"bar"}`,
			path:    "/merge",
		},
	}
	server := test.NewUnixSocketServer()

	for i := range testUnits {
		u := testUnits[i]
		path := HookPath(componentconfig.PreHookType, u.path)
		server.RegisterHandler(path, func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodPost {
				t.Errorf("expect request method %s to be %s", req.Method, http.MethodPost)
				return
			}

			bodyBytes, err := ioutil.ReadAll(req.Body)
			if err != nil {
				t.Errorf("can't read body, %v", err)
				return
			}

			if u.payload != string(bodyBytes) {
				t.Errorf("expect payload %s to be %s", string(bodyBytes), u.payload)
				return
			}

			json.NewEncoder(w).Encode(u.patch)
		})
	}

	ready := make(chan struct{})
	go func() {
		close(ready)
		server.Start()
	}()

	defer server.Stop()
	<-ready

	hc := newHookConnector("test", server.GetAddress(), componentconfig.PolicyFail)

	for _, u := range testUnits {
		p := &PatchData{}
		if err := hc.PreHook(context.Background(), p, http.MethodPost, u.path, []byte(u.payload)); err != nil {
			t.Errorf("can't perform a hook, %v", err)
			return
		}

		if !reflect.DeepEqual(p, u.patch) {
			t.Errorf("expected %+#v, got %+#v", p, u.patch)
			return
		}
	}
}

type testConnectorUnit struct {
	patch   *PatchData
	path    string
	payload string
}
