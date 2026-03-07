# sw (Stealth Wright) Architecture Design

> Silent browser automation CLI with stealth capabilities

## 개요

```
sw = Stealth + Wright (Playwright의 wright 계승)

"은밀하게 브라우저를 다루는 장인"
```

**특징**:
- Playwright CLI와 동일한 UX
- Stealth Mode 기본 내장 (봇 탐지 회피)
- 2글자 명령어 (`sw`)
- AI 친화적 Skill 시스템
- 브라우저 백엔드 교체 가능 (인터페이스 추상화)

---

## 1. 아키텍처 개요

### 1.1 시스템 다이어그램

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              sw (Stealth Wright)                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────┐                    ┌──────────────────────────┐  │
│  │   CLI Client     │                    │      Daemon Server       │  │
│  │                  │                    │                          │  │
│  │  ┌────────────┐  │   Unix Socket      │  ┌────────────────────┐  │  │
│  │  │   cobra    │  │  (JSON-RPC)        │  │   Session Manager  │  │  │
│  │  │  commands  │──┼───────────────────►│  │  ┌──────────────┐  │  │  │
│  │  └────────────┘  │                    │  │  │   Browser    │  │  │  │
│  │                  │                    │  │  │   Instance   │  │  │  │
│  │  ┌────────────┐  │                    │  │  │ ┌──────────┐ │  │  │  │
│  │  │   client   │◄─┼────────────────────│  │  │ │   Page   │ │  │  │  │
│  │  └────────────┘  │                    │  │  │ └──────────┘ │  │  │  │
│  │                  │                    │  │  └──────────────┘  │  │  │
│  └──────────────────┘                    │  │                    │  │  │
│                                          │  │  ┌────────────────┐ │  │  │
│                                          │  │  │ Snapshot Gen.  │ │  │  │
│                                          │  │  └────────────────┘ │  │  │
│                                          │  │                    │  │  │
│                                          │  │  ┌────────────────┐ │  │  │
│                                          │  │  │ Stealth Module │ │  │  │
│                                          │  │  └────────────────┘ │  │  │
│                                          │  └────────────────────┘  │  │
│                                          └──────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                     Browser Abstraction Layer                     │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │  │
│  │  │ interface   │  │  selenium   │  │ playwright  │  (pluggable)  │  │
│  │  │ (abstract)  │  │  base-go    │  │    -go      │               │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘               │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  ~/.sw/                                                                 │
│  ├── sessions/           # 세션 레지스트리                               │
│  │   └── default.json                                                   │
│  ├── profiles/           # 브라우저 프로필 (persistent mode)             │
│  │   └── ud-default/                                                    │
│  └── sockets/            # Unix sockets                                  │
│      └── <hash>/                                                        │
│          └── default.sock                                               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. 브라우저 추상화 레이어

### 2.1 인터페이스 정의

**`internal/browser/interface.go`**:

```go
package browser

import (
    "context"
    "time"
)

// Browser는 브라우저 인스턴스를 나타내는 인터페이스
type Browser interface {
    // Lifecycle
    Close() error
    NewPage(opts ...PageOption) (Page, error)
    NewContext(opts ...ContextOption) (Context, error)
    
    // Info
    Version() string
    IsConnected() bool
}

// Context는 브라우저 컨텍스트 (세션 격리)
type Context interface {
    NewPage(opts ...PageOption) (Page, error)
    Pages() []Page
    Close() error
    
    // Storage
    StorageState() (*StorageState, error)
    SetStorageState(state *StorageState) error
}

// Page는 브라우저 페이지를 나타내는 인터페이스
type Page interface {
    // Navigation
    Goto(url string, opts ...GotoOption) error
    GoBack() error
    GoForward() error
    Refresh() error
    
    // Info
    URL() string
    Title() string
    Content() (string, error)
    
    // Interaction
    Click(selector string, opts ...ClickOption) error
    DblClick(selector string, opts ...ClickOption) error
    Hover(selector string) error
    Type(selector, text string, opts ...TypeOption) error
    Press(selector, key string) error
    Fill(selector, text string) error
    
    // Query
    Query(selector string) (Element, error)
    QueryAll(selector string) ([]Element, error)
    WaitForSelector(selector string, opts ...WaitOption) (Element, error)
    
    // Screenshot
    Screenshot(opts ...ScreenshotOption) ([]byte, error)
    
    // Eval
    Evaluate(script string, args ...any) (any, error)
    
    // Lifecycle
    Close() error
}

// Element는 DOM 요소를 나타내는 인터페이스
type Element interface {
    Click(opts ...ClickOption) error
    Hover() error
    Type(text string, opts ...TypeOption) error
    Fill(text string) error
    
    TextContent() (string, error)
    InnerText() (string, error)
    GetAttribute(name string) (string, error)
    
    IsVisible() bool
    IsEnabled() bool
    IsChecked() bool
    
    BoundingBox() (*Rect, error)
    Screenshot(opts ...ScreenshotOption) ([]byte, error)
}

// Types
type StorageState struct {
    Cookies []Cookie
    Origins []OriginStorage
}

type Cookie struct {
    Name     string
    Value    string
    Domain   string
    Path     string
    Expires  time.Time
    HTTPOnly bool
    Secure   bool
    SameSite string
}

type OriginStorage struct {
    Origin string
    LocalStorage  []StorageEntry
    SessionStorage []StorageEntry
}

type StorageEntry struct {
    Name  string
    Value string
}

type Rect struct {
    X      float64
    Y      float64
    Width  float64
    Height float64
}
```

