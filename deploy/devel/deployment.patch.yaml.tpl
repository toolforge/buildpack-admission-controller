- op: add
  # this avoids the local dev env from pulling the image from outside and using
  # the cached one (probably locally built)
  path: /spec/template/spec/containers/0/imagePullPolicy
  value: Never
- op: replace
  path: /spec/template/spec/containers/0/env/1/value
  # this resolves to the ip of the host running minikube
  value: host.minikube.internal,192.168.49.1
# Enable debugging
- op: replace
  path: /spec/template/spec/containers/0/env/0/value
  value: "true"
- op: replace
  path: /spec/template/spec/containers/0/env/3/value
  value: "@@BUILD_ID@@-dev"
