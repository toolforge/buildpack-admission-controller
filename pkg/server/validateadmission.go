package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineRunAdmission type is where the project is stored and the handler method is linked
type PipelineRunAdmission struct {
	Domains  []string
	Builders []string
}

// HandleAdmission is the logic of the whole webhook, really.  This is where
// the decision to allow a Kubernetes ingress update or create or not takes place.
func (p *PipelineRunAdmission) HandleAdmission(review *admissionv1.AdmissionReview) error {
	// logrus.Debugln(review.Request)
	req := review.Request
	var pipelinerun pipelinev1.PipelineRun
	if err := json.Unmarshal(req.Object.Raw, &pipelinerun); err != nil {
		logrus.Errorf("Could not unmarshal raw object: %v", err)
		review.Response = &admissionv1.AdmissionResponse{
			UID:     review.Request.UID,
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
		return nil
	}
	logrus.Debugf("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pipelinerun.Name, req.UID, req.Operation, req.UserInfo)

	domstr := strings.Join(p.Domains, "|")
	buildstr := strings.Join(p.Builders, "|")
	user := req.UserInfo.Username
	for _, param := range pipelinerun.Spec.Params {
		harborRe := regexp.MustCompile(fmt.Sprintf("^%s/%s/", domstr, user))
		builderRe := regexp.MustCompile(fmt.Sprintf("^%s$", buildstr))
		logrus.Debugf("Found PipeLineRun param: %v", param.Name)
		if param.Name == "APP_IMAGE" && !harborRe.MatchString(param.Value.StringVal) {
			review.Response = &admissionv1.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: false,
				Result:  &metav1.Status{Message: fmt.Sprintf("Harbor path invalid: %s", param.Value.StringVal)},
			}
			return nil
		}
		if param.Name == "BUILDER_IMAGE" && !builderRe.MatchString(param.Value.StringVal) {
			fmt.Printf("builder: %v\n", param.Value.StringVal)
			logrus.Debugf("Found builder: %v", param.Value.StringVal)
			review.Response = &admissionv1.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: false,
				Result: &metav1.Status{
					Message: "Disallowed builder",
				},
			}
			return nil

		}
	}

	review.Response = &admissionv1.AdmissionResponse{
		UID:     review.Request.UID,
		Allowed: true,
		Result: &metav1.Status{
			Message: "Welcome to the Toolforge!",
		},
	}
	return nil
}
