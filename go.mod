module gerrit.wikimedia.org/cloud/tools/buildpack-admission-webhook

go 1.15

require (
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sirupsen/logrus v1.8.1
	github.com/tektoncd/pipeline v0.27.1
	k8s.io/api v0.20.7
	k8s.io/apimachinery v0.20.7
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect

)

exclude github.com/go-logr/logr v1.0.0
