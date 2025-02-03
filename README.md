# KubeRecovery
<img src="https://raw.githubusercontent.com/freepik-company/kuberecovery/master/docs/img/logo.png" alt="KubeRecovery Logo (Main) logo." width="150">

![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/freepik-company/kuberecovery)
![GitHub](https://img.shields.io/github/license/freepik-company/kuberecovery)

KubeRecovery is a Kubernetes operator that helps in recovering the resources that are deleted accidentally.
Like a recycle bin, it provides a way to recover the resources that are deleted by mistake. It is a simple 
operator that watches for the resources that are deleted and provides a way to recover them.

## Motivation

Has it ever happened to you that while using `k9s`, you accidentally pressed Ctrl+D on the only secret in the cluster that
wasn’t versioned, and no matter how many times you hit Ctrl+Z, you couldn’t recover it? Or that you deleted a deployment
and have no idea which image was running at that moment?

Well, KubeRecovery is your solution. With KubeRecovery, you can recover mistakenly deleted resources as if they were in
a recycle bin.

## How it works

KubeRecovery is based on just two components:
* **RecoveryConfig**: This resource is responsible for defining which cluster resources you want to audit and recover 
in case of a major disaster.
```yaml
apiVersion: kuberecovery.freepik.com/v1alpha1
kind: RecoveryConfig
metadata:
  name: recoveryconfig-sample
spec:

  # Resources to watch and save as RecoveryResource object when they are deleted
  # apiVersion and resources * is not supported, use specific version and resource instead
  # For namespaces use "*" to watch all namespaces
  resourcesIncluded:
    - apiVersion: "apps/v1"
      resources: ["deployments"]
      namespaces: ["*"]
    - apiVersion: "v1"
      resources: ["services"]
      namespaces: ["*"]

  # Resources to exclude from watching and saving as RecoveryResource object
  # apiVersion * is not supported, use specific apiVersion instead
  # Namespaces, names and resources regexp are supported, so you can define * to exclude all resources
  # of a resource or namespace or defau* to exclude all resources that start with defau
  resourcesExcluded:
    - apiVersion: "v1"
      resources: ["services"]
      namespaces: ["kube-system"]
      names: ["*"]

  # Retention period for RecoveryResource objects
  # Just support us, ns, ms, s, m, h and d as time units
  retention:
    period: 240h
```

* **RecoveryResource**: This resource is created when a resource is deleted. It contains all the necessary information 
to restore the deleted resource.
```yaml
apiVersion: kuberecovery.freepik.com/v1alpha1
kind: RecoveryResource
metadata:
  finalizers:
  - kuberecovery.freepik.com/finalizer
  # This finalizer is used to protect the resource from being deleted before the retentionUntil label date.
  - kuberecovery.freepik.com/protectFinalizer
  labels:
    kuberecovery.freepik.com/recoveryConfig: recoveryconfig-sample
    # This label is used to know when the resource is going to be deleted.
    kuberecovery.freepik.com/retentionUntil: 2025-02-09T151001
    # This label is used to know when the resource was saved at.
    kuberecovery.freepik.com/savedAt: 2025-01-30T151001
  name: recoveryconfig-sample-service-test-20250130151001
spec:
  <resource-deleted>
```
If any RecoveryResource is tagged with `kuberecovery.freepik.com/recover` and set to `"true"`, the deleted resource will 
be automatically restored.

## Deployment
We recommend to deploy KubeRecovery operator with our [Helm registry](https://freepik-company.github.io/kuberecovery/).

## How to collaborate

This project is done on top of [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder), so read about that project
before collaborating. Of course, we are open to external collaborations for this project. For doing it you must fork the
repository, make your changes to the code and open a PR. The code will be reviewed and tested (always)

> We are developers and hate bad code. For that reason we ask you the highest quality on each line of code to improve
> this project on each iteration.

## Contributors
🧑🏽‍🦱[@dfradehubs](https://github.com/dfradehubs) - Daniel Fradejas

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

