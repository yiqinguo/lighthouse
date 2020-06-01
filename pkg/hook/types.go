package hook

import (
	"context"
	gjson "encoding/json"
	"net/http"
	"net/http/httptest"
)

type PatchData struct {
	PatchType string `json:"patchType,omitempty"`
	PatchData []byte `json:"patchData,omitempty"`
}

type PostHookData struct {
	StatusCode int              `json:"statusCode,omitempty"`
	Body       gjson.RawMessage `json:"body,omitempty"`
}

type HookHandler interface {
	PreHook(ctx context.Context, patch *PatchData, method, path string, body []byte) error
	PostHook(ctx context.Context, patch *PatchData, method, path string, body []byte) error
}

type PreHookFunc func(w http.ResponseWriter, r *http.Request) error
type PostHookFunc func(w *httptest.ResponseRecorder, r *http.Request)
