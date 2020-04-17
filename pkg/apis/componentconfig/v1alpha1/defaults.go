package v1alpha1

import (
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func SetDefaults_HookConfiguration(obj *HookConfiguration) {
	if obj.Timeout == 0 {
		obj.Timeout = 5
	}

	if obj.RemoteEndpoint == "" {
		obj.RemoteEndpoint = "unix:///var/run/docker.sock"
	}
}

func SetDefaults_HookConfigurationItem(obj *HookConfigurationItem) {
	if obj.FailurePolicy == "" {
		obj.FailurePolicy = PolicyFail
	}
}

func SetDefaults_HookStage(obj *HookStage) {
	if obj.Method == "" {
		obj.Method = http.MethodPost
	}
}
