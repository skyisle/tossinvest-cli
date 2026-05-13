# Operations

운영 측면의 가이드 — API 회귀 감시 cron 설정, 알림 채널, 백업 정책 등.

## API 회귀 감시 (`tossctl monitor api`)

토스 웹 API는 예고 없이 변경됩니다. 과거에 두 번 user-facing 회귀가 있었습니다:

- [#15 / #17](https://github.com/JungHoonGhae/tossinvest-cli/issues/15) — User-Agent 핑거프린팅 차단 (v0.3.6에서 fix)
- [#29](https://github.com/JungHoonGhae/tossinvest-cli/issues/29) — `/sections/all` body 계약 변경 (v0.4.8에서 fix)

`monitor api` 명령은 6개 read-only endpoint를 schema-invariant probe로 호출해서 위와 같은 변경을 사용자보다 먼저 감지합니다.

### 동작 원리

```
[당신 머신: tossctl monitor api]
       ↓ (당신 세션 쿠키)
[토스 서버] ← 본인 계좌 조회
       ↓ (응답)
[당신 머신: 응답 schema 체크]
       ↓ (실패 시만)
[당신이 설정한 Discord webhook] → 당신 채널
```

- **본인 세션, 본인 계좌만**: `~/Library/Application Support/tossctl/session.json` 의 쿠키 사용. 다른 사용자 데이터는 어디에도 흐르지 않습니다.
- **webhook 페이로드 PII 차단**: 응답 본문은 webhook 송출에 포함되지 않음. endpoint 이름·HTTP 상태·schema 진단 메시지 (예: "result.sections is empty") 만 흐름. 단위 테스트로 강제 (`TestProbeChecksDoNotLeakResponseBodyOnFailure`).

### Probe 목록

| 이름 | Endpoint | 보호하는 명령 |
| --- | --- | --- |
| `account-list` | `GET /api/v1/account/list` | `account list` |
| `account-summary-overview` | `GET /api/v3/my-assets/summaries/markets/all/overview` | `account summary` |
| `portfolio-positions` | `POST /api/v2/dashboard/asset/sections/all` (`SORTED_OVERVIEW`) | `portfolio positions` |
| `watchlist` | `POST /api/v2/dashboard/asset/sections/all` (`WATCHLIST`) | `watchlist list` |
| `quote-stock-infos` | `GET /api/v2/stock-infos/A005930` | `quote get` |
| `pending-orders` | `GET /api/v1/trading/orders/histories/all/pending` | `orders list` |

각 probe는 status 200 + 핵심 JSON 경로 존재 + 타입 검증을 합니다. Toss가 새 필드를 추가하는 것은 무시(false positive 회피), 핵심 필드가 사라지거나 빈 응답을 받으면 실패.

### Discord webhook 설정

1. Discord 채널 → 설정 → 통합 → Webhook → 새 Webhook 생성
2. URL 복사 (`https://discord.com/api/webhooks/.../...`)
3. 환경변수 또는 플래그로 전달 (코드 / 환경 변수 파일에 commit 금지)

```bash
# 일회성
TOSSCTL_MONITOR_WEBHOOK=https://discord.com/api/webhooks/... tossctl monitor api

# 또는 ~/.zshrc 같은 dotfile에 export (개인 머신, git에 안 들어가는 곳)
export TOSSCTL_MONITOR_WEBHOOK=https://discord.com/api/webhooks/...
tossctl monitor api
```

### Cron 예시 (macOS / Linux)

`crontab -e` 에 추가:

```cron
# 매시간 정각마다 monitor api 실행, 실패만 webhook 알림
0 * * * * TOSSCTL_MONITOR_WEBHOOK=https://discord.com/api/webhooks/... /usr/local/bin/tossctl monitor api --quiet >/dev/null 2>&1
```

또는 launchd / systemd timer 로 대체 가능. `monitor api` 는 실패 시 exit 1 을 반환하므로 어떤 스케줄러든 호환됩니다.

### 출력 예시

정상 (모든 probe 통과):

```
  ✓ account-list — status=200 (43ms)
  ✓ account-summary-overview — status=200 (53ms)
  ✓ portfolio-positions — status=200 (52ms)
  ✓ watchlist — status=200 (16ms)
  ✓ quote-stock-infos — status=200 (44ms)
  ✓ pending-orders — status=200 (19ms)

6 passed, 0 failed
```

실패 (예: #29 같은 body-contract 회귀):

```
  ✓ account-list — status=200 (43ms)
  ✓ account-summary-overview — status=200 (53ms)
  ✓ watchlist — status=200 (15ms)
  ✓ quote-stock-infos — status=200 (44ms)
  ✓ pending-orders — status=200 (19ms)
  ✗ portfolio-positions — status=200: result.sections is empty — likely body-contract regression (#29-class)

5 passed, 1 failed
```

Discord 알림 페이로드 (정확히 이게 webhook으로 송출됨, 그 이상 없음):

```
🚨 tossctl API regression detected (0.4.9)
2026-05-13 10:00 UTC — 1/6 probes failed

❌ portfolio-positions — POST wts-cert-api.tossinvest.com/api/v2/dashboard/asset/sections/all
    status=200, result.sections is empty — likely body-contract regression (#29-class)
```

### 새 probe 추가

새 read-only endpoint 의존이 생기면 `internal/monitor/probes.go` 의 `Probes()` 반환 슬라이스에 항목 추가:

```go
{
    Name:   "new-endpoint",
    Method: "POST",
    URL:    cert + "/api/v2/...",
    Body:   `{"types":["..."]}`,
    Check: func(status int, body []byte) error {
        if err := expectStatus(status, body, 200); err != nil {
            return err
        }
        return expectPath(body, "result.someKey", "array")
    },
},
```

검사 작성 시 주의: 에러 메시지에 응답 본문 fragment 를 직접 박지 말 것 (PII 유출). `expectStatus` / `expectPath` 는 안전하게 설계됨. 커스텀 검사는 `TestProbeChecksDoNotLeakResponseBodyOnFailure` 가 SHA-of-shape 마커로 가드합니다.
