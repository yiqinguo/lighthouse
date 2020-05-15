package hook

import (
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/mYmNeo/lighthouse/pkg/apis/componentconfig"
)

var json = jsoniter.Config{
	EscapeHTML:             false,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
}.Froze()

func HookPath(hookType componentconfig.HookType, path string) string {
	return strings.ToLower(strings.Join([]string{"/", string(hookType), path}, ""))
}

func fixUnexpectedEscape(d []byte) []byte {
	return []byte(strings.ReplaceAll(strings.ReplaceAll(string(d), `\u003c`, "<"), `\u003e`, ">"))
}