### 2.2 드라이버 구현

**`internal/drivers/seleniumbase/driver.go`**:

```go
package seleniumbase

import (
    "github.com/kyungw00k/seleniumbase-go/sb"
    "github.com/user/sw/internal/browser"
)

// Driver는 seleniumbase-go를 사용한 Browser 인터페이스 구현
type Driver struct {
    // driver implementation
}

func NewDriver(opts ...Option) (browser.Browser, error) {
    // seleniumbase-go 초기화
}

// ... 인터페이스 구현
```

### 2.3 드라이버 교체 예시

```go
// 현재: seleniumbase-go
import "github.com/user/sw/internal/drivers/seleniumbase"
browser, _ := seleniumbase.NewDriver()

// 미래: playwright-go 직접 사용
import "github.com/user/sw/internal/drivers/playwright"
browser, _ := playwright.NewDriver()

// 미래: rod 사용
import "github.com/user/sw/internal/drivers/rod"
browser, _ := rod.NewDriver()
```

---

## 3. Stealth Module

### 3.1 Stealth 기능

**`internal/stealth/stealth.go`**:

```go
package stealth

// Config는 Stealth 설정
type Config struct {
    // Fingerprint randomization
    UserAgent       string
    Viewport        Viewport
    Screen          Screen
    Timezone        string
    Locale          string
    WebRTC          WebRTCConfig
    Canvas          CanvasConfig
    AudioContext    AudioConfig
    
    // Behavior humanization
    TypingSpeed     time.Duration
    MouseMovement   bool
    RandomDelays    bool
    
    // Detection evasion
    HideWebDriver   bool
    ModifyNavigator bool
    WebGLRenderer   string
}

// Module은 Stealth 기능을 제공
type Module struct {
    config Config
}

func NewModule(opts ...Option) *Module

// Apply는 브라우저 컨텍스트에 Stealth 설정을 적용
func (m *Module) Apply(ctx browser.Context) error

// HumanizeTyping은 인간처럼 타이핑
func (m *Module) HumanizeTyping(page browser.Page, selector, text string) error

// HumanizeClick은 인간처럼 클릭
func (m *Module) HumanizeClick(page browser.Page, selector string) error
```

### 3.2 Stealth 기능 목록

| 기능 | 설명 |
|------|------|
| **Fingerprint Randomization** | 브라우저 핑거프린트 무작위화 |
| **User-Agent Spoofing** | UA 위조 |
| **WebGL Renderer Spoofing** | WebGL 렌더러 정보 위조 |
| **Canvas Fingerprint** | 캔버스 핑거프린트 노이즈 추가 |
| **Audio Fingerprint** | 오디오 핑거프린트 노이즈 |
| **WebRTC Leak Prevention** | WebRTC IP 누출 방지 |
| **Timezone Spoofing** | 타임존 위조 |
| **Hide WebDriver** | navigator.webdriver 숨김 |
| **Human-like Typing** | 인간 같은 타이핑 속도 |
| **Human-like Mouse** | 인간 같은 마우스 이동 |
| **Random Delays** | 무작위 지연 |

---

## 4. CLI 명령어

