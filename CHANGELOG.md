# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- CASMCMS-8918: Get SLES packages from `artifactory` instead of `slemaster` to avoid build problems

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

[1.0.0] - (no date)
