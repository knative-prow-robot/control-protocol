/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8s

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/kmeta"
	pkgsecurity "knative.dev/pkg/test/security"
)

func GetFirstTerminationMessage(pod *corev1.Pod) string {
	if pod != nil {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Terminated != nil && cs.State.Terminated.Message != "" {
				return cs.State.Terminated.Message
			}
		}
	}
	return ""
}

func GetOperationsResult(ctx context.Context, pod *corev1.Pod, result interface{}) error {
	if pod == nil {
		return fmt.Errorf("pod was nil")
	}
	terminationMessage := GetFirstTerminationMessage(pod)
	if terminationMessage == "" {
		return fmt.Errorf("did not find termination message for pod %q", pod.Name)
	}
	err := json.Unmarshal([]byte(terminationMessage), &result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal terminationmessage: %q : %q", terminationMessage, err)
	}
	return nil
}

// PodReference will return a reference to the pod.
func PodReference(namespace string, name string) (corev1.ObjectReference, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: namespace,
		},
	}
	scheme := runtime.NewScheme()
	err := corev1.SchemeBuilder.AddToScheme(scheme)
	if err != nil {
		return corev1.ObjectReference{}, errors.WithStack(err)
	}
	kinds, _, err := scheme.ObjectKinds(pod)
	if err != nil {
		return corev1.ObjectReference{}, errors.WithStack(err)
	}
	if !(len(kinds) > 0) {
		return corev1.ObjectReference{}, errors.New("want len(kinds) > 0")
	}
	kind := kinds[0]
	pod.APIVersion, pod.Kind = kind.ToAPIVersionAndKind()
	return kmeta.ObjectReference(pod), nil
}

func WithDefaultPodSecurityContext(cfg map[string]interface{}) {
	if _, set := cfg["podSecurityContext"]; !set {
		cfg["podSecurityContext"] = map[string]interface{}{}
	}
	podSecurityContext := cfg["podSecurityContext"].(map[string]interface{})
	podSecurityContext["runAsNonRoot"] = pkgsecurity.DefaultPodSecurityContext.RunAsNonRoot
	podSecurityContext["seccompProfile"] = map[string]interface{}{}
	seccompProfile := podSecurityContext["seccompProfile"].(map[string]interface{})
	seccompProfile["type"] = pkgsecurity.DefaultPodSecurityContext.SeccompProfile.Type

	if _, set := cfg["containerSecurityContext"]; !set {
		cfg["containerSecurityContext"] = map[string]interface{}{}
	}
	containerSecurityContext := cfg["containerSecurityContext"].(map[string]interface{})
	containerSecurityContext["allowPrivilegeEscalation"] =
		pkgsecurity.DefaultContainerSecurityContext.AllowPrivilegeEscalation
	containerSecurityContext["capabilities"] = map[string]interface{}{}
	capabilities := containerSecurityContext["capabilities"].(map[string]interface{})
	if len(pkgsecurity.DefaultContainerSecurityContext.Capabilities.Drop) != 0 {
		capabilities["drop"] = []string{}
		for _, drop := range pkgsecurity.DefaultContainerSecurityContext.Capabilities.Drop {
			capabilities["drop"] = append(capabilities["drop"].([]string), string(drop))
		}
	}
	if len(pkgsecurity.DefaultContainerSecurityContext.Capabilities.Add) != 0 {
		capabilities["add"] = []string{}
		for _, drop := range pkgsecurity.DefaultContainerSecurityContext.Capabilities.Drop {
			capabilities["add"] = append(capabilities["add"].([]string), string(drop))
		}
	}
}
