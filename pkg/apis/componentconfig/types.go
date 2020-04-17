package componentconfig

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HookConfiguration struct {
	metav1.TypeMeta
	Timeout        time.Duration
	ListenAddress  string
	RemoteEndpoint string
	WebHooks       HookConfigurationList
}

type HookConfigurationList []HookConfigurationItem

type HookConfigurationItem struct {
	Name          string
	Endpoint      string
	FailurePolicy FailurePolicyType
	Stages        HookStageList
}

type HookStageList []HookStage

type HookStage struct {
	Method     string
	URLPattern string
}

type FailurePolicyType string

const (
	PolicyFail   FailurePolicyType = "Fail"
	PolicyIgnore FailurePolicyType = "Ignore"
)
