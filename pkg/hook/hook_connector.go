package hook

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
	"github.com/mYmNeo/lighthouse/pkg/util"
)

type hookerConnector struct {
	name          string
	endpoint      string
	failurePolicy componentconfig.FailurePolicyType
	client        *http.Client
}

var _ HookHandler = &hookerConnector{}

func newHookConnector(name, endpoint string, failurePolicy componentconfig.FailurePolicyType) *hookerConnector {
	hc := &hookerConnector{
		name:          name,
		endpoint:      endpoint,
		failurePolicy: failurePolicy,
	}

	if !strings.HasPrefix(endpoint, "unix://") {
		klog.Fatalf("only support unix protocol")
	}

	hc.client = util.BuildClientOrDie(endpoint)
	return hc
}

func (hc *hookerConnector) Hook(ctx context.Context, patch *PatchData, path string, body []byte) error {
	url := fmt.Sprintf("http://%s", hc.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil && !hc.allowFailure() {
		klog.Errorf("can't create request %s, %v", url, err)
		return err
	}

	if req != nil {
		req.URL.Path = path

		klog.V(4).Infof("Send hook request %s for %s", path, hc.name)
		resp, err := hc.client.Do(req)
		if err != nil && !hc.allowFailure() {
			return err
		}

		if resp != nil {
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && !hc.allowFailure() {
				return fmt.Errorf("post is not success")
			}

			klog.V(4).Infof("Decode hook response %s for %s", path, hc.name)
			return json.NewDecoder(resp.Body).Decode(patch)
		}
	}

	return nil
}

func (hc *hookerConnector) allowFailure() bool {
	return hc.failurePolicy == componentconfig.PolicyIgnore
}
