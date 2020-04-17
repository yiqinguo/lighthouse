package hook

import (
	"context"

	jsoniter "github.com/json-iterator/go"
)

type PatchData struct {
	PatchType string `json:"patchType,omitempty"`
	PatchData []byte `json:"patchData,omitempty"`
}

type HookHandler interface {
	Hook(ctx context.Context, patch *PatchData, path string, body []byte) error
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary
