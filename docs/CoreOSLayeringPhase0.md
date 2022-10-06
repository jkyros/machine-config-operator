# Using OCP CoreOS Layering Phase 0

In service to our [OCP CoreOS Layering Enhancement](https://github.com/openshift/enhancements/blob/master/enhancements/ocp-coreos-layering/ocp-coreos-layering.md#phase-0). 

While layering is really cool, it's also an "advanced" use of the MCO, and it comes with some trade-offs.  

> WARNING: once you do this you are effectively "taking the wheel" when it comes to managing the OS image during upgrades. This means that the `machine-config-operator` will just skip the part of the upgrade where it upgrades the OS image. 

>  WARNING: It is HIGHLY RECOMMENDED that you specify an `osImageURL` that is an image digest (`foo@sha256:1234...`) rather than an image tag (`foo:latest`), otherwise your nodes may end up on different images. 
> The MCO will let you use a tag as an `osImageURL`, but each Node's  `machine-config-daemon` pulls the image for itself -- which means that if the image the tag points at were to change after the first node were updated

> WARNING: Downgrading to "older" images will work in many cases, but we mostly test going forwards not backwards. If you go too far back you can end up with weird happenings like [this bug](https://mail.google.com/mail/u/0/#inbox/FMfcgzGqQSMglmXtGjNQgLrvpZnJrgCT)  when something like the kubelet gets backrev'd too far. 

> WARNING: At current, `MachineConfig` takes precedence over config files included in a derived image. Conflicts will not break anything, but right now `MachineConfig` always wins and will just overwrite the change. 

How it works right now is: 
1. Get your base image
2. Derive an image from it (Dockerfile "FROM" base image, install some packages, files, whatever)
3. Push your derived image somewhere where it can be pulled
4. Override `osImageURL` in a `MachineConfig` with that image 
5. Wait for the MCO to roll it out 

For the time being, this is very self-service, but as we progress through the phases of "OCP CoreOS Layering", this should get easier to use. 



## 1. Get  Your Base Image
Nothing will stop you at this point from using a completely arbitrary image, but if you want to succeed, you should derive your layered image from the base image matching your cluster. 

> NOTE: at some point we intend these images to be published to a predictable place for you to easily retrieve, but for now you will want to pull the image out of the release payload matching your cluster

### 1b. If You Don't Have A Cluster Yet
`oc adm release info --image-for rhel-coreos-8 quay.io/openshift-release-dev/ocp-release:4.12.0-ec.3-x86_64`

> NOTE:  use the release you intend to build your cluster off of, not the one listed here


### 1a. If You Have A Cluster Already

You can grab it from the `clusterVersion` object: 
`oc get clusterversion -o jsonpath='{.items[].status.desired.image}{"\n"}''` 

Or you can grab it from `controllerConfig`: 
`oc get controllerconfig -o jsonpath='{.items[].spec.baseOperatingSystemContainer}`'

```
[jkyros@jkyros-t590 cluster]$ oc get clusterversion -o jsonpath='{.items[].status.desired.image}{"\n"}'
quay.io/jkyros/ocp-release@sha256:5677c8143bfd2a089faca1e52237a22bfb289d9c03fbc2c79651018a64a44a61
```


## 2. Derive Your Image 
There are a lot of things you can do here, I suggest using https://github.com/coreos/coreos-layering-examples as a guideline for now, but obviously you'll need to use RPM packages that match your OS and version  (RHCOS, SCOS, etc). 

>NOTE: To use entitled packages in these derived builds,the standard entitlement guidance in https://docs.openshift.com/container-platform/4.11/cicd/builds/running-entitled-builds.html does apply, but IT WILL INCLUDE YOUR ENTITLEMENT CREDENTIALS IN THE IMAGE unless you remove them and build the image with --squash

For now, the standard `rpm-ostree` rules apply, so if you install packages that do things like install into /opt/ things might not work perfectly. 

```
FROM quay.io/jkyros/ocp-release@sha256:5677c8143bfd2a089faca1e52237a22bfb289d9c03fbc2c79651018a64a44a61
RUN rpm-ostree cliwrap install-to-root /
RUN rpm-ostree override replace http://download.eng.bos.redhat.com/brewroot/vol/rhel-8/packages/kernel/4.18.0/372.24.1.el8_6/x86_64/kernel-4.18.0-372.24.1.el8_6.x86_64.rpm \
http://download.eng.bos.redhat.com/brewroot/vol/rhel-8/packages/kernel/4.18.0/372.24.1.el8_6/x86_64/kernel-core-4.18.0-372.24.1.el8_6.x86_64.rpm \
http://download.eng.bos.redhat.com/brewroot/vol/rhel-8/packages/kernel/4.18.0/372.24.1.el8_6/x86_64/kernel-modules-4.18.0-372.24.1.el8_6.x86_64.rpm \ 
http://download.eng.bos.redhat.com/brewroot/vol/rhel-8/packages/kernel/4.18.0/372.24.1.el8_6/x86_64/kernel-modules-extra-4.18.0-372.24.1.el8_6.x86_64.rpm && rpm-ostree cleanup -m 
```

> WARNING: Take care that you run your cleanup as part of your command so the temp files/caches don't end up in your final image 

>NOTE: You can 

```
podman build -t localhost/derive-test-1 . 
```

## 3. Push Your Image

`podman push localhost/derive-test-1 quay.io/jkyros/derived-images:derive-test-1`


## 4. Override OSImageURL 
```
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
  name: 99-external-image-worker
spec:
  osImageURL: "quay.io/jkyros/derived-images:derive-test-1"
```


## FAQ: 

#### 1. Can I override OSImageURL during cluster build? 
Yes, you can.  If you supply a `MachineConfig` containing an overridden `OSImageURL`, the cluster will build and use it. 
