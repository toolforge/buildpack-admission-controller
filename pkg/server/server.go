package server

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/json"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

// AdmissionController is an abstraction to work with the admission handler
type AdmissionController interface {
	HandleAdmission(review *admissionv1.AdmissionReview) error
}

// AdmissionControllerServer combines a decoder with an AdmissionController
type AdmissionControllerServer struct {
	AdmissionController AdmissionController
	Decoder             runtime.Decoder
}

func (acs *AdmissionControllerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if data, err := ioutil.ReadAll(r.Body); err == nil {
		body = data
	}
	logrus.Debugln(string(body))
	review := &admissionv1.AdmissionReview{}
	_, _, err := acs.Decoder.Decode(body, nil, review)
	if err != nil {
		logrus.Errorln("Can't decode request", err)
	}
	acs.AdmissionController.HandleAdmission(review)
	responseInBytes, err := json.Marshal(review)
	if err != nil {
		logrus.Errorln("Can't decode review", err)
	}
	if _, err := w.Write(responseInBytes); err != nil {
		logrus.Errorln(err)
	}
}

// GetAdmissionServerNoSSL is a way to allows very simple testing without
// certs getting in the way.
func GetAdmissionServerNoSSL(ac AdmissionController, listenOn string) *http.Server {
	server := &http.Server{
		Handler: &AdmissionControllerServer{
			AdmissionController: ac,
			Decoder:             codecs.UniversalDeserializer(),
		},
		Addr: listenOn,
	}

	return server
}

//GetAdmissionValidationServer is a constructor for producing a working TLS-enabled webhook
func GetAdmissionValidationServer(ac AdmissionController, tlsCert, tlsKey, listenOn string) *http.Server {
	sCert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
	server := GetAdmissionServerNoSSL(ac, listenOn)
	server.TLSConfig = &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
	if err != nil {
		logrus.Error(err)
	}
	return server
}
