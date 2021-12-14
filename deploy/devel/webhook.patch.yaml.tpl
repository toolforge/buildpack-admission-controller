- op: replace
  path: /webhooks/0/name
  value: buildpack-admission.buildpack-admission.svc.local
- op: replace
  path: /webhooks/0/clientConfig/caBundle
  value: @@CA_BUNDLE@@
