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

var _ HookHandler = (*hookerConnector)(nil)

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

func (hc *hookerConnector) performHook(ctx context.Context, patch *PatchData, method, path string, body []byte) error {
	url := fmt.Sprintf("http://%s", hc.endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil && !hc.allowFailure() {
		klog.Errorf("can't create request %s, %v", url, err)
		return err
	}

	if req != nil {
		req.URL.Path = path

		klog.V(4).Infof("Send request %s %s for %s", method, path, hc.name)
		resp, err := hc.client.Do(req)
		if err != nil && !hc.allowFailure() {
			return err
		}

		if resp != nil {
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && !hc.allowFailure() {
				return fmt.Errorf("post is not success, status code is %d", resp.StatusCode)
			}

			if resp.Body != nil {
				klog.V(4).Infof("Decode response %s for %s", path, hc.name)
				return json.NewDecoder(resp.Body).Decode(patch)
			}
		}
	}

	return nil
}

func (hc *hookerConnector) PreHook(ctx context.Context, patch *PatchData, method, path string, body []byte) error {
	return hc.performHook(ctx, patch, method, HookPath(componentconfig.PreHookType, path), body)
}

func (hc *hookerConnector) PostHook(ctx context.Context, patch *PatchData, method, path string, body []byte) error {
	return hc.performHook(ctx, patch, method, HookPath(componentconfig.PostHookType, path), body)
}

func (hc *hookerConnector) allowFailure() bool {
	return hc.failurePolicy == componentconfig.PolicyIgnore
}
