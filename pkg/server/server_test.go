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

func getAdmissionReview(user string, builderImage string, appImage string) admissionv1.AdmissionReview {
	if appImage == "" {
		appImage = "harbor.toolsbeta.wmflabs.org/test4/python:snap"
	}
	if builderImage == "" {
		builderImage = "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"
	}
	jsonObject := map[string]interface{}{
		"apiVersion": "tekton.dev/v1beta1",
		"kind":       "PipelineRun",
		"metadata": map[string]string{
			"name":      "test2run",
			"namespace": "image-build",
		},
		"spec": map[string]interface{}{
			"pipelineRef": map[string]string{
				"name": "buildpacks",
			},
			"params": []map[string]string{
				{
					"name":  "BUILDER_IMAGE",
					"value": builderImage,
				},
				{
					"name":  "APP_IMAGE",
					"value": appImage,
				},
				{
					"name":  "SOURCE_URL",
					"value": "https://github.com/earwig/mwparserfromhell",
				},
			},
		},
	}
	if user == "" {
		user = "test"
	}
	rawObject, err := json.Marshal(jsonObject)
	if err != nil {
		logrus.Errorln(err)
	}
	return admissionv1.AdmissionReview{
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
				Username: user,
				UID:      "25",
				Groups:   []string{"toolforge"},
			},
			Object: runtime.RawExtension{
				Raw: rawObject,
			},
		},
	}
}

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
	goodUser := "gooduser"
	goodDomain := "gooddomain"
	goodBuilder := "good/builder:v1"
	inc := &PipelineRunAdmission{
		AllowedDomains:  []string{goodDomain},
		AllowedBuilders: []string{goodBuilder},
		SystemUsers:     []string{goodUser},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(inc, ":8080").Handler)
	goodAdmissionReview := getAdmissionReview(goodUser, goodBuilder, goodDomain)
	requestString := string(encodeRequest(&goodAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if review.Request.UID != goodAdmissionReview.Request.UID {
		t.Error("Request and response UID don't match")
	}
}
func TestHookFailsOnBadPipelineRun(t *testing.T) {
	nsc := &PipelineRunAdmission{
		AllowedDomains:  []string{"harbor.toolforge.org", "harbor.toolsbeta.wmflabs.org"},
		AllowedBuilders: []string{"paketobuildpacks/builder:base", "gcr.io/buildpacks/builder:v1", "docker-registry.tools.wmflabs.org/toolforge-buster0-builder"},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	badAdmissionReview := getAdmissionReview("", "", "")
	requestString := string(encodeRequest(&badAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if review.Response.Allowed {
		t.Error("Allowed pipelinerun that should not have been allowed!")
	}
}

func TestHookDoesNotAllowBadUserBadDomain(t *testing.T) {
	badUser := "baduser"
	badDomain := "baddomain"
	goodBuilder := "good/builder:v1"
	nsc := &PipelineRunAdmission{
		AllowedDomains:  []string{"gooddomain"},
		AllowedBuilders: []string{goodBuilder},
		SystemUsers:     []string{"gooduser"},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	badAdmissionReview := getAdmissionReview(badUser, goodBuilder, badDomain)
	requestString := string(encodeRequest(&badAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if review.Response.Allowed {
		t.Error("Allowed pipelinerun that should not have been allowed!")
	}
}

func TestHookAllowsGoodUserBadDomain(t *testing.T) {
	goodUser := "gooduser"
	badDomain := "baddomain"
	goodBuilder := "good/builder:v1"
	nsc := &PipelineRunAdmission{
		AllowedDomains:  []string{"gooddomain"},
		AllowedBuilders: []string{goodBuilder},
		SystemUsers:     []string{goodUser},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	goodAdmissionReview := getAdmissionReview(goodUser, goodBuilder, badDomain)
	requestString := string(encodeRequest(&goodAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if !review.Response.Allowed {
		t.Error("Failed to allow pipelinerun should have been allowed!")
	}
}
func TestHookAllowsGoodUserGoodDomain(t *testing.T) {
	goodUser := "gooduser"
	goodDomain := "gooddomain"
	goodBuilder := "good/builder:v1"
	nsc := &PipelineRunAdmission{
		AllowedDomains:  []string{goodDomain},
		AllowedBuilders: []string{goodBuilder},
		SystemUsers:     []string{goodUser},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	goodAdmissionReview := getAdmissionReview(goodUser, goodBuilder, goodDomain)
	requestString := string(encodeRequest(&goodAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if !review.Response.Allowed {
		t.Error("Failed to allow pipelinerun should have been allowed!")
	}
}

func TestHookAllowsGoodUserGoodDomainWithHTTPProtocol(t *testing.T) {
	goodUser := "gooduser"
	goodDomain := "gooddomain"
	goodBuilder := "http://good/builder:v1"
	nsc := &PipelineRunAdmission{
		AllowedDomains:  []string{goodDomain},
		AllowedBuilders: []string{goodBuilder},
		SystemUsers:     []string{goodUser},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	goodAdmissionReview := getAdmissionReview(goodUser, goodBuilder, goodDomain)
	requestString := string(encodeRequest(&goodAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if !review.Response.Allowed {
		t.Error("Failed to allow pipelinerun should have been allowed!")
	}
}

func TestHookAllowsGoodUserGoodDomainWithHTTPSProtocol(t *testing.T) {
	goodUser := "gooduser"
	goodDomain := "gooddomain"
	goodBuilder := "https://good/builder:v1"
	nsc := &PipelineRunAdmission{
		AllowedDomains:  []string{goodDomain},
		AllowedBuilders: []string{goodBuilder},
		SystemUsers:     []string{goodUser},
	}
	server := httptest.NewServer(GetAdmissionServerNoSSL(nsc, ":8080").Handler)
	goodAdmissionReview := getAdmissionReview(goodUser, goodBuilder, goodDomain)
	requestString := string(encodeRequest(&goodAdmissionReview))
	myr := strings.NewReader(requestString)
	r, _ := http.Post(server.URL, "application/json", myr)
	review := decodeResponse(r.Body)
	t.Log(review.Response)
	if !review.Response.Allowed {
		t.Error("Failed to allow pipelinerun should have been allowed!")
	}
}
