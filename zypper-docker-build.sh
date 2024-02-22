#!/bin/bash
#
# MIT License
#
# (C) Copyright 2024 Hewlett Packard Enterprise Development LP
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

# This script is called during the Docker image build.
# It isolates the zypper operations, some of which require artifactory authentication,
# and scrubs the zypper environment after the necessary operations are completed.

# Preconditions:
# 1. Following variables have been set in the Dockerfile: SP ARCH
# 2. zypper-refresh-patch-clean.sh script has also been copied into the current directory

# Based on the script of the same name in the csm-config repo

set -e +xv
trap "rm -rf /root/.zypp" EXIT

# Get artifactory credentials and use them to set the csm-rpms stable sles15sp$SP repository URI
ARTIFACTORY_USERNAME=$(test -f /run/secrets/ARTIFACTORY_READONLY_USER && cat /run/secrets/ARTIFACTORY_READONLY_USER)
ARTIFACTORY_PASSWORD=$(test -f /run/secrets/ARTIFACTORY_READONLY_TOKEN && cat /run/secrets/ARTIFACTORY_READONLY_TOKEN)
CREDS=${ARTIFACTORY_USERNAME:-}
# Append ":<password>" to credentials variable, if a password is set
[[ -z ${ARTIFACTORY_PASSWORD} ]] || CREDS="${CREDS}:${ARTIFACTORY_PASSWORD}"
SLES_MIRROR_URL="https://${CREDS}@artifactory.algol60.net/artifactory/sles-mirror"
SLES_PRODUCTS_URL="${SLES_MIRROR_URL}/Products"
SLES_UPDATES_URL="${SLES_MIRROR_URL}/Updates"

zypper --non-interactive rr --all
zypper --non-interactive clean -a
zypper --non-interactive ar "${SLES_PRODUCTS_URL}/SLE-Module-Basesystem/15-SP4/${ARCH}/product/" "sles15sp4-Module-Basesystem-product"
zypper --non-interactive ar "${SLES_UPDATES_URL}/SLE-Module-Basesystem/15-SP4/${ARCH}/update/" "sles15sp4-Module-Basesystem-update"
zypper --non-interactive ar "${SLES_PRODUCTS_URL}/SLE-Module-Development-Tools/15-SP4/${ARCH}/product/" "sles15sp4-Module-Development-Tools-product"
zypper --non-interactive ar "${SLES_UPDATES_URL}/SLE-Module-Development-Tools/15-SP4/${ARCH}/update/" "sles15sp4-Module-Development-Tools-update"
zypper --non-interactive --gpg-auto-import-keys refresh
zypper --non-interactive in -f --no-confirm go1.19
# Apply security patches (this script also does a zypper clean)
./zypper-refresh-patch-clean.sh
# Remove all repos & scrub the zypper directory
zypper --non-interactive rr --all
rm -f /etc/zypp/repos.d/*
