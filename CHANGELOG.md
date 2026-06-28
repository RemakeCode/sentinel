## [2.0.0-beta.2](https://github.com/RemakeCode/sentinel/compare/v2.0.0-beta.1...v2.0.0-beta.2) (2026-06-28)

## [2.0.0-beta.1](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.8...v2.0.0-beta.1) (2026-06-26)

### ⚠ BREAKING CHANGES

* for 2.0. Migration was only used during initial config
load; removed as part of breaking changes.
* The app now requires GTK4 instead of GTK3.

### build

* migrate runtime dependency from GTK3 to GTK4 ([7913d35](https://github.com/RemakeCode/sentinel/commit/7913d3554ba593cafb3d6a42606d108dff50a130))

### 🚀 New Features

* add achievement progress notification modes ([86a3be9](https://github.com/RemakeCode/sentinel/commit/86a3be9938218fdfa68eac0a701adc762c3d77f4))
* add achievement progress update setting ([6902599](https://github.com/RemakeCode/sentinel/commit/6902599e2030ad2bd3d838977fcef6f2cc902e49))
* add notification sse connection retries ([61a770a](https://github.com/RemakeCode/sentinel/commit/61a770a204b957ddae450ab38be0e0036f8b6bf4))
* add per-game refresh with context menu and UI feedback ([58b97e1](https://github.com/RemakeCode/sentinel/commit/58b97e129501e92238b49ea28520801fe3eae064))
* add Steam emulator INI achievement parsing ([cea5433](https://github.com/RemakeCode/sentinel/commit/cea5433367fe06d6e6ef1b79f546113903f34b5f))
* **decky-plugin:** add per-game refresh from library context menu ([72c8058](https://github.com/RemakeCode/sentinel/commit/72c8058dbd2d2198927985e7265ee5caaa10701f))
* **decky-plugin:** align settings and achievement sort parity ([43aaf31](https://github.com/RemakeCode/sentinel/commit/43aaf31d4b38b34dfae8d66a6b9c106bf8120df4))
* **decky:** add dedicated backend entrypoint ([9fb9a00](https://github.com/RemakeCode/sentinel/commit/9fb9a0019c60285db289189755b0d0e47da02307))
* implement achievements progress toggle setting for sentinel-decky ([a4f058b](https://github.com/RemakeCode/sentinel/commit/a4f058bfa70015b38605f5a84dc7264dc4eb2ac5))
* remove migrate package ([be64b89](https://github.com/RemakeCode/sentinel/commit/be64b89c476c02a1bdc36fd197f7b26e907ab338))
* support for more emus ([ca0e883](https://github.com/RemakeCode/sentinel/commit/ca0e883d3a1990ca0ff217b6c2008bdeaa5158a7))

### 🐛 Bug Fixes

* blank context menu item on Linux ([22f38ec](https://github.com/RemakeCode/sentinel/commit/22f38ecb317f9a96c48bca8704aabb7b9d04fc1c))
* **ci:** install alsa headers for notifier tests ([f89072f](https://github.com/RemakeCode/sentinel/commit/f89072fc0a67e06c30cfa9a2e42e564cf5faa04a))
* **ci:** install alsa headers for notifier tests ([9d64dcf](https://github.com/RemakeCode/sentinel/commit/9d64dcfb41d3eb3c53e82906c44a6abdc665daed))
* **decky:** avoid backend output pipes ([32903d3](https://github.com/RemakeCode/sentinel/commit/32903d3bc312e160fd7bcf530108c7a31e527809))
* fix crash when a prefix is suddenly removed by another process  ([6306079](https://github.com/RemakeCode/sentinel/commit/63060797488da69d47d0c71de3f4a56b35ea6d04)), closes [#39](https://github.com/RemakeCode/sentinel/issues/39)
* game state tracker race conditions ([8511de3](https://github.com/RemakeCode/sentinel/commit/8511de3698230b389e756e87abfedba1ecab7020))
* library sync and race condition issues ([324bc8b](https://github.com/RemakeCode/sentinel/commit/324bc8b6374c1baa23edcc355d20190dc8a54931))
* library sync and race condition issues ([31f0aa8](https://github.com/RemakeCode/sentinel/commit/31f0aa8b09bd30abf49d976d32685c271678ae19))
* logging from go backend into the python backend stub to be picked up by decky-plugin-service ([f85d53f](https://github.com/RemakeCode/sentinel/commit/f85d53fe88b493be278d597b484e1eff006583f4))
* normalize achievement icon paths from Steam key source API ([dca59c0](https://github.com/RemakeCode/sentinel/commit/dca59c0a9237822b819b0668ffc42a75d2c77de6))
* notifications missed on the first time a game creates its folder ([86da881](https://github.com/RemakeCode/sentinel/commit/86da88163093e3bb616776adddac9660dd3c0798))
* **notifier:** normalize bundled notification sounds ([fb2a622](https://github.com/RemakeCode/sentinel/commit/fb2a622e8c3473b44b2c7ea318ad5f9a19481ca3))
* **notifier:** normalize bundled notification sounds ([71d3e11](https://github.com/RemakeCode/sentinel/commit/71d3e11fe3d0a8a8b634de7824bcb40dd8960b1b))
* **notifier:** omit missing achievement icons ([5456795](https://github.com/RemakeCode/sentinel/commit/5456795c81215a2ab435419c18c6360fa6a7e919))
* **notifier:** replace shell notifications with native playback ([f0402ea](https://github.com/RemakeCode/sentinel/commit/f0402eadcb1966bc301bbc5ef937d6e85a7f0fb9))
* **notifier:** replace shell notifications with native playback ([3f6c1f7](https://github.com/RemakeCode/sentinel/commit/3f6c1f70b46c0f9278dee9a5ce4850d6c7d397ab))
* **notifier:** retry audio speaker initialization ([18cf34f](https://github.com/RemakeCode/sentinel/commit/18cf34fc25dc26b3317f0c948b1c2795278cbfe8))
* **notifier:** retry audio speaker initialization ([b7a449a](https://github.com/RemakeCode/sentinel/commit/b7a449a8ba18bcddabae4b50050875af9fb8bb04))
* optimizations of the pathWalker and fixes for the compatdata directory as prefix ([5c34abd](https://github.com/RemakeCode/sentinel/commit/5c34abd6741cb994df638ef887a462a57c8214f7))
* polling re-render loop in games-context ([1795628](https://github.com/RemakeCode/sentinel/commit/1795628d984349a97f1fefd7aed4102cf55e3e0a))
* restore GetNotificationExpireTime (used by frontend settings) ([10350d7](https://github.com/RemakeCode/sentinel/commit/10350d7ac23f7224575702afe867871b1c6430d7))
* slog format string and json.Marshal error handling ([5c995ed](https://github.com/RemakeCode/sentinel/commit/5c995ed8dc3bf94df048f97eb0b27dfa79c44746))
* **steam:** remove cache read repair ([729b95c](https://github.com/RemakeCode/sentinel/commit/729b95c6b09366d0c9e0d08d9878cf21ba6a654a))
* unlock hidden achievements should  no longer be blurred ([40ff254](https://github.com/RemakeCode/sentinel/commit/40ff2548778d873ef616a3ebf21b12ca412c0211))

## [1.1.0-beta.8](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.7...v1.1.0-beta.8) (2026-06-14)

### 🚀 New Features

* **decky:** add dedicated backend entrypoint ([94e902a](https://github.com/RemakeCode/sentinel/commit/94e902a107e7ae52757369a3ba9e4283eaaab23e))

### 🐛 Bug Fixes

* logging from go backend into the python backend stub to be picked up by decky-plugin-service ([84516b7](https://github.com/RemakeCode/sentinel/commit/84516b71e1928761f145e4012e18b6d0d17d38b8))

## [1.1.0-beta.7](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.6...v1.1.0-beta.7) (2026-06-12)

### 🚀 New Features

* blur hidden achievements ([1403895](https://github.com/RemakeCode/sentinel/commit/1403895f31e3b5e810be9d0ec4a0d9f71f004f81))

## [1.1.0-beta.6](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.5...v1.1.0-beta.6) (2026-06-12)

### 🚀 New Features

* blur hidden achievements ([#58](https://github.com/RemakeCode/sentinel/issues/58)) ([495e6db](https://github.com/RemakeCode/sentinel/commit/495e6db9de422e9dcd2159b4ad6cbdf502ad0f1e))

## [1.1.0-beta.5](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.4...v1.1.0-beta.5) (2026-06-12)

### 🚀 New Features

* blur hidden achievements ([6d3556c](https://github.com/RemakeCode/sentinel/commit/6d3556cafa870d15d4c728557ffdd9592e36c3ac))
* blur hidden achievements ([bcec3b2](https://github.com/RemakeCode/sentinel/commit/bcec3b2f4f585f91fb506f8a3dc2f8f524a9d126))

## [1.1.0-beta.4](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.3...v1.1.0-beta.4) (2026-06-08)

### 🐛 Bug Fixes

* **decky-plugin:** missing flag in plugin config ([#49](https://github.com/RemakeCode/sentinel/issues/49)) ([435eb41](https://github.com/RemakeCode/sentinel/commit/435eb411f79dee042881977fcd46624d0efbe671))
* fix permission errors ([5d3a39c](https://github.com/RemakeCode/sentinel/commit/5d3a39c97c1db86df5517e171416886c10c13229))

## [1.1.0-beta.3](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.2...v1.1.0-beta.3) (2026-06-08)

### 🚀 New Features

* autostart implementation now uses wails3's implementation ([15f2fb5](https://github.com/RemakeCode/sentinel/commit/15f2fb591f2218d8a4d2682fddc7fc673bf3de4a))
* decky Plugin implementation ([2627993](https://github.com/RemakeCode/sentinel/commit/2627993d2d1490df57addfeff040505c8df3b441))
* decky Plugin implementation ([a22366e](https://github.com/RemakeCode/sentinel/commit/a22366efc6beb7aee14963e9527ee12439ddcf35))
* decky Plugin implementation ([2d9cb67](https://github.com/RemakeCode/sentinel/commit/2d9cb67b9881222162cde43703d68db9fdd60c15))
* **decky-plugin:** achievement page implementations ([e5baec9](https://github.com/RemakeCode/sentinel/commit/e5baec98062814e9a1a7ff17803095f4263fcae8))
* **decky-plugin:** add a service status section to the setttings menu ([7ab8228](https://github.com/RemakeCode/sentinel/commit/7ab8228a0d30d08937e1bd33805322637de7f467))
* **decky-plugin:** add Library empty state ([a42ca3b](https://github.com/RemakeCode/sentinel/commit/a42ca3b7976d9e220829b057cdb1cb34f8592b72))
* **decky-plugin:** add settings page to replicate the relevant desktop settings ([7962b5d](https://github.com/RemakeCode/sentinel/commit/7962b5de9c8a3c821a60679078ab03ac796f0f96))
* **decky-plugin:** api service auth ([1df7abc](https://github.com/RemakeCode/sentinel/commit/1df7abce3be27633b8454a3972e230769eced8e1))
* **decky-plugin:** beta release readiness ([873bb40](https://github.com/RemakeCode/sentinel/commit/873bb40ad2b4f6d614ebf734e0a097e0ca9fb026))
* **decky-plugin:** implement `matched game` view to display game details and achievements with corresponding states ([20e4a97](https://github.com/RemakeCode/sentinel/commit/20e4a9770ee0ecd36202dbbffd7e7775079b4da2))
* **decky-plugin:** implement `unmatched game` picker, settings for managing unmatched games ([b3468ac](https://github.com/RemakeCode/sentinel/commit/b3468ac5a6b581423e1f71ee2c3ad902e46eb965))
* **decky-plugin:** implement Library page for Sentinel found GSE supported games ([e7a6775](https://github.com/RemakeCode/sentinel/commit/e7a67755bbde0ab7773a8b85978064de7260f6e8))
* **decky-plugin:** implement Library page for Sentinel found GSE supported games ([45fb413](https://github.com/RemakeCode/sentinel/commit/45fb4134da85c7220f342f6b2c0f615a72a868a7))
* **decky-plugin:** implement notification sound ([21ffafa](https://github.com/RemakeCode/sentinel/commit/21ffafaf95ca35040ce929d535979a59269a4648))
* **decky-plugin:** more work on the achievement page implementations ([3d34fbb](https://github.com/RemakeCode/sentinel/commit/3d34fbb03b08592d86fe960dfed5304418d5a312))
* **decky-plugin:** non-Steam game tracker. Checks for when a non steam game starts and stop ([d28dacb](https://github.com/RemakeCode/sentinel/commit/d28dacbab63c5edbb4d0577cc6246e073e2f10c7))
* initial implementation for sentinel decky-loader plugin ([af724b4](https://github.com/RemakeCode/sentinel/commit/af724b4d39abfce60c739ef0f72789c62c4b1b18))
* initial implementation for sentinel decky-loader plugin ([b9fb717](https://github.com/RemakeCode/sentinel/commit/b9fb717b37d6840df80df8bf5efe1c0a93ff6d6a))
* sentinel-decky-plugin backend ([d2059ac](https://github.com/RemakeCode/sentinel/commit/d2059ac9bfb7449406f93c2d89143317d10069bb))

### 🐛 Bug Fixes

* **ci:** clean node_modules before npm ci; build Go binary before decky plugin ([f30094e](https://github.com/RemakeCode/sentinel/commit/f30094ea75530771ff57ace4b4b30630d748356e))
* **ci:** semantic-dry-run condition reverted to push-only ([ec83b32](https://github.com/RemakeCode/sentinel/commit/ec83b326b9981d48a5005b23ccb793cff3f293bd))
* **decky-plugin:** beta release readiness ([4ae4d64](https://github.com/RemakeCode/sentinel/commit/4ae4d64e4bd8d1852ef7b8259e0aea492e06ef9c))
* **decky-plugin:** beta release readiness ([88a9fbe](https://github.com/RemakeCode/sentinel/commit/88a9fbec756b2987bf121c418ef9839104999b87))
* **decky-plugin:** empty state button fix ([20fb1b2](https://github.com/RemakeCode/sentinel/commit/20fb1b2bb4dc29214a9c5bc15592ea280622a665))
* **decky-plugin:** empty State fixes ([c2264c3](https://github.com/RemakeCode/sentinel/commit/c2264c31fdaa2a8df82460737b02a5c4f1c61fce))
* **decky-plugin:** missing delete in preflight ([70a91c8](https://github.com/RemakeCode/sentinel/commit/70a91c8651403639a26e48d4bf71e9ad4dda9c5c))
* **decky-plugin:** progressBar alignment for locked achievements and add new colour for the active and focus states of the sort menu ([c05c7b0](https://github.com/RemakeCode/sentinel/commit/c05c7b059ea818831f6b0a93d1d642d9f7ef56e1))
* **decky-plugin:** scrolling issues ([74a1eb4](https://github.com/RemakeCode/sentinel/commit/74a1eb4e6b9883ca7353250102992ba99d8e47ae))
* **decky-plugin:** the running state enum and check from the correct object ([85df09f](https://github.com/RemakeCode/sentinel/commit/85df09fb9150a88b904f86143311cf4e6fca0c7d))
* nvidia crashes by removing view-transition animations ([cc45cd9](https://github.com/RemakeCode/sentinel/commit/cc45cd9269fd521a7160079930db117766a130d9))
* nvidia crashes by removing view-transition animations ([57d2f5a](https://github.com/RemakeCode/sentinel/commit/57d2f5a6f9cc4bd5bc0b5af85285105497eeb45a))
* **watcher:** error with watcher panic when when no prefix path ([1eefb14](https://github.com/RemakeCode/sentinel/commit/1eefb14c30fc83cb36dcb96f89a71924c5e44853))

## [1.1.0-beta.2](https://github.com/RemakeCode/sentinel/compare/v1.1.0-beta.1...v1.1.0-beta.2) (2026-06-08)

### 🚀 New Features

* sentinel decky plugin ([662fb1b](https://github.com/RemakeCode/sentinel/commit/662fb1b0b2708442493eaf7ccf90566378c13940))

## [1.1.0-beta.1](https://github.com/RemakeCode/sentinel/compare/v1.0.7...v1.1.0-beta.1) (2026-06-08)

### 🚀 New Features

* Sentinel decky plugin Implementation  ([7b7c3e7](https://github.com/RemakeCode/sentinel/commit/7b7c3e7e5514c6924d32556e91d5b4b75450f676))

## [1.0.8](https://github.com/RemakeCode/sentinel/compare/v1.0.7...v1.0.8) (2026-06-09)

### 🐛 Bug Fixes

* fix crash when a prefix is suddenly removed by another process  ([6306079](https://github.com/RemakeCode/sentinel/commit/63060797488da69d47d0c71de3f4a56b35ea6d04)), closes [#39](https://github.com/RemakeCode/sentinel/issues/39)

## [1.0.7](https://github.com/RemakeCode/sentinel/compare/v1.0.6...v1.0.7) (2026-04-18)

### 🐛 Bug Fixes

* missing image cover ([18c7625](https://github.com/RemakeCode/sentinel/commit/18c7625d58c8597b99afdfdf984ca99f6cfaee7c))

## [1.0.6](https://github.com/RemakeCode/sentinel/compare/v1.0.5...v1.0.6) (2026-04-13)

### 🐛 Bug Fixes

* Nvidia crash issues ([#33](https://github.com/RemakeCode/sentinel/issues/33)) ([59c6569](https://github.com/RemakeCode/sentinel/commit/59c6569bff39004b6902e7946e28a2a7b54ebb69))

## [1.0.5](https://github.com/RemakeCode/sentinel/compare/v1.0.4...v1.0.5) (2026-04-10)

### 🐛 Bug Fixes

* package name mismatch for rpm package ([#30](https://github.com/RemakeCode/sentinel/issues/30)) ([8442c73](https://github.com/RemakeCode/sentinel/commit/8442c73f4cc4309a4f400b5187491c363524ae19))

## [1.0.4](https://github.com/RemakeCode/sentinel/compare/v1.0.3...v1.0.4) (2026-04-09)

### 🐛 Bug Fixes

* Config init issue on Settings page ([#27](https://github.com/RemakeCode/sentinel/issues/27)) ([56e0153](https://github.com/RemakeCode/sentinel/commit/56e0153ce97f97a0814ae0b3d86a34fe1017ffb7))

## [1.0.3](https://github.com/RemakeCode/sentinel/compare/v1.0.2...v1.0.3) (2026-04-09)

### 🐛 Bug Fixes

* Fix issue with settings not initializing properly on first load ([#25](https://github.com/RemakeCode/sentinel/issues/25)) ([fa3f40d](https://github.com/RemakeCode/sentinel/commit/fa3f40d38c66437c5c43ae98b80b35c1a65fabbc))

## [1.0.2](https://github.com/RemakeCode/sentinel/compare/v1.0.1...v1.0.2) (2026-04-07)

### 🐛 Bug Fixes

* Issues with failing builds ([#24](https://github.com/RemakeCode/sentinel/issues/24)) ([986d3cb](https://github.com/RemakeCode/sentinel/commit/986d3cb1fe58ef42ca4d2dd6df4705feb9316e10))

## [1.0.1](https://github.com/RemakeCode/sentinel/compare/v1.0.0...v1.0.1) (2026-04-06)

### 🐛 Bug Fixes

* Prefix selector window should now correctly display hidden folders (dot folders) ([#20](https://github.com/RemakeCode/sentinel/issues/20)) ([395307f](https://github.com/RemakeCode/sentinel/commit/395307f545b4e8e1c85f06acb502926fc4f07b3e))

## 1.0.0 (2026-04-06)
Initial version release
