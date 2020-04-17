package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type HookConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	Timeout         time.Duration         `json:"timeout,omitempty"`
	ListenAddress   string                `json:"listenAddress,omitempty"`
	RemoteEndpoint  string                `json:"remoteEndpoint,omitempty"`
	WebHooks        HookConfigurationList `json:"webhooks,omitempty"`
}

type HookConfigurationList []HookConfigurationItem

type HookConfigurationItem struct {
	Name          string            `json:"name,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty"`
	FailurePolicy FailurePolicyType `json:"failurePolicy,omitempty"`
	Stages        HookStageList     `json:"stages,omitempty"`
}

type HookStageList []HookStage

type HookStage struct {
	Method     string `json:"method,omitempty"`
	URLPattern string `json:"urlPattern,omitempty"`
}

type FailurePolicyType string

const (
	PolicyFail   FailurePolicyType = "Fail"
	PolicyIgnore FailurePolicyType = "Ignore"
)
