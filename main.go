package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/toolforge/buildpack-admission-controller/pkg/server"
)

// Config is the general configuration of the webhook via env variables
type Config struct {
	ListenOn        string   `default:"0.0.0.0:8080"`
	TLSCert         string   `default:"/etc/webhook/certs/tls.crt"`
	TLSKey          string   `default:"/etc/webhook/certs/tls.key"`
	AllowedDomains  []string `default:"tools-harbor.wmcloud.org,toolsbeta-harbor.wmcloud.org"`
	SystemUsers     []string `default:"system:serviceaccount:tekton-pipelines:tekton-pipelines-controller"`
	AllowedBuilders []string `default:"paketobuildpacks/builder:base,gcr.io/buildpacks/builder:v1,docker-registry.tools.wmflabs.org/toolforge-bullseye0-builder:latest"`
	Debug           bool     `default:"true"`
}

func main() {
	config := &Config{}
	err := envconfig.Process("", config)

	if config.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err != nil {
		logrus.Fatalln(err)
	}

	logrus.Infoln(config)
	prac := server.PipelineRunAdmission{AllowedDomains: config.AllowedDomains, AllowedBuilders: config.AllowedBuilders, SystemUsers: config.SystemUsers}
	s := server.GetAdmissionValidationServer(&prac, config.TLSCert, config.TLSKey, config.ListenOn)
	err = s.ListenAndServeTLS("", "")
	if err != nil {
		logrus.Fatalln(err)
	}
}
