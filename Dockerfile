# Copyright 2020-2021 Hewlett Packard Enterprise Development LP
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
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL
# THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
# OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
# ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
# OTHER DEALINGS IN THE SOFTWARE.
#
# (MIT License)

# Dockerfile for cray-console-node service

# Build will be where we build the go binary
FROM arti.dev.cray.com/baseos-docker-master-local/sles15sp2:sles15sp2 as build
RUN set -eux \
    && zypper --non-interactive install go1.14

# Apply security patches
COPY zypper-refresh-patch-clean.sh /
RUN /zypper-refresh-patch-clean.sh && rm /zypper-refresh-patch-clean.sh

# Configure go env - installed as package but not quite configured
ENV GOPATH=/usr/local/golib
RUN export GOPATH=$GOPATH

# Copy in all the necessary files
COPY src/console_node $GOPATH/src/console_node

# NOTE: no vendor code yet - commented out to make build work,
#  add back in when vendor code is needed
COPY vendor/ $GOPATH/src

# Build configure_conman
RUN set -ex && go build -v -i -o /app/console_node $GOPATH/src/console_node

### Final Stage ###
# Start with a fresh image so build tools are not included
FROM arti.dev.cray.com/baseos-docker-master-local/sles15sp2:sles15sp2 as base

# Install conman application from package
RUN set -eux \
    && zypper --non-interactive install conman \
    && zypper --non-interactive install less \
    && zypper --non-interactive install --no-recommends adobe-sourcecodepro-fonts \
    && zypper --non-interactive install --no-recommends cantarell-fonts \
    && zypper --non-interactive install --no-recommends dracut \
    && zypper --non-interactive install --no-recommends gcr-viewer \
    && zypper --non-interactive install --no-recommends gtk3-branding-SLE \
    && zypper --non-interactive install --no-recommends gvfs-backends \
    && zypper --non-interactive install --no-recommends gvfs-fuse \
    && zypper --non-interactive install --no-recommends gvfs \
    && zypper --non-interactive install --no-recommends postfix \
    && zypper --non-interactive install --no-recommends sound-theme-freedesktop \
    && zypper --non-interactive install --no-recommends udisks2 \
    && zypper --non-interactive install --no-recommends udisks2-lang \
    && zypper --non-interactive install --no-recommends wallpaper-branding-SLE \
    && zypper --non-interactive install --no-recommends xkeyboard-config-lang \
    && zypper --non-interactive install vi \
    && zypper --non-interactive install openssh \
    && zypper --non-interactive install jq \
    && zypper --non-interactive install curl \
    && zypper --non-interactive install tar

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
