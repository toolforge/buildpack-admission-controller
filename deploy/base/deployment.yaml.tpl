
apiVersion: apps/v1
kind: Deployment
metadata:
  name: buildpack-admission
  namespace: buildpack-admission
  labels:
    name: buildpack-admission
spec:
  replicas: 2
  selector:
    matchLabels:
      name: buildpack-admission
  template:
    metadata:
      name: buildpack-admission
      labels:
        name: buildpack-admission
    spec:
      containers:
        - name: webhook
          image: buildpack-admission
          env:
            - name: "DEBUG"
              value: "false"
            - name: "ALLOWEDDOMAINS"
              value: "harbor.toolforge.org,harbor.toolsbeta.wmflabs.org"
            - name: "SYSTEMUSERS"
              value: "system:serviceaccount:tekton-pipelines:tekton-pipelines-controller"
            - name: "BUILDID"
              value: "@@BUILD_ID@@"
          resources:
            limits:
              memory: 50Mi
              cpu: 300m
            requests:
              memory: 50Mi
              cpu: 300m
          volumeMounts:
            - name: webhook-certs
              mountPath: /etc/webhook/certs
              readOnly: true
          securityContext:
            readOnlyRootFilesystem: true
      volumes:
        - name: webhook-certs
          secret:
            secretName: buildpack-admission-certs
