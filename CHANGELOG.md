# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Workflows update with `devctl`

## [2.0.0] - 2024-08-07

### Changed

- Migration to kubebuilder - operation simplification & metric addition

## [1.6.4] - 2024-07-30

### Changed

- Bump architect-orb@5.3.1 to fix CVE-2024-24790.

## [1.6.3] - 2024-03-12

## [1.6.2] - 2024-03-12

### Changed

- Migrate to app-build-suite

## [1.6.1] - 2024-02-27

### Added

- Ensuring that the organization namespace is patched with the organization labels in case they are not present.

### Changed

- Configure `gsoci.azurecr.io` as the default container image registry.

## [1.6.0] - 2023-10-03

### Added

- New config var `resyncPeriod` to control the reconcile loop resync period

## [1.5.0] - 2023-10-02

### Changed

- Propagate `global.podSecurityStandards.enforced` value set to `false` for PSS migration

## [1.4.0] - 2023-07-04

### Changed

- Add Service Monitor.

## [1.3.0] - 2023-06-28

### Changed

- Update deployment to be PSS compliant.

## [1.2.0] - 2023-06-19

### Removed

- Remove pull secret from chart.

## [1.1.0] - 2023-06-19

### Added

- Add `Namespace` and `Age` columns to `Organization` CRD.

### Removed

- Stop pushing to `openstack-app-collection`.

## [1.0.7] - 2023-04-25

### Changed

- Remove shared app collection from circle CI
- Dependencies: replaced `github.com/giantswarm/operatorkit/v8` with `github.com/giantswarm/operatorkit/v7` (latest version)

## [1.0.6] - 2023-03-22

### Added

- Added the use of the runtime/default seccomp profile.

### Fixed

- Prevented deletion of Organization CR until the organization namespace is deleted successfully

## [1.0.5] - 2023-01-16

## [1.0.4] - 2022-11-15

### Changed

- Don't return an error in case credentiald fails during deletion

## [1.0.3] - 2022-08-05

### Changed

- Update CI (archigtect-orb)

## [1.0.2] - 2022-03-31

### Changed

- Dependencies: replace `github.com/dgrijalva/jwt-go` with `github.com/golang-jwt/jwt/v4`.

## [1.0.1] - 2022-03-31

### Added

- Add Organization CR example
- Move PR template into .github folder

## [1.0.0] - 2022-03-25

### Changed

- Drop dependency on apiextensions by moving Organization API into this repository.
- Update k8sclient and operatorkit to v6.
- Require Go v1.17

### Fixed

- Add missing `imagePullSecret`.

## [0.10.3] - 2022-01-12

### Changed

- Don't return an error in case deletion of legacy organization fails.

## [0.10.2] - 2021-10-28

### Fixed

- Use `Status()` client to patch `Organization`'s status with a namespace.

## [0.10.1] - 2021-10-26

### Changed

- Don't return an error in case creation of legacy organization fails.

## [0.10.0] - 2021-10-12

### Added

- Ensure `Organization` CR in Azure MCs have the `subscriptionid` annotation set.

### Changed

- Use `Patch` to save `Namespace` in `Status` to avoid write conflicts.

## [0.9.0] - 2021-07-05

### Added

- Introduce local interfaces to make easier to test the organization handler.
- Save created namespace in Organization status field.

### Changed

- Updated `apiextensions` dependency to v3.
- Updated `operatorkit` dependency to v5.
- Updated `k8sclient` dependency to v5.
- Always try to create organization in `companyd` even when organization namespace already exists.
- Pin `jwt-go` dependency to avoid version with vulnerabilities.

## [0.8.0] - 2021-05-24

### Changed

- Update `architect-orb` to v3.0.0.

## [0.7.0] - 2021-05-21

### Changed

- Set config version in `Chart.yaml`.

## [0.6.0] - 2021-05-17

### Added

- Also delete legacy `companyd` organizations when deleting `Organization` CRs.
- Prevent deleting legacy credentials when deleting `Organization` CRs.

## [0.5.0] - 2020-09-24

### Changed

- Updated Kubernetes dependencies to v1.18.9 and operatorkit to v2.0.1.

### Added

- Add monitoring labels.

## [0.4.0] - 2020-08-21

### Added

- Add NetworkPolicy.

## [0.3.0] - 2020-08-14

### Changed

- Updated backward incompatible Kubernetes dependencies to v1.18.5.

## [0.2.0] - 2020-08-13

### Added

- Add dependabot configuration.

### Changed

- Update operatorkit to v1.2.0 and k8sclient to v3.1.2.

## [0.1.0] - 2020-06-03

### Added

- First release.

[Unreleased]: https://github.com/giantswarm/organization-operator/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/giantswarm/organization-operator/compare/v1.6.4...v2.0.0
[1.6.4]: https://github.com/giantswarm/organization-operator/compare/v1.6.3...v1.6.4
[1.6.3]: https://github.com/giantswarm/organization-operator/compare/v1.6.2...v1.6.3
[1.6.2]: https://github.com/giantswarm/organization-operator/compare/v1.6.1...v1.6.2
[1.6.1]: https://github.com/giantswarm/organization-operator/compare/v1.6.0...v1.6.1
[1.6.0]: https://github.com/giantswarm/organization-operator/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/giantswarm/organization-operator/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/giantswarm/organization-operator/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/giantswarm/organization-operator/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/giantswarm/organization-operator/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/giantswarm/organization-operator/compare/v1.0.7...v1.1.0
[1.0.7]: https://github.com/giantswarm/organization-operator/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/giantswarm/organization-operator/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/giantswarm/organization-operator/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/giantswarm/organization-operator/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/giantswarm/organization-operator/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/giantswarm/organization-operator/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/giantswarm/organization-operator/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/giantswarm/organization-operator/compare/v0.10.3...v1.0.0
[0.10.3]: https://github.com/giantswarm/organization-operator/compare/v0.10.2...v0.10.3
[0.10.2]: https://github.com/giantswarm/organization-operator/compare/v0.10.1...v0.10.2
[0.10.1]: https://github.com/giantswarm/organization-operator/compare/v0.10.0...v0.10.1
[0.10.0]: https://github.com/giantswarm/organization-operator/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/giantswarm/organization-operator/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/giantswarm/organization-operator/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/giantswarm/organization-operator/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/giantswarm/organization-operator/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/giantswarm/organization-operator/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/giantswarm/organization-operator/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/giantswarm/organization-operator/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/giantswarm/organization-operator/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/giantswarm/organization-operator/releases/tag/v0.1.0
