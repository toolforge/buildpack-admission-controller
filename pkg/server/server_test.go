package server

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

var (
	AdmissionRequestFail = admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "e911777a-c418-11e8-bbad-025000000001",
			Kind: metav1.GroupVersionKind{
				Group: "tekton.dev", Version: "v1beta1", Kind: "PipelineRun",
			},
			Namespace: "image-build",
			Operation: "CREATE",
			UserInfo: authenticationv1.UserInfo{
				Username: "test",
				UID:      "25",
				Groups:   []string{"toolforge"},
			},
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"apiVersion": "tekton.dev/v1beta1",
					"kind": "PipelineRun",
					"metadata": {
					    "name": "test2run",
					    "namespace": "image-build"
					},
					"spec": {
					    "pipelineRef": {
					        "name": "buildpacks"
					    },
					    "params": [
						     {
						         "name": "BUILDER_IMAGE",
						         "value": "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"
						     },
						     {
							"name": "APP_IMAGE",
						        "value": "harbor.toolsbeta.wmflabs.org/test4/python:snap"
						     },
						     {
							"name": "SOURCE_URL",
						        "value": "https://github.com/earwig/mwparserfromhell"
						     }
					    ]
					}
				    }`),
			},
		},
	}
	AdmissionRequestPass = admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind: "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: "e911857d-c318-11e8-bbad-025000000001",
			Kind: metav1.GroupVersionKind{
				Group: "tekton.dev", Version: "v1beta1", Kind: "PipelineRun",
			},
			Operation: "CREATE",
			Namespace: "image-build",
			UserInfo: authenticationv1.UserInfo{
				Username: "test",
				UID:      "25",
				Groups:   []string{"toolforge"},
			},
			Object: runtime.RawExtension{
				Raw: []byte(`{
					"kind": "PipelineRun",
					"apiVersion": "tekton.dev/v1beta1",
					"metadata": {
					    "name": "tool-test",
					    "namespace": "image-build",
					    "uid": "4b54be10-8d3c-11e9-8b7a-080027f5f85c",
					    "creationTimestamp": "2019-06-12T18:02:51Z"
					},
					"spec": {
					    "pipelineRef": {
					        "name": "buildpacks"
					    },
					    "params": [
						     {
						         "name": "BUILDER_IMAGE",
						         "value": "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"
						     },
						     {
							"name": "APP_IMAGE",
						        "value": "harbor.toolsbeta.wmflabs.org/test/python:latest"
						     },
						     {
							"name": "SOURCE_URL",
						        "value": "https://github.com/earwig/mwparserfromhell"
						     }
					    ]
					}
				    }`),
			},
		},
	}
)

func decodeResponse(body io.ReadCloser) *admissionv1.AdmissionReview {
	response, _ := ioutil.ReadAll(body)
	review := &admissionv1.AdmissionReview{}
	codecs.UniversalDeserializer().Decode(response, nil, review)
	return review
}

func encodeRequest(review *admissionv1.AdmissionReview) []byte {
	ret, err := json.Marshal(review)
	if err != nil {
		logrus.Errorln(err)
	}
	return ret
}

func TestServeReturnsCorrectJson(t *testing.T) {
	inc := &PipelineRunAdmission{
		Domains:  []string{"harbor.toolforge.org", "harbor.toolsbeta.wmflabs.org"},
		Builders: []string{"paketobuildpacks/builder:base", "gcr.io/buildpacks/builder:v1", "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(inc, ":8080").Handler)
	requestString := string(encodeRequest(&AdmissionRequestPass))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if review.Request.UID != AdmissionRequestPass.Request.UID {
		t.Error("Request and response UID don't match")
	}
}
func TestHookFailsOnBadPipelineRun(t *testing.T) {
	nsc := &PipelineRunAdmission{
		Domains:  []string{"harbor.toolforge.org", "harbor.toolsbeta.wmflabs.org"},
		Builders: []string{"paketobuildpacks/builder:base", "gcr.io/buildpacks/builder:v1", "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	requestString := string(encodeRequest(&AdmissionRequestFail))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if review.Response.Allowed {
		t.Error("Allowed pipelinerun that should not have been allowed!")
	}
}
func TestHookPassesOnRightPipelineRun(t *testing.T) {
	nsc := &PipelineRunAdmission{
		Domains:  []string{"harbor.toolforge.org", "harbor.toolsbeta.wmflabs.org"},
		Builders: []string{"paketobuildpacks/builder:base", "gcr.io/buildpacks/builder:v1", "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	requestString := string(encodeRequest(&AdmissionRequestPass))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if !review.Response.Allowed {
		t.Error("Failed to allow pipelinerun should have been allowed!")
	}
}
