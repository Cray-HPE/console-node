# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.9.0] - 2025-05-05
### Added
- CASMCMS-9085 - Initial implementation of websocket api for console services.
- CASMCMS-9090 - use suse 1.5 sp6 base image from artifactory.algol60.net.

### Dependencies
- Update github.com/hashicorp/go-cleanhttp v0.5.1 to v0.5.2
- Update github.com/hashicorp/go-retryablehttp v0.5.4 to v0.7.7
- Update golang.org/x/crypto v0.31.0 to v0.36.0
- Update golang.org/x/net v0.21.0 to v0.38.0
- Update golang.org/x/sys v0.28.0 to  v0.31.0
- Update golang.org/x/text v0.21.0 to v0.23.0
- Update gopkg.in/square/go-jose.v2 v2.3.1 to v2.6.0

## [2.8.0] - 2025-02-03
### Fixed
- CASMTRIAGE-7715 - fix file permissions for log directories as well as files

### Dependencies
- CASMCMS-9266: Bump `golang.org/x/crypto` from 0.17.0 to 0.31.0 ([#110](https://github.com/Cray-HPE/console-node/pull/110))
- Bump `golang.org/x/crypto` from 0.17.0 to 0.31.0 ([#110](https://github.com/Cray-HPE/console-node/pull/110))

## [2.7.0] - 2024-11-22
### Fixed
- CASMTRIAGE-7594 - clean up resilience, rebalance nodes, and accept other worker nodes
- CASMCMS-9126 - watch permissions on log files to insure they can be written to

## [2.6.0] - 2024-11-22
### Fixed
- CASMCMS-9217: Close response bodies for GET requests; drain and close response bodies on error path, if needed

## [2.5.0] - 2024-10-15
### Added
- CASMCMS-8681 - add add inotify-tools to the base image.

## [2.4.0] - 2024-09-05
### Changed
- CASMCMS-9147 - stop using alpine:latest base image.
- CASMTRIAGE-7345 - fix inclusion of ps utility after upgrade to sp5.

### Dependencies
- CASMCMS-9136: Bump `cray-services` base chart to 11.0

## [2.3.0] - 2024-06-10
### Changed
- CASMCMS-9056 - update base image to sles15-sp5

## [2.2.1] - 2024-06-10
### Dependencies
- CASMCMS-9027: Bump `golang.org/x/crypto` from 0.0.0-20210616213533-5ff15b29337e to 0.17.0 ([#88](https://github.com/Cray-HPE/console-node/pull/88))

## [2.2.0] - 2024-05-03
### Added
- CASMCMS-8899 - add support for Paradise (xd224) nodes.

## [2.1.0] - 2024-02-22
### Changed
- Disabled concurrent Jenkins builds on same branch/commit
- Added build timeout to avoid hung builds
- CASMCMS-8918: Get SLES packages from `artifactory` instead of `slemaster` to avoid build problems

### Removed
- Removed defunct files leftover from previous versioning system

## [2.0.0] - 2023-04-05
### Changed
 - CASMCMS-8252: Enabled building of unstable artifacts
 - CASMCMS-8252: Updated header of update_versions.conf to reflect new tool options
 - CASMCMS-7169: Conman will be restarted if there is a change to the credentials
 - CASMCMS-7167: Implementing location api for pod location data to filter through console data

### Fixed
 - CASMCMS-8252: Update Chart with correct image and chart version strings during builds.

## [1.7.3] - 2023-02-24
### Changed
- CASMCMS-8423: Linting changes for new version of gofmt.

## [1.7.2] - 2023-2-2
### Changed
- CASMINST-5878: Remove post-install hook that depends on console-operator PVC.

## [1.7.1] - 2022-12-20
### Added
- Add Artifactory authentication to Jenkinsfile

## [1.7.0] - 2022-09-12
### Changed
 - Spelling corrections.
 - CASMCMS-8076: Changed base image to use sp4

## [1.6.0] - 2022-08-04
### Changed
 - CASMCMS-8140: Fix handling Hill nodes.

## [1.5.0] - 2022-07-13
### Changed
 - CASMCMS-8016: Update hsm api to v2.

## [1.4.0] - 2022-07-12
### Changed
 - CASMCMS-7830: Update the base image to newer version and resolve *.dev.cray.com addresses.
 - CASMCMS-8055: Add pod anti-affinity to helm chart.
