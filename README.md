# Summary

Lighthouse is a framework to pre-hook/post-hook runtime request/response. With this hook, we can dynamically add options to any other OCI
 arguments which aren't supported in Kubernetes.

# Architecture

![LighthouseDesign.svg](doc/LighthouseDesign.svg)

# Hook Configuration example

```
apiVersion: lighthouse.io/v1alpha1
kind: hookConfiguration
timeout: 10
# This field is for kubelet --docker-endpoint
listenAddress: unix:///var/run/lighthouse.sock
webhooks:
- name: lighthouse.io
  endpoint: unix://@lighthouse-hook
  failurePolicy: Fail
  stages:
  - urlPattern: /containers/create
    method: post
    type: PreHook
  - urlPattern: /containers/create
    method: post
    type: PostHook
```

# How to use it in Kubernetes

Set kubelet options `--docker-endpoint` to the field of `listenAddress` in your hook configuration