# TODO switch the default image to rhel9 and drop the rhel8 one in 4.15 because
# we can require by the time we get to 4.14 that we don't have any rhel8 hosts left
FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.20-openshift-4.15 AS builder
ARG TAGS=""
WORKDIR /go/src/github.com/openshift/machine-config-operator
COPY . .
# FIXME once we can depend on a new enough host that supports globs for COPY,
# just use that.  For now we work around this by copying a tarball.
RUN make install DESTDIR=./instroot && tar -C instroot -cf instroot.tar .

FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.20-openshift-4.15 AS rhel9-builder
ARG TAGS=""
WORKDIR /go/src/github.com/openshift/machine-config-operator
COPY . .
RUN make install DESTDIR=./instroot

FROM registry.ci.openshift.org/ocp/4.15:base
ARG TAGS=""
COPY --from=builder /go/src/github.com/openshift/machine-config-operator/instroot.tar /tmp/instroot.tar
RUN cd / && tar xf /tmp/instroot.tar && rm -f /tmp/instroot.tar
COPY --from=rhel9-builder /go/src/github.com/openshift/machine-config-operator/instroot/usr/bin/machine-config-daemon /usr/bin/machine-config-daemon.rhel9
COPY install /manifests
# API is now migrated including controller, so copy our CRDs straight from /vendor 
COPY vendor/github.com/openshift/api/machineconfiguration/v1/*.crd.yaml /manifests

RUN if [ "${TAGS}" = "fcos" ]; then \
    # comment out non-base/extensions image-references entirely for fcos
    sed -i '/- name: rhel-coreos-/,+3 s/^/#/' /manifests/image-references && \
    # also remove extensions from the osimageurl configmap (if we don't, oc won't rewrite it, and the placeholder value will survive and get used)
    sed -i '/baseOSExtensionsContainerImage:/ s/^/#/' /manifests/0000_80_machine-config-operator_05_osimageurl.yaml && \
    # rewrite image names for fcos
    sed -i 's/rhel-coreos/fedora-coreos/g' /manifests/*; \
    elif [ "${TAGS}" = "scos" ]; then \
    # rewrite image names for scos
    sed -i 's/rhel-coreos/centos-stream-coreos-9/g' /manifests/*; fi && \
    # pin nmstate to 2.2.9 until we update to https://github.com/openshift/machine-config-operator/pull/3720
    if ! rpm -q util-linux; then dnf install -y util-linux; fi && dnf -y install nmstate-2.2.9-6.rhaos4.14.el8 && dnf clean all && rm -rf /var/cache/dnf/*
COPY templates /etc/mcc/templates
ENTRYPOINT ["/usr/bin/machine-config-operator"]
LABEL io.openshift.release.operator true
