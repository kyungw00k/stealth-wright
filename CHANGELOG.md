# Changelog

## [0.2.0](https://github.com/kyungw00k/stealth-wright/compare/v0.1.0...v0.2.0) (2026-03-07)


### Features

* add check command for checkbox/radio buttons ([0f1bb8c](https://github.com/kyungw00k/stealth-wright/commit/0f1bb8cc45ad309b581f8a1ec83f8a077d87bf28))
* add missing CLI flags for playwright-cli parity ([e5a0387](https://github.com/kyungw00k/stealth-wright/commit/e5a03878a723d66a57587e3239e942829d26833e))
* add RecordVideo support to driver and session layers ([53ebf8e](https://github.com/kyungw00k/stealth-wright/commit/53ebf8e1e270f78b9c6bf6ae645c8cdae6ba3ac5))
* add SnapshotParams type to protocol ([5d3aaf3](https://github.com/kyungw00k/stealth-wright/commit/5d3aaf3b386f483db787efbf3a0576afe9679afa))
* **cli:** add devices command, env vars, session isolation, and install --skills ([9a258a8](https://github.com/kyungw00k/stealth-wright/commit/9a258a882bd7d5f012d3cb546003966e44e9b2e1))
* **cli:** add install-browser, devtools-start, and playwright-cli parity fixes ([570f470](https://github.com/kyungw00k/stealth-wright/commit/570f470ef2c5c2c33dadd497c699ebfb648f4e82))
* **cli:** add semantic locators, find command, and annotated screenshot ([2b7e83c](https://github.com/kyungw00k/stealth-wright/commit/2b7e83cef5f9d369bfd18a5084c76e9d5a272193))
* **cli:** add video test, pid output, and ffmpeg error message improvements ([372a798](https://github.com/kyungw00k/stealth-wright/commit/372a798ee85d4be14ca125cb73714fb10daf79d9))
* **daemon:** add playwright-cli-style output and improve command behaviors ([0fdbb37](https://github.com/kyungw00k/stealth-wright/commit/0fdbb376749331005c2b5dffcc6d49f0b4710dbc))
* **device:** add mobile device emulation support ([1dc716f](https://github.com/kyungw00k/stealth-wright/commit/1dc716f5df474fef5acc939bf776f24e7104c0ed))
* implement all playwright-cli commands and CI integration tests ([740e78a](https://github.com/kyungw00k/stealth-wright/commit/740e78ac7d603ce4f0e3217a000ccc3161c8b0f8))
* implement playwright-cli feature parity and improve snapshot format ([bf3a883](https://github.com/kyungw00k/stealth-wright/commit/bf3a883593d3a23dfcd24e739e876fc1445c4fc7))
* implement video recording, snapshot/screenshot enhancements, and modifier support ([285417f](https://github.com/kyungw00k/stealth-wright/commit/285417fc276681a82cbd82b1f8293f6864b5b354))
* initial project implementation ([0321187](https://github.com/kyungw00k/stealth-wright/commit/032118770d71989a4d85bdaa210ea100bb1b633c))
* **video:** replace RecordVideo with CDP screencast for stealth-compatible recording ([3d237ff](https://github.com/kyungw00k/stealth-wright/commit/3d237ffc45153b5a762dc9832fc34a38bc7f07d6))


### Bug Fixes

* detach daemon process from parent ([d159b5c](https://github.com/kyungw00k/stealth-wright/commit/d159b5c09cd41c36c3250879450c2bbe04909731))
* improve snapshot element capture and test robustness ([2d30f0e](https://github.com/kyungw00k/stealth-wright/commit/2d30f0e7dbdff462eedb56125370ae0133b83cfa))
* panic in DefaultSocketPath with short paths ([2795104](https://github.com/kyungw00k/stealth-wright/commit/2795104813f3b85f96df01ba32ba658dbc830d72))
* remove unsupported syscall attributes for macOS ([d719c0f](https://github.com/kyungw00k/stealth-wright/commit/d719c0f9d3cd2730dab4c8e0d8b01448e54cad17))
* skip slow browser detection tests ([c2bba36](https://github.com/kyungw00k/stealth-wright/commit/c2bba369de6b91a9bcd52504c835c0da4305c9c8))
* **video:** fix video recording and save output files to client directory ([e5a3313](https://github.com/kyungw00k/stealth-wright/commit/e5a3313650b23c108329cfc1ddb6be3034c25c5d))


### Documentation

* rewrite SKILL.md to reflect current sw implementation ([60c4cfe](https://github.com/kyungw00k/stealth-wright/commit/60c4cfe01ed825f3e383e016cd337b0cf4132d25))
* **skill:** update SKILL.md for device emulation, env vars, and sessions ([e164c5b](https://github.com/kyungw00k/stealth-wright/commit/e164c5b651bff06c1028e16ad0db3669577993e2))
* update README to reflect current implementation ([5dd42ad](https://github.com/kyungw00k/stealth-wright/commit/5dd42adb8adf9fe56ae1d5de322969a770e97561))
* update README, SKILL.md, and ARCHITECTURE.md to reflect current implementation ([451dbfc](https://github.com/kyungw00k/stealth-wright/commit/451dbfc821909d1e4eccf26993e1c5da775172de))
