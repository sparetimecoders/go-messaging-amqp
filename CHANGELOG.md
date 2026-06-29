# Changelog

## [0.0.4](https://github.com/sparetimecoders/go-messaging-amqp/compare/v0.0.3...v0.0.4) (2026-06-29)


### Bug Fixes

* **deps:** update module github.com/rabbitmq/amqp091-go to v1.12.0 ([#46](https://github.com/sparetimecoders/go-messaging-amqp/issues/46)) ([c655fc7](https://github.com/sparetimecoders/go-messaging-amqp/commit/c655fc7e461b02da5803f9d0805ed361746a9536))
* **deps:** update opentelemetry-go monorepo to v1.44.0 ([#50](https://github.com/sparetimecoders/go-messaging-amqp/issues/50)) ([75acc8d](https://github.com/sparetimecoders/go-messaging-amqp/commit/75acc8d1d75bb2ff388caed35ae68cb8a6f7e0dc))

## [0.0.3](https://github.com/sparetimecoders/go-messaging-amqp/compare/v0.0.2...v0.0.3) (2026-04-07)


### Bug Fixes

* address code review findings ([#38](https://github.com/sparetimecoders/go-messaging-amqp/issues/38)) ([c7a025f](https://github.com/sparetimecoders/go-messaging-amqp/commit/c7a025f27726a37a5261a196a61db0c0d711fb10))
* address code review issues across AMQP transport ([#34](https://github.com/sparetimecoders/go-messaging-amqp/issues/34)) ([2adf08f](https://github.com/sparetimecoders/go-messaging-amqp/commit/2adf08f177028657d48c23444cdd4c932b1791a6))
* **deps:** correct messaging import paths ([#40](https://github.com/sparetimecoders/go-messaging-amqp/issues/40)) ([221d47d](https://github.com/sparetimecoders/go-messaging-amqp/commit/221d47d06dd025e172706f4829af5bbbd092efb8))
* **deps:** update opentelemetry-go monorepo to v1.43.0 ([#39](https://github.com/sparetimecoders/go-messaging-amqp/issues/39)) ([86ca541](https://github.com/sparetimecoders/go-messaging-amqp/commit/86ca541ce18abebbd8b4de49cc9b90d7b36e25f4))
* propagate parent context in extractToContext ([#35](https://github.com/sparetimecoders/go-messaging-amqp/issues/35)) ([53c75b3](https://github.com/sparetimecoders/go-messaging-amqp/commit/53c75b3771826c6bfa49da10b70392197ac16623))

## [0.0.2](https://github.com/sparetimecoders/go-messaging-amqp/compare/v0.0.1...v0.0.2) (2026-03-13)


### Features

* add outbox raw publisher adapter for AMQP ([#21](https://github.com/sparetimecoders/go-messaging-amqp/issues/21)) ([d40334a](https://github.com/sparetimecoders/go-messaging-amqp/commit/d40334a337ebc2b748aab5d727c3f09a1a4f2a05))


### Bug Fixes

* **deps:** update opentelemetry-go monorepo to v1.42.0 ([#23](https://github.com/sparetimecoders/go-messaging-amqp/issues/23)) ([7c9c8e2](https://github.com/sparetimecoders/go-messaging-amqp/commit/7c9c8e2a2a020d330664d51bf74ffdb172343dd3))
* downgrade consumer loop exit log from Error to Warn ([#26](https://github.com/sparetimecoders/go-messaging-amqp/issues/26)) ([37aeaf1](https://github.com/sparetimecoders/go-messaging-amqp/commit/37aeaf10d0adc394d4d3b6ec1af0ca21338b7525))
* remove synchronize trigger from review gate ([#24](https://github.com/sparetimecoders/go-messaging-amqp/issues/24)) ([b67c676](https://github.com/sparetimecoders/go-messaging-amqp/commit/b67c676036a7c3af37c337fadfba21b3ab5cea2c))

## 0.0.1 (2026-03-06)


### Features

* add automerge workflow for PRs ([#6](https://github.com/sparetimecoders/go-messaging-amqp/issues/6)) ([f9ad462](https://github.com/sparetimecoders/go-messaging-amqp/commit/f9ad46228a3d0b92bc7f982b558ec64fdc6f2dba))
* initial AMQP transport implementation ([1780f64](https://github.com/sparetimecoders/go-messaging-amqp/commit/1780f64e77daa69b448c9cbc5a0ad8f22781511b))


### Bug Fixes

* allow github-actions[bot] through review gate ([4413200](https://github.com/sparetimecoders/go-messaging-amqp/commit/4413200f6fa84128a490adc385cdc4d2205783e4))
* **ci:** add job name to match required status check ([#16](https://github.com/sparetimecoders/go-messaging-amqp/issues/16)) ([8168aef](https://github.com/sparetimecoders/go-messaging-amqp/commit/8168aefa85c0c5abe9662382cad327d4ef1cde6c))
* **deps:** update module github.com/prometheus/client_golang to v1.23.2 ([#9](https://github.com/sparetimecoders/go-messaging-amqp/issues/9)) ([9a87c0b](https://github.com/sparetimecoders/go-messaging-amqp/commit/9a87c0b236e1805e2f533cec4a79044b9e18fa6d))
* **deps:** update opentelemetry-go monorepo to v1.41.0 ([#10](https://github.com/sparetimecoders/go-messaging-amqp/issues/10)) ([5bfcd44](https://github.com/sparetimecoders/go-messaging-amqp/commit/5bfcd44d5d040a003f7222af00425b5d6ee0d87c))
* rename module and use tagged spec/tck dependencies ([#3](https://github.com/sparetimecoders/go-messaging-amqp/issues/3)) ([e6d5771](https://github.com/sparetimecoders/go-messaging-amqp/commit/e6d5771416d0605a533b3cbda34e02b27d25f117))
* review gate allows owner PRs, requires approval for others ([fe7f4b3](https://github.com/sparetimecoders/go-messaging-amqp/commit/fe7f4b32533afd632cd87ba42bea1c95171452f5))
* set initial-version to 0.0.1 for release-please ([566dc42](https://github.com/sparetimecoders/go-messaging-amqp/commit/566dc42788ec1ca029d356fd681edf6a4794a519))
* update Go module path to match repo ([#2](https://github.com/sparetimecoders/go-messaging-amqp/issues/2)) ([1c423b9](https://github.com/sparetimecoders/go-messaging-amqp/commit/1c423b998c2cb5fc0c6a3f3921abcbc65e014535))