### 4.1 전역 옵션

```bash
sw [GLOBAL OPTIONS] <command> [ARGS]

Global Options:
  -s, --session <name>    세션 이름 (default: "default")
  -b, --browser <type>    브라우저 타입 (chromium, firefox, webkit)
      --headed            헤드 모드
      --persistent        프로필 디스크에 저장
      --profile <path>    커스텀 프로필 경로
      --config <file>     설정 파일
      --stealth           Stealth mode 활성화 (default: true)
      --no-stealth        Stealth mode 비활성화
  -v, --version           버전 출력
  -h, --help              도움말
```

### 4.2 핵심 명령어

| 명령 | 파라미터 | 설명 |
|------|----------|------|
| `open [url]` | `url?` | 브라우저 시작 |
| `close` | - | 브라우저 종료 |
| `goto <url>` | `url` | URL 이동 |
| `snapshot` | - | Element refs 생성 |
| `click <ref>` | `ref`, `button?` | 클릭 |
| `fill <ref> <text>` | `ref`, `text` | 텍스트 입력 |
| `type <text>` | `text`, `submit?` | 포커스 요소에 입력 |
| `press <key>` | `key` | 키 입력 |
| `hover <ref>` | `ref` | 마우스 오버 |
| `screenshot` | `ref?`, `filename?` | 스크린샷 |

### 4.3 Stealth 명령어

| 명령 | 설명 |
|------|------|
| `stealth on` | Stealth mode 활성화 |
| `stealth off` | Stealth mode 비활성화 |
| `stealth status` | Stealth 설정 표시 |
| `stealth fingerprint` | 현재 핑거프린트 표시 |

### 4.4 Session 관리

| 명령 | 설명 |
|------|------|
| `list` | 활성 세션 목록 |
| `close-all` | 모든 세션 종료 |
| `kill-all` | 강제 종료 (zombie 정리) |
| `delete-data` | 세션 데이터 삭제 |

---

## 5. 디렉토리 구조

```
sw/
├── cmd/
│   └── sw/
│       └── main.go              # CLI 진입점
│
├── internal/
│   ├── browser/                 # 브라우저 인터페이스 (추상화)
│   │   ├── interface.go         # Browser, Page, Element 인터페이스
│   │   ├── types.go             # 공통 타입
│   │   └── options.go           # 옵션 타입들
│   │
│   ├── drivers/                 # 구현체들
│   │   └── seleniumbase/        # seleniumbase-go 구현
│   │       ├── driver.go        # Browser 구현
│   │       ├── page.go          # Page 구현
│   │       ├── element.go       # Element 구현
│   │       └── convert.go       # 타입 변환
│   │
│   ├── client/
│   │   ├── client.go            # Daemon 통신 클라이언트
│   │   └── transport.go         # Unix socket 통신
│   │
│   ├── daemon/
│   │   ├── server.go            # Daemon 서버
│   │   ├── commands.go          # 명령 디스패치
│   │   └── lifecycle.go         # 시작/종료 관리
│   │
│   ├── session/
│   │   ├── manager.go           # 세션 관리
│   │   └── registry.go          # 세션 저장소
│   │
│   ├── snapshot/
│   │   └── generator.go         # Element ref 생성
│   │
│   ├── stealth/                 # Stealth module
│   │   ├── stealth.go           # 메인 모듈
│   │   ├── fingerprint.go       # 핑거프린트 우회
│   │   ├── detection.go         # 봇 탐지 회피
│   │   ├── behavior.go          # 인간 같은 동작
│   │   └── scripts/             # 브라우저 스크립트
│   │       ├── webdriver.js
│   │       ├── navigator.js
│   │       └── webgl.js
│   │
│   ├── commands/                # CLI 명령 구현
│   │   ├── root.go
│   │   ├── open.go
│   │   ├── close.go
│   │   ├── goto.go
│   │   ├── click.go
│   │   ├── fill.go
│   │   ├── snapshot.go
│   │   ├── screenshot.go
│   │   ├── stealth.go
│   │   └── session.go
│   │
│   └── config/
│       └── config.go            # 설정 관리
│
├── pkg/
│   └── protocol/
│       └── types.go             # JSON-RPC 타입
│
├── skills/
│   └── sw/
│       ├── SKILL.md             # AI Skill 파일
│       └── references/
│           ├── session-management.md
│           ├── storage-state.md
│           ├── stealth-mode.md
│           ├── request-mocking.md
│           └── troubleshooting.md
│
├── go.mod
├── go.sum
├── Makefile
├── ARCHITECTURE.md
└── README.md
```

