# Change Log
All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Added

- go: Updated build to use Go 1.17.7. [#32](https://github.com/elastic/stream/pull/32)

## [0.6.2]

### Fixed

- Only message sent via TCP and TLS are framed using newlines. UDP messages will
no longer contain a trailing newline. [#31](https://github.com/elastic/stream/pull/31)

## [0.6.1]

### Fixed

- Fixed libdbus shared object error in Dockerfile. [#30](https://github.com/elastic/stream/pull/30)

## [0.6.0]

### Added

- Added file template helper function. [#25](https://github.com/elastic/stream/pull/25)
- Added regular expression-based body matching [#26](https://github.com/elastic/stream/pull/26)

### Fixed

- Ensure basic auth and body are only tested if explicitly set. [#28](https://github.com/elastic/stream/pull/28)

## [0.5.0]

- Added option to set up custom buffer size for the log reader. [#22](https://github.com/elastic/stream/pull/22)
- Added support for glob patterns. [#22](https://github.com/elastic/stream/pull/22)
- Convert `http-server` output into a mock HTTP server.[#23](https://github.com/elastic/stream/pull/23)

## [0.4.0]

- Added HTTP Server output. [#19](https://github.com/elastic/stream/pull/19)

## [0.3.0]

- Added `--rate-limit` flag to control rate (in bytes/sec) of UDP streaming. [#12](https://github.com/elastic/stream/pull/12)
- Added `gcppubsub` output option. [#13](https://github.com/elastic/stream/pull/13)

## [0.2.0]

- Added `--insecure` to disable TLS verification for the TLS and webhook outputs.

## [0.1.0]

### Added

- Added webhook output option.
- Added the ability to set all flags via environment variable.

## [0.0.1]

### Added

- Added pcap and log file inputs.
- Added udp, tcp, and tls outputs.

[Unreleased]: https://github.com/elastic/stream/compare/v0.6.2...HEAD
[0.6.2]: https://github.com/elastic/stream/releases/tag/v0.6.2
[0.6.1]: https://github.com/elastic/stream/releases/tag/v0.6.1
[0.6.0]: https://github.com/elastic/stream/releases/tag/v0.6.0
[0.5.0]: https://github.com/elastic/stream/releases/tag/v0.5.0
[0.4.0]: https://github.com/elastic/stream/releases/tag/v0.4.0
[0.3.0]: https://github.com/elastic/stream/releases/tag/v0.3.0
[0.2.0]: https://github.com/elastic/stream/releases/tag/v0.2.0
[0.1.0]: https://github.com/elastic/stream/releases/tag/v0.1.0
[0.0.1]: https://github.com/elastic/stream/releases/tag/v0.0.1
