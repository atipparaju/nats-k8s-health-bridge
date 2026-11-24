# Dependency license and redistribution evidence — pre-publication

**Reviewed:** 2026-11-24 · **Scope:** 28 modules selected by the current `go.mod` on private `main` · **Issue:** [#80](https://github.com/atipparaju/nats-k8s-health-bridge/issues/80) · **Status:** **preliminary source-license inventory; NOT legal clearance**

This is a *source-repository license-file verification*, not a certification that all linked libraries, generated files, embedded code, OS packages or final Docker images comply with redistribution obligations. The SPDX image SBOM previously reported **35 packages**, including **29** entries where both license fields were `NOASSERTION`. That scanner result is a separate inventory scope, **not** a count of unlicensed Go modules. See [pre-publication GO/NO-GO](pre-publication-go-no-go.md).

## Current module graph — license texts located at exact selected releases

The links below identify the upstream source used for each module. Apache/MIT/BSD describe the observed license-file text and must not substitute for checking file-level exceptions or retained notices. For pseudo-versions, a matching upstream commit hash was inspected rather than assuming a version tag existed.

| Go module (selected version) | Upstream license evidence | Preliminary classification |
| --- | --- | --- |
| `github.com/nats-io/nats.go v1.55.0` | [LICENSE](https://github.com/nats-io/nats.go/blob/v1.55.0/LICENSE) | Apache-2.0 |
| `github.com/prometheus/client_golang v1.25.0` | [LICENSE](https://github.com/prometheus/client_golang/blob/v1.25.0/LICENSE), [NOTICE](https://github.com/prometheus/client_golang/blob/v1.25.0/NOTICE) | Apache-2.0; notice review |
| `github.com/prometheus/client_model v0.6.3` | [LICENSE](https://github.com/prometheus/client_model/blob/v0.6.3/LICENSE), [NOTICE](https://github.com/prometheus/client_model/blob/v0.6.3/NOTICE) | Apache-2.0; notice review |
| `github.com/spf13/cobra v1.10.1` | [LICENSE.txt](https://github.com/spf13/cobra/blob/v1.10.1/LICENSE.txt) | Apache-2.0 |
| `github.com/spf13/viper v1.21.0` | [LICENSE](https://github.com/spf13/viper/blob/v1.21.0/LICENSE) | MIT |
| `github.com/beorn7/perks v1.0.1` | [LICENSE](https://github.com/beorn7/perks/blob/v1.0.1/LICENSE) | MIT |
| `github.com/cespare/xxhash/v2 v2.3.0` | [LICENSE.txt](https://github.com/cespare/xxhash/blob/v2.3.0/LICENSE.txt) | MIT |
| `github.com/fsnotify/fsnotify v1.9.0` | [LICENSE](https://github.com/fsnotify/fsnotify/blob/v1.9.0/LICENSE) | BSD-style |
| `github.com/go-viper/mapstructure/v2 v2.4.0` | [LICENSE](https://github.com/go-viper/mapstructure/blob/v2.4.0/LICENSE) | MIT |
| `github.com/inconshreveable/mousetrap v1.1.0` | [LICENSE](https://github.com/inconshreveable/mousetrap/blob/v1.1.0/LICENSE) | Apache-2.0 |
| `github.com/klauspost/compress v1.20.1` | [LICENSE](https://github.com/klauspost/compress/blob/v1.20.1/LICENSE) | BSD-style; review retained headers |
| `github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822` | [LICENSE at commit a7dc8b61c822](https://github.com/munnerz/goautoneg/blob/a7dc8b61c822/LICENSE) | BSD-style |
| `github.com/nats-io/nkeys v0.4.16` | [LICENSE](https://github.com/nats-io/nkeys/blob/v0.4.16/LICENSE) | Apache-2.0 |
| `github.com/nats-io/nuid v1.0.1` | [LICENSE](https://github.com/nats-io/nuid/blob/v1.0.1/LICENSE) | Apache-2.0 |
| `github.com/pelletier/go-toml/v2 v2.2.4` | [LICENSE](https://github.com/pelletier/go-toml/blob/v2.2.4/LICENSE) | MIT |
| `github.com/prometheus/common v0.72.0` | [LICENSE](https://github.com/prometheus/common/blob/v0.72.0/LICENSE), [NOTICE](https://github.com/prometheus/common/blob/v0.72.0/NOTICE) | Apache-2.0; notice review |
| `github.com/prometheus/procfs v0.22.0` | [LICENSE](https://github.com/prometheus/procfs/blob/v0.22.0/LICENSE), [NOTICE](https://github.com/prometheus/procfs/blob/v0.22.0/NOTICE) | Apache-2.0; notice review |
| `github.com/sagikazarmark/locafero v0.11.0` | [LICENSE](https://github.com/sagikazarmark/locafero/blob/v0.11.0/LICENSE) | MIT |
| `github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8` | [LICENSE at commit 5f936abd7ae8](https://github.com/sourcegraph/conc/blob/5f936abd7ae8/LICENSE) | MIT |
| `github.com/spf13/afero v1.15.0` | [LICENSE.txt](https://github.com/spf13/afero/blob/v1.15.0/LICENSE.txt) | Apache-2.0 |
| `github.com/spf13/cast v1.10.0` | [LICENSE](https://github.com/spf13/cast/blob/v1.10.0/LICENSE) | MIT |
| `github.com/spf13/pflag v1.0.10` | [LICENSE](https://github.com/spf13/pflag/blob/v1.0.10/LICENSE) | BSD-style |
| `github.com/subosito/gotenv v1.6.0` | [LICENSE](https://github.com/subosito/gotenv/blob/v1.6.0/LICENSE) | MIT |
| `go.yaml.in/yaml/v3 v3.0.5` | [LICENSE in canonical yaml/go-yaml v3.0.5](https://github.com/yaml/go-yaml/blob/v3.0.5/LICENSE) | Mixed MIT + Apache-2.0, file-specific text |
| `golang.org/x/crypto v0.57.0` | [LICENSE](https://github.com/golang/crypto/blob/v0.57.0/LICENSE) | BSD-style |
| `golang.org/x/sys v0.48.0` | [LICENSE](https://github.com/golang/sys/blob/v0.48.0/LICENSE) | BSD-style |
| `golang.org/x/text v0.42.0` | [LICENSE](https://github.com/golang/text/blob/v0.42.0/LICENSE) | BSD-style |
| `google.golang.org/protobuf v1.36.12` | [LICENSE](https://github.com/protocolbuffers/protobuf-go/blob/v1.36.12/LICENSE) | BSD-style |

**Directly verified:** All 28 entries above have an accessible license text at their selected tag or pseudo-version commit. Four selected Prometheus repositories also have a versioned upstream `NOTICE`. Verification does **not** establish the distribution terms of every individual file in each module, or that a top-level bridge `NOTICE` is automatically required.

## Source repository vs binary/container distribution

| Publishing activity | License and notice implications to examine | Status |
| --- | --- | --- |
| Public source repository with `go.mod`, without copied vendor code | Include valid permission for the bridge's own authored files; confirm any borrowed/generated code, snippets, copied documentation/assets, and their notices. Dependency references alone do not equal embedding the complete dependency source | **Pending code provenance and history review** |
| Source tree with third-party vendored/copied contents, if any | Preserve source copyright and required LICENSE/NOTICE for those actual files; review modifications and license compatibility | **Presence/absence not fully audited in all historical refs** |
| Distributed compiled Linux/arm64/amd64 executable | Evaluate the linked dependency licenses and any reproduced notices/copyright information, including four Prometheus upstream notices; design the distribution notice bundle | **Not cleared** |
| Distroless-based container image | All binary obligations plus OS image packages (SBOM currently includes GPL/MPL and LicenseRef entries); verify package distribution, attribution, and license texts independently | **Not cleared** |
| GHCR image publication | Binary/container obligations **plus** immutable published image manifest, signing/attestation policy, actual package visibility and separate publication authorization | **Not authorized** |

## Verification still necessary

- [x] Locate license text for all **28** pinned Go modules at exact tags/pseudo-version commits.
- [x] Identify four exact-tag Prometheus `NOTICE` files for attribution review.
- [ ] Review code-level provenance, generated/copied files, test assets and original copyright ownership.
- [ ] Inspect all third-party source licenses/NOTICE for file-level exceptions and the obligations of the **specific distributed artifact**.
- [ ] Resolve the 29 SBOM `NOASSERTION` records with source evidence and classify any remaining OS-package LicenseRef and GPL/MPL expressions; keep scope (image vs source) explicit.
- [ ] Decide and prepare a correct distribution notice bundle **only after** required content is established. Do not invent attribution, duplicate incompatible terms, or imply certification.
- [ ] Obtain owner/qualified review of the final attribution and rights evidence; store sensitive legal conclusions privately, not in public Git history.

**Disposition: useful technical evidence obtained; legal/licensing clearance is NOT COMPLETE.** This document does not authorize public source visibility, a release or a registry push.
