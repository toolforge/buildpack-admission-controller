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
	AllowedDomains  []string
	SystemUsers     []string
	AllowedBuilders []string
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

	domstr := strings.Join(p.AllowedDomains, "|")
	buildstr := strings.Join(p.AllowedBuilders, "|")
	builderRe := regexp.MustCompile(fmt.Sprintf("^%s$", buildstr))
	user := req.UserInfo.Username
	acceptedUser := false
	logrus.Debugf("Validating user")
	for _, au := range p.SystemUsers {
		logrus.Debugf("checking against	%v user %v", au, user)
		if au == user {
			logrus.Debugf("Found allowed user, accepting (allowed one %v): %v", au, user)
			acceptedUser = true
			break
		}
	}
	for _, param := range pipelinerun.Spec.Params {
		expectedURL := fmt.Sprintf("^(https?://)?%s/%s/", domstr, user)
		harborRe := regexp.MustCompile(expectedURL)
		logrus.Debugf("Found PipeLineRun param: %v", param.Name)
		switch param.Name {
		case "APP_IMAGE":
			logrus.Debugf("Validating APP_IMAGE")
			if acceptedUser || harborRe.MatchString(param.Value.StringVal) {
				logrus.Debugf("APP_IMAGE: ok")
				continue
			}
			logrus.Debugf("APP_IMAGE: not ok")
			review.Response = &admissionv1.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: false,
				Result: &metav1.Status{
					Message: fmt.Sprintf(
						"Harbor domain (gotten %s) not matching AllowedDomains (expected %s) nor the user (gotten %s) matches any of the "+
							"system users (expected %s).",
						param.Value.StringVal,
						expectedURL,
						user,
						p.SystemUsers),
				},
			}
			return nil
		case "BUILDER_IMAGE":
			logrus.Debugf("Validating BUILDER_IMAGE")
			logrus.Debugf("Found builder: %v", param.Value.StringVal)
			logrus.Debugf("builder: %v\n", param.Value.StringVal)
			if builderRe.MatchString(param.Value.StringVal) {
				logrus.Debugf("BUILDER_IMAGE: ok")
				continue
			}
			logrus.Debugf("BUILDER_IMAGE: not ok")
			review.Response = &admissionv1.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: false,
				Result: &metav1.Status{
					Message: fmt.Sprintf("Builder (gotten %s) does not match AllowedBuilders (expected %s).", param.Value.StringVal, buildstr),
				},
			}
			return nil
		case "SOURCE_URL":
			// permitted parameter, but not validated
		case "USER_ID":
			// permitted parameter, but not validated (TODO: should it?)
		case "GROUP_ID":
			// permitted parameter, but not validated (TODO: should it?)
		case "SOURCE_REFERENCE":
			// permitted parameter, but not validated (TODO: should it?)
		default:
			// Since (at least as of writing this) we use pipelines created by the upstream Tekton
			// project, we want to ensure that only parameters that we've ensured to be safe can be
			// used.
			logrus.Debugf("Pipeline parameter %v is not allowed", param.Name)

			review.Response = &admissionv1.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: false,
				Result: &metav1.Status{
					Message: fmt.Sprintf("Pipeline parameter %v cannot be used", param.Name),
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
