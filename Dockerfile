#
# MIT License
#
# (C) Copyright 2020-2022, 2024 Hewlett Packard Enterprise Development LP
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

# NOTE: to update to SP6, it needs to update to a newer version of 'Go', which
#  will take a bit of work.

# Build will be where we build the go binary
FROM artifactory.algol60.net/csm-docker/stable/registry.suse.com/suse/sle15:15.5 as build

ARG SP=5
ARG ARCH=x86_64

# Do zypper operations using a wrapper script, to isolate the necessary artifactory authentication
COPY zypper-docker-build.sh /
# The above script calls the following script, so we need to copy it as well
COPY zypper-refresh-patch-clean.sh /
RUN --mount=type=secret,id=ARTIFACTORY_READONLY_USER --mount=type=secret,id=ARTIFACTORY_READONLY_TOKEN \
    ./zypper-docker-build.sh go1.19 && \
    rm /zypper-docker-build.sh /zypper-refresh-patch-clean.sh

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
#FROM artifactory.algol60.net/csm-docker/stable/registry.suse.com/suse/sle15:15.5 as base
#ARG SLES_MIRROR=https://slemaster.us.cray.com/SUSE
#ARG ARCH=x86_64
#RUN set -eux \
#    && zypper --non-interactive rr --all \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-Basesystem/15-SP5/${ARCH}/product/ sles15sp5-Module-Basesystem-product \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-Basesystem/15-SP5/${ARCH}/update/ sles15sp5-Module-Basesystem-update \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Products/SLE-Module-HPC/15-SP5/${ARCH}/product/ sles15sp5-Module-HPC-product \
#    && zypper --non-interactive ar ${SLES_MIRROR}/Updates/SLE-Module-HPC/15-SP5/${ARCH}/update/ sles15sp5-Module-HPC-update \
#    && zypper --non-interactive install conman less vi openssh jq curl tar

### Final Stage ###
# Start with a fresh image so build tools are not included
FROM arti.hpc.amslabs.hpecorp.net/baseos-docker-master-local/sles15sp5:sles15sp5 as base

ARG SP=5
ARG ARCH=x86_64

# Do zypper operations using a wrapper script, to isolate the necessary artifactory authentication
COPY zypper-docker-build.sh /
# The above script calls the following script, so we need to copy it as well
COPY zypper-refresh-patch-clean.sh /
RUN --mount=type=secret,id=ARTIFACTORY_READONLY_USER --mount=type=secret,id=ARTIFACTORY_READONLY_TOKEN \
    ./zypper-docker-build.sh conman less vi openssh jq curl tar procps inotify-tools --remove polkit && \
    rm /zypper-docker-build.sh /zypper-refresh-patch-clean.sh

# Copy in the needed files
COPY --from=build /app/console_node /app/
COPY scripts/conman.conf /app/conman_base.conf
COPY scripts/ssh-key-console /usr/bin
COPY scripts/ssh-pwd-console /usr/bin

# Change ownership of the app dir and switch to user 'nobody'
RUN chown -Rv 65534:65534 /app /etc/conman.conf
USER 65534:65534

# Environment Variables -- Used by the HMS secure storage pkg
ENV VAULT_ADDR="http://cray-vault.vault:8200"
ENV VAULT_SKIP_VERIFY="true"

RUN echo 'alias ll="ls -l"' > /app/bashrc
RUN echo 'alias vi="vim"' >> /app/bashrc

ENTRYPOINT ["/app/console_node"]
