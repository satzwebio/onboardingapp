package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
)

// TeamOnboardingAppSpec defines the desired state of TeamOnboardingApp
type TeamOnboardingAppSpec struct {
	TeamName    string `json:"teamName"`
	Environment string `json:"environment"`
	Namespace   string `json:"namespace"`
	WebApp      WebAppSpec `json:"webApp"`
	Database    DatabaseSpec `json:"database"`
	ConfigMaps  []ConfigMapSpec `json:"configMaps,omitempty"`
	Secrets     []SecretSpec `json:"secrets,omitempty"`
}

type WebAppSpec struct {
	Image     string `json:"image"`
	Replicas  int32  `json:"replicas"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type DatabaseSpec struct {
	Image     string `json:"image"`
	Replicas  int32  `json:"replicas"`
	Storage   StorageSpec `json:"storage"`
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type StorageSpec struct {
	StorageClassName string `json:"storageClassName"`
	Size            string `json:"size"`
}

type ConfigMapSpec struct {
	Name string `json:"name"`
	Data map[string]string `json:"data"`
}

type SecretSpec struct {
	Name       string `json:"name"`
	StringData map[string]string `json:"stringData"`
}

// TeamOnboardingAppStatus defines the observed state of TeamOnboardingApp
type TeamOnboardingAppStatus struct {
	Phase      string `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Team",type="string",JSONPath=".spec.teamName"
//+kubebuilder:printcolumn:name="Environment",type="string",JSONPath=".spec.environment"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TeamOnboardingApp is the Schema for the teamonboardingapps API
type TeamOnboardingApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamOnboardingAppSpec   `json:"spec,omitempty"`
	Status TeamOnboardingAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TeamOnboardingAppList contains a list of TeamOnboardingApp
type TeamOnboardingAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TeamOnboardingApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TeamOnboardingApp{}, &TeamOnboardingAppList{})
}