# Buildpack Pipeline Validation Webhook for Toolforge

This is a [Kubernetes Admission Validation Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks) deployed to check that
users are not setting ingress values that could interfere with other users.

## Use and development

This is pending adaption to Toolforge.  Currently it depends on local docker images and it
can be built and deployed on Kubernetes by insuring any node it is expected to run on
has access to the image it uses.  The image will need to be in a registry most likely when deployed.

It was developed using [Go Modules](https://github.com/golang/go/wiki/Modules), which will
validate the hash of every imported library during build.  At this time, it depends on
these external go libraries:

	* github.com/kelseyhightower/envconfig
	* github.com/sirupsen/logrus
	* k8s.io/api
	* k8s.io/apimachinery

To build on minikube and launch, follow these steps:

* `eval $(minikube docker-env)`
* `docker build -t buildpack-admission:latest .`

That creates the image on minikube's docker daemon.  Then to launch the service:

NOTE: Before you run ca-bundle.sh on MacOS, read the comments in that file and adjust accordingly

* `./get-cert.sh`  <-- creates a CSR and a secret with the TLS cert and key
* `./ca-bundle.sh` <-- places the correct ca-bundle in the service.yaml file
* `kubectl apply -f service.yaml`

As long as a suitable image can be placed where needed on toolforge, which can be done locally if
node affinity is used or some similar mechanism to prevent it being needed on every
spun-up node, the last three steps are likely all that is needed to bootstrap.

## Testing

At the top level, run `go test ./...` to capture all tests.  If you need to see output
or want to examine things more, use `go test -test.v ./...`

## Deploying

Since this was designed for use in [Toolforge](https://wikitech.wikimedia.org/wiki/Portal:Toolforge "Toolforge Portal"), so the instructions here focus on that.

The version of docker on the builder host is very old, so the builder/scratch pattern in
the Dockerfile won't work.

* Build the container on the docker-builder host (currently tools-docker-imagebuilder-01.tools.eqiad1.wikimedia.cloud). `$ docker build . -f deployments/Dockerfile -t docker-registry.tools.wmflabs.org/buildpack-admission:latest`
* Push the image to the internal repo: `root@tools-docker-imagebuilder-01:~# docker push docker-registry.tools.wmflabs.org/buildpack-admission:latest`
* The caBundle should be set correctly in a [kustomize](https://kustomize.io/) folder. You should now just be able to run `root@tools-k8s-control-1:# kubectl -k deployments/toolforge` to deploy to tools and `root@toolsbeta-test-k8s-control-1:# kubectl -k deployments/toolsbeta` to make the deployment work.

## Updating the certs

Certificates created with the Kubernetes API are valid for one year. When upgrading Kubernetes (or whenever necessary)
it is wise to rotate the certs for this service. To do so simply run (as cluster admin or root@control host) `root@tools-k8s-control-1:# ./get-cert.sh`. That will recreate the cert secret. Then delete the existing pods to ensure
that the golang web services are serving the new cert.
