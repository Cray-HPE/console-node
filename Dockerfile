#
# MIT License
#
# (C) Copyright 2020-2022 Hewlett Packard Enterprise Development LP
#
# Permission is hereby granted, free of charge, to any person obtaining a
# copy of this software and associated documentation files (the "Software"),
# to deal in the Software without restriction, including without limitation
# the rights to use, copy, modify, merge, publish, distribute, sublicense,
# and/or sell copies of the Software, and to permit persons to whom the
# Software is furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included
# in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.
#
# Dockerfile for cray-console-node service

# Build will be where we build the go binary
FROM artifactory.algol60.net/csm-docker/stable/registry.suse.com/suse/sle15:15.4 as build

# The current sles15sp4 base image starts with a lock on coreutils, but this prevents a necessary
# security patch from being applied. Thus, adding this command to remove the lock if it is
# present.
RUN zypper --non-interactive removelock coreutils || true

ARG SLES_MIRROR=https://slemaster.us.cray.com/SUSE
ARG ARCH=x86_64
RUN set -eux \
  && zypper --non-interactive rr --all \
  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Basesystem/15-SP4/${ARCH}/product/ sles15sp4-Module-Basesystem-product \
  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Basesystem/15-SP4/${ARCH}/update/ sles15sp4-Module-Basesystem-update \
  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Development-Tools/15-SP4/${ARCH}/product/ sles15sp4-Module-Development-Tools-product \
  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Development-Tools/15-SP4/${ARCH}/update/ sles15sp4-Module-Development-Tools-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Containers/15-SP4/${ARCH}/product/ sles15sp4-Module-Containers-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Containers/15-SP4/${ARCH}/update/ sles15sp4-Module-Containers-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Desktop-Applications/15-SP4/${ARCH}/product/ sles15sp4-Module-Desktop-Applications-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Desktop-Applications/15-SP4/${ARCH}/update/ sles15sp4-Module-Desktop-Applications-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-HPC/15-SP4/${ARCH}/product/ sles15sp4-Module-HPC-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-HPC/15-SP4/${ARCH}/update/ sles15sp4-Module-HPC-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Legacy/15-SP4/${ARCH}/product/ sles15sp4-Module-Legacy-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Legacy/15-SP4/${ARCH}/update/ sles15sp4-Module-Legacy-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Public-Cloud/15-SP4/${ARCH}/product/ sles15sp4-Module-Public-Cloud-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Public-Cloud/15-SP4/${ARCH}/update/ sles15sp4-Module-Public-Cloud-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Python2/15-SP4/${ARCH}/product/ sles15sp4-Module-Python2-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Python2/15-SP4/${ARCH}/update/ sles15sp4-Module-Python2-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Server-Applications/15-SP4/${ARCH}/product/ sles15sp4-Module-Server-Applications-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Server-Applications/15-SP4/${ARCH}/update/ sles15sp4-Module-Server-Applications-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Web-Scripting/15-SP4/${ARCH}/product/ sles15sp4-Module-Web-Scripting-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Web-Scripting/15-SP4/${ARCH}/update/ sles15sp4-Module-Web-Scripting-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Product-SLES/15-SP4/${ARCH}/product/ sles15sp4-Product-SLES-product \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Product-SLES/15-SP4/${ARCH}/update/ sles15sp4-Product-SLES-update \
#  && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-INSTALLER/15-SP4/${ARCH}/update/ sles15sp4-SLE-INSTALLER-update \
  && zypper --non-interactive clean \
  && zypper --non-interactive install go1.19

# Apply security patches
COPY zypper-refresh-patch-clean.sh /
RUN /zypper-refresh-patch-clean.sh && rm /zypper-refresh-patch-clean.sh

# Configure go env - installed as package but not quite configured
ENV GOPATH=/usr/local/golib
RUN export GOPATH=$GOPATH

# Copy in all the necessary files
COPY src/console_node $GOPATH/src/console_node
COPY vendor/ $GOPATH/src

# Build configure_conman
RUN set -ex \
    && go env -w GO111MODULE=auto \
    && go build -v -i -o /app/console_node $GOPATH/src/console_node

# NOTE:
#  We need to switch to the below image, but for now it does not include the 'nobody' user
#  and we need to figure out why/how that user was removed from the image.
#FROM artifactory.algol60.net/csm-docker/stable/registry.suse.com/suse/sle15:15.4 as base
#ARG SLES_MIRROR=https://slemaster.us.cray.com/SUSE
#ARG ARCH=x86_64
#RUN set -eux \
#    && zypper --non-interactive rr --all \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Basesystem/15-SP4/${ARCH}/product/ sles15sp4-Module-Basesystem-product \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Basesystem/15-SP4/${ARCH}/update/ sles15sp4-Module-Basesystem-update \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-HPC/15-SP4/${ARCH}/product/ sles15sp4-Module-HPC-product \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-HPC/15-SP4/${ARCH}/update/ sles15sp4-Module-HPC-update \
#    && zypper --non-interactive install conman less vi openssh jq curl tar

### Final Stage ###
# Start with a fresh image so build tools are not included
FROM arti.hpc.amslabs.hpecorp.net/baseos-docker-master-local/sles15sp4:sles15sp4 as base

# The current sles15sp4 base image starts with a lock on coreutils, but this prevents a necessary
# security patch from being applied. Thus, adding this command to remove the lock if it is
# present.
RUN zypper --non-interactive removelock coreutils || true

# Install conman application from package
RUN set -eux \
    && zypper --non-interactive install conman less vi openssh jq curl tar

# NOTE: polkit is not needed but is included with one of the above packages.
#  It has frequent security issues so just remove it here.
RUN zypper --non-interactive rm polkit

# Apply security patches
COPY zypper-refresh-patch-clean.sh /
RUN /zypper-refresh-patch-clean.sh && rm /zypper-refresh-patch-clean.sh

# Copy in the needed files
COPY --from=build /app/console_node /app/
COPY scripts/conman.conf /app/conman_base.conf
COPY scripts/ssh-console /usr/bin

# Change ownership of the app dir and switch to user 'nobody'
RUN chown -Rv 65534:65534 /app /etc/conman.conf
USER 65534:65534

# Environment Variables -- Used by the HMS secure storage pkg
ENV VAULT_ADDR="http://cray-vault.vault:8200"
ENV VAULT_SKIP_VERIFY="true"

RUN echo 'alias ll="ls -l"' > /app/bashrc
RUN echo 'alias vi="vim"' >> /app/bashrc

ENTRYPOINT ["/app/console_node"]