---

## 6. 의존성

```go
// go.mod
module github.com/kyungw00k/sw

go 1.22

require (
    github.com/spf13/cobra v1.8.0           // CLI 프레임워크
    github.com/spf13/viper v1.18.0          // 설정 관리
    
    // 브라우저 백엔드 (현재)
    github.com/kyungw00k/seleniumbase-go v0.x.x
    github.com/playwright-community/playwright-go v0.5700.1
)
```

---

## 7. 구현 현황

### Phase 1: MVP ✅
- [x] 프로젝트 스캐폴딩
- [x] Browser 인터페이스 정의 (`internal/browser/`)
- [x] seleniumbase-go 드라이버 구현 (`internal/drivers/seleniumbase/`)
- [x] 기본 CLI 구조 (cobra, `cmd/sw/main.go`)
- [x] Daemon 서버 (Unix socket JSON-RPC, `internal/daemon/`)
- [x] 핵심 명령: `open`, `close`, `goto`, `snapshot`, `click`, `fill`, `hover`, `check`, `uncheck`, `press`, `type`, `drag`, `select`, `upload`, `eval`, `resize`
- [x] Mouse/Keyboard 명령: `mousemove`, `mousedown`, `mouseup`, `mousewheel`, `keydown`, `keyup`
- [x] Storage: `cookie-*`, `localstorage-*`, `sessionstorage-*`, `state-save`, `state-load`, `delete-data`
- [x] Network: `network`, `console`, `route`, `unroute`, `route-list`
- [x] Tabs: `tab-list`, `tab-new`, `tab-close`, `tab-select`
- [x] Tracing: `tracing-start`, `tracing-stop`
- [x] Session: `list`, `close-all`, `kill-all`, `daemon`

### Phase 2: Stealth Mode ✅
- [x] seleniumbase-go 통합으로 Stealth 기본 활성화
- [x] `--stealth` / `--no-stealth` 플래그
- [x] `SW_STEALTH` 환경변수 지원

### Phase 3: 기능 확장 ✅
- [x] Persistent session (`--persistent`, `--profile`)
- [x] 설정 파일 지원 (`--config`)
- [x] AI Skill 시스템 (`skills/sw/SKILL.md`)
- [x] 디바이스 에뮬레이션 (`--device`, `sw devices`)
- [x] 세션 격리 (named sessions, per-session socket/pid)
- [x] Video 레코딩 (`video-start`, `video-stop` — CDP screencast + ffmpeg → WebM)
- [x] `install-browser` 명령 (playwright-go `Install()` API)
- [x] `devtools-start` / `show` (CDP F12 키 시뮬레이션)
- [x] `pdf` 명령

### Phase 4: AI-Native 기능 ✅
- [x] Semantic Locators (`--role`, `--text`, `--label`, `--placeholder`, `--exact`)
  - `click`, `fill`, `hover`, `check`, `uncheck`, `dblclick` 에서 ref 없이 시맨틱 대상 지정 가능
- [x] `find` 명령 — 현재 스냅샷에서 시맨틱 기준 요소 검색
- [x] Annotated Screenshot (`screenshot --annotate`) — JS overlay로 ref 번호를 페이지에 주입 후 스크린샷
- [x] 통합 테스트 스위트 (`test/integration_test.go`, 40+ 테스트)

### 제한사항 / Blocked
- `open --extension`: playwright-go가 `ExtensionContextFactory` 미노출 → 미구현
- `devtools-start`: playwright-go에 DevTools API 없음 → CDP F12 키 시뮬레이션으로 우회 (headed+Chromium 한정)

---

## 8. 참고 자료

- [Playwright CLI](https://github.com/microsoft/playwright-cli)
- [SeleniumBase-go](https://github.com/kyungw00k/seleniumbase-go)
- [playwright-go](https://github.com/playwright-community/playwright-go)
- [puppeteer-extra-plugin-stealth](https://github.com/berstend/puppeteer-extra/tree/master/packages/puppeteer-extra-plugin-stealth)
- [undetected-chromedriver](https://github.com/ultrafunkamsterdam/undetected-chromedriver)
