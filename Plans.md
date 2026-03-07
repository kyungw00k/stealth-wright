# sw Plans.md

작성일: 2026-03-07

---

## 현황 요약

playwright-cli 커맨드·옵션 대비 sw의 구현 커버리지 조사 결과.

### 커버리지 현황

- Core/Navigation/Keyboard/Mouse/Storage/Tabs/Network/Route/Tracing: **완전 구현**
- Video (video-start / video-stop): **구현 완료** (CDP screencast + ffmpeg, stealth 호환)
- 미구현 명령: `install-browser`, `devtools-start`
- 시맨틱 차이: `show` (playwright-cli = DevTools 열기, sw = 창 포커스)
- 옵션 누락: `open --extension`

---

## Phase 1: playwright-cli parity (구현 가능한 항목만)

| Task | 내용 | 구현 가능성 | Status |
|------|------|------------|--------|
| 1.1 | `install-browser` 명령 추가 (`--browser` 옵션) | ✅ `playwright.Install(&RunOptions{Browsers: []string{browser}})` | cc:완료 |
| 1.2 | `devtools-start` 명령 추가 (CDP F12 시뮬레이션, headed+Chromium 한정) | ⚠️ playwright-go API 없음, CDP 우회 구현 | cc:완료 |
| 1.3 | `show` 시맨틱: playwright-cli와 동일하게 alias로 `devtools-start` 동작 부여 | ⚠️ 1.2 구현 후 연동 | cc:완료 |
| 1.4 | `open --extension` 옵션 | ❌ playwright-go 미지원 (ExtensionContextFactory 미노출) | blocked |

## Phase 1-B: 출력 형식 수정

| Task | 내용 | Status |
|------|------|--------|
| B.1 | `click <ref> [button]` positional arg 추가 | cc:완료 |
| B.2 | `dblclick <ref> [button]` positional arg 추가 | cc:완료 |
| B.3 | Console 출력 조건: errors>0 OR warnings>0 일 때만 출력 | cc:완료 |

## Phase 2: 비디오 레코딩 검증

| Task | 내용 | Status |
|------|------|--------|
| 2.1 | `sw open` → `sw video-start` → 페이지 조작 → `sw video-stop` E2E 동작 확인 | cc:TODO |
| 2.2 | ffmpeg 경로 탐색 (`findPlaywrightFFmpeg`) 실패 시 에러 메시지 개선 | cc:TODO |
| 2.3 | 비디오 통합 테스트 추가 (video-start / video-stop / 파일 생성 확인) | cc:TODO |

---

## 메모

### playwright-go API 조사 결과 (2026-03-07)

| 항목 | API | 결론 |
|------|-----|------|
| `install-browser` | `playwright.Install(&RunOptions{Browsers: []string{...}})` | 완전 구현 가능 |
| `devtools-start` | `Devtools *bool` deprecated, CDP 우회 가능 | 제한적 구현 |
| `open --extension` | `ExtensionContextFactory` 미노출, `Args` 로 `--load-extension` 전달만 가능 | 실질적 불가 |

- 비디오는 ffmpeg 경로가 `ms-playwright/ffmpeg-{revision}/ffmpeg-mac` 구조에 의존 → `sw install` 실행 없이도 동작하는지 확인 필요
- `devtools-start` / `show` 는 playwright-cli가 내부 웹 UI 앱(devtoolsApp.js)을 별도 Chromium 프로세스로 spawn하는 방식 — sw에서 동일 UX 재현 불가, CDP 기반 안내 메시지로 대체 검토
