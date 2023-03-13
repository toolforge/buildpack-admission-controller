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

When deploying you have to take into account two things:

* The code of the controller (any `*.go` files)
* The code and config for the deployment (the files under `deployment/*`) -> you only need to run the deploy

### If you changed any controller code (usually `*.go` file)

You will have to build a new docker image, and send a new commit updating the deploy code to use the new image tag.

To build the image you can use the [cookbook](https://wikitech.wikimedia.org/wiki/Portal:Toolforge/Admin/Kubernetes/Components#Build):

> cookbook wmcs.toolforge.k8s.component.build --git-url https://github.com/toolforge/buildpack-admission-controller


That will give you an image tag:
```
dcaro@vulcanus$ cookbook wmcs.toolforge.k8s.component.build --git-url https://github.com/toolforge/buildpack-admission-controller
...
[DOLOGMSG]: build & push docker image docker-registry.tools.wmflabs.org/toolforge-buildpack-admission-controller:f90bd8f from https://github.com/toolforge/buildpack-admission-controller (f90bd8f)
END (PASS) - Cookbook wmcs.toolforge.k8s.component.build (exit_code=0)
```

And now you have to create a new commit setting that tag (`f90bd8f`) in the `deployment/values/tools.yaml` and `deployment/values/toolsbeta.yaml` files, send that for review and get it merged.

Once merged, you can deploy (see the next step).


### Deploy the new image and/or deployment code


To deploy you can use [cookbook](https://wikitech.wikimedia.org/wiki/Portal:Toolforge/Admin/Kubernetes/Components#Deploy):

```
cookbook wmcs.toolforge.k8s.component.deploy \
    --git-url https://github.com/toolforge/buildpack-admission-controller \
    --cluster-name toolsbeta
```
