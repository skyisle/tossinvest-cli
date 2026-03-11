# tossinvest-cli

[![GitHub stars](https://img.shields.io/github/stars/JungHoonGhae/tossinvest-cli)](https://github.com/JungHoonGhae/tossinvest-cli/stargazers)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://go.dev/)
[![Status: Beta](https://img.shields.io/badge/status-beta-orange.svg)](https://github.com/JungHoonGhae/tossinvest-cli)
[![CI](https://github.com/JungHoonGhae/tossinvest-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/JungHoonGhae/tossinvest-cli/actions/workflows/ci.yml)

토스증권 웹 세션을 재사용해 조회와 제한된 거래 기능을 터미널에서 다루기 위한 비공식 CLI입니다. 실행 바이너리는 `tossctl`입니다.

> **주의**: 이 프로젝트는 토스증권 공식 제품이 아닙니다. 웹 내부 API는 예고 없이 바뀔 수 있고, 잘못 쓰면 실제 계좌에 영향을 줄 수 있습니다.

## 개요

이 프로젝트는 아래 같은 작업을 CLI에서 다루는 것을 목표로 합니다.

- 계좌 상태를 읽기
- 시세와 미체결 주문을 확인하기
- 조건이 맞으면 주문을 준비하거나 실행하기
- 결과를 다른 시스템으로 넘기기

현재는 조회 기능과 제한된 거래 베타 범위를 중심으로 구현되어 있습니다.

## 주요 용도

예를 들면 이런 용도에 잘 맞습니다.

- 자산 현황을 스크립트에서 주기적으로 확인하기
- 미체결 주문과 시세를 `json`으로 뽑아 다른 시스템에 넘기기
- 조건 계산과 실제 실행 단계를 분리하기
- 자동화 도구가 토스증권 작업을 명령 단위로 호출하기

## 현재 상태

지금 바로 되는 것:

- 브라우저 로그인 기반 세션 저장과 재사용
- 계좌, 포트폴리오, 미체결 주문, 관심종목, 시세 조회
- 제한된 거래 베타
  - `미국주식`
  - `매수`
  - `지정가`
  - `KRW`
  - `비소수점`
- 당일 미체결 주문 취소
- 거래 분석 문서와 민감정보를 정리한 fixture 관리

아직 더 필요한 것:

- `order amend` 재검증
- 즉시 체결 주문 확인 흐름 강화
- `매도`, `시장가`, `국내주식`, `소수점 주문`

## 요구 사항

| Requirement | Notes |
|-------------|-------|
| Go | `>= 1.25` |
| Python | `>= 3.11` |
| Playwright Chromium | `auth login`에 필요 |

## 설치

macOS에서는 Homebrew 설치를 기본 경로로 생각하고 있습니다.

```bash
brew tap JungHoonGhae/tossinvest-cli
brew install tossctl
```

설치 후에는 먼저 환경을 확인합니다.

```bash
tossctl version
tossctl doctor
tossctl auth doctor
```

`auth login`까지 쓰려면 Homebrew Python에 Playwright와 Chromium을 준비해야 합니다.

```bash
PY="$(brew --prefix python@3.11)/bin/python3.11"
"$PY" -m pip install playwright
"$PY" -m playwright install chromium
```

소스에서 직접 빌드하려면:

```bash
git clone https://github.com/JungHoonGhae/tossinvest-cli.git
cd tossinvest-cli
make build

cd auth-helper
python3 -m pip install -e .
python3 -m playwright install chromium
```

## 빠른 시작

```bash
tossctl version
tossctl doctor
tossctl auth doctor
tossctl auth login
tossctl auth status
tossctl quote get TSLL --output json
tossctl account summary --output json
tossctl orders list --output json
```

## 명령 표면

조회:

```bash
tossctl version
tossctl doctor

tossctl auth login
tossctl auth doctor
tossctl auth status
tossctl auth logout

tossctl account list
tossctl account summary
tossctl portfolio positions
tossctl portfolio allocation
tossctl orders list
tossctl watchlist list
tossctl quote get <symbol>
```

거래:

```bash
tossctl order preview
tossctl order place
tossctl order cancel
tossctl order amend
tossctl order permissions status
tossctl order permissions grant --ttl 300
tossctl order permissions revoke
```

현재 live 검증이 끝난 범위:

- `order preview`
- `order place` for `US buy limit / KRW / non-fractional`
- `order cancel` for same-day pending orders

아직 베타로 봐야 하는 범위:

- `order amend`

## 주문 예시

아래는 `TSLL` 1주를 `500원` 지정가로 미리보기한 뒤 실제 주문하는 흐름입니다.

```bash
tossctl order preview \
  --symbol TSLL \
  --market us \
  --side buy \
  --type limit \
  --qty 1 \
  --price 500 \
  --currency-mode KRW \
  --output json

tossctl order permissions grant --ttl 300

tossctl order place \
  --symbol TSLL \
  --market us \
  --side buy \
  --type limit \
  --qty 1 \
  --price 500 \
  --currency-mode KRW \
  --execute \
  --dangerously-skip-permissions \
  --confirm <preview-token> \
  --output json

tossctl orders list --output json
```

## 거래 안전장치

금융 자동화는 다른 앱 자동화와 다릅니다.

- 비공식 API라서 언제든 깨질 수 있음
- 잘못된 판단이 바로 실계좌로 이어질 수 있음
- 환전, 추가 인증, 상품 위험고지 같은 분기가 자동화를 흔들 수 있음
- 조회 자동화보다 주문 자동화의 사고 비용이 훨씬 큼

그래서 이 프로젝트는 일부러 몇 단계 확인을 남겨두고 있습니다.

- `order preview`
- `order permissions grant`
- `--execute`
- `--dangerously-skip-permissions`
- `--confirm`

금융에서는 편의성보다 오작동 방지가 먼저입니다.

## 개발

```bash
make tidy
make fmt
make build
make test
```

`auth-helper`는 브라우저 로그인만 담당합니다. 토스증권 도메인 로직은 Go CLI 쪽에 남겨두고, 브라우저 자동화는 분리해 유지합니다.

## 문서

- [`docs/reverse-engineering/`](docs/reverse-engineering/)
- [`docs/trading/`](docs/trading/)
- [`auth-helper/README.md`](auth-helper/README.md)

## 로컬 저장 경로

- config dir: `$(os.UserConfigDir)/tossctl`
- cache dir: `$(os.UserCacheDir)/tossctl`
- session file: `<config dir>/session.json`
- permission file: `<config dir>/trading-permission.json`

개발 중에는 아래 플래그로 경로를 덮어쓸 수 있습니다.

- `--config-dir`
- `--session-file`

## Support

도움이 되었다면 유지보수에 힘을 보태 주세요.

<a href="https://www.buymeacoffee.com/lucas.ghae">
  <img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" height="50">
</a>

## Contributing

버그 제보와 PR은 환영합니다.

## License

MIT
