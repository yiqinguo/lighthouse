package hook

import (
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func HookPath(hookType componentconfig.HookType, path string) string {
	return strings.ToLower(strings.Join([]string{"/", string(hookType), path}, ""))
}
