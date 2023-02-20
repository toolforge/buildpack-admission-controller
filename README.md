# Buildpack Pipeline Validation Webhook for Toolforge

This is a [Kubernetes Admission Validation Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks) deployed to check that
users are not setting ingress values that could interfere with other users.

## Use and development

This is pending adaption to Toolforge.  Currently it depends on local docker images and it
can be built and deployed on Kubernetes by ensuring any node it is expected to run on
has access to the image it uses.  The image will need to be in a registry most likely when deployed.

**Export your local repository ip**

The IP to use will vary depending on your machine and OS,
This is usually:

- `export DEV_DOMAIN_IP=$(minikube ssh "grep host.minikube.internal /etc/hosts" | awk '{print $1}')`

But sometimes (ex. debian bullseye) you will need to use the external IP of your host:

- `export DEV_DOMAIN_IP=$(hostname -I| awk '{print $1}')`

**Build on minikube**

To build on minikube (current supported k8s version is 1.21) and launch, just run:
  * (Launch a cluster if you have not already done so.)
  * `eval $(minikube docker-env)`
  * `docker build -f Dockerfile -t buildpack-admission:latest .`
  * `./deploy.sh local`

After you've made changes, update the image and restart the running container:
  * `eval $(minikube docker-env)` (if you did not already do this in the current session)
  * `docker build -f Dockerfile -t buildpack-admission:latest .`
  * `kubectl rollout restart -n buildpack-admission deployment buildpack-admission`

**Making changes**
Before you make a change, you need setup pre-commit on your local machine.

* run `pip3 install pre-commit` on your local machine (`brew install pre-commit` if using homebrew)
* run `pre-commit install` to setup the git hook scripts.

After the above steps, you can go ahead and make changes, commit and push.
## Testing

At the top level, run `go test ./...` to capture all tests.  If you need to see output
or want to examine things more, use `go test -test.v ./...`

## Deploying To Production

NOTE: this might change soon, once https://phabricator.wikimedia.org/T291915 is resolved

Since this was designed for use in [Toolforge](https://wikitech.wikimedia.org/wiki/Portal:Toolforge "Toolforge Portal"), so the instructions here focus on that.

The version of docker on the builder host is very old, so the builder/scratch pattern in
the Dockerfile won't work.

* Build the container on the docker-builder host (currently tools-docker-imagebuilder-01.tools.eqiad1.wikimedia.cloud)
and push image to the internal repo:

  with a checkout of the repo somewhere in the docker image builder host (in a home directory is probably great), run:

    `root@tools-docker-imagebuilder-01:# ./deploy.sh -b <tools or toolsbeta>`
  The command above builds the image on the image builder host and pushes it to the internal docker registry.

* The caBundle should be set correctly in a [kustomize](https://kustomize.io/) folder. You should now just be able to run the command below on the k8s control node as root (or as a cluster-admin user), with a checkout of the repo there somewhere (in a home directory is probably great):

    `myuser@tools-k8s-control-1:# ./deploy.sh <tools or toolsbeta>`

  to deploy to toolforge or toolsbeta.

## Updating the certs

Certificates created with the Kubernetes API are valid for one year. When upgrading Kubernetes (or whenever necessary)
it is wise to rotate the certs for this service. To do so simply run (as cluster admin or root@control host), with a checkout of the repo there somewhere:

`root@tools-k8s-control-1:# ./deploy.sh -c <tools or toolsbeta>`

Or any time by simply running (as cluster admin or root@control host)
`root@tools-k8s-control-1:# ./utils/regenerate_certs.sh`. That will recreate the cert secret. Then delete the existing pods to ensure
that the golang web services are serving the new cert.
