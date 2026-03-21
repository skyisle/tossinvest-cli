# Configuration

`tossctl`은 기본적으로 조회 중심으로 동작합니다. 거래 기능은 설치 직후 바로 열리지 않고, 사용자가 로컬 `config.json`에서 기능별로 직접 허용해야만 사용할 수 있습니다.

## 기본 경로

- config dir: `$(os.UserConfigDir)/tossctl`
- config file: `<config dir>/config.json`

먼저 현재 설정을 확인하거나 기본 파일을 만들 수 있습니다.

```bash
tossctl config show
tossctl config init
```

## 기본 설정

파일이 없더라도 CLI는 동작하지만, 이 경우 거래 기능은 모두 비활성으로 간주됩니다.

`tossctl config init`으로 생성되는 기본 파일은 아래와 같습니다.

```json
{
  "$schema": "https://raw.githubusercontent.com/JungHoonGhae/tossinvest-cli/main/schemas/config.schema.json",
  "schema_version": 2,
  "trading": {
    "grant": false,
    "place": false,
    "sell": false,
    "kr": false,
    "fractional": false,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": false,
    "dangerous_automation": {
      "complete_trade_auth": false,
      "accept_product_ack": false,
      "accept_fx_consent": false
    }
  }
}
```

## 필드 설명

- `trading.grant`
  - `tossctl order permissions grant` 허용 여부
- `trading.place`
  - `tossctl order place` 허용 여부
- `trading.sell`
  - `tossctl order place --side sell` 허용 여부
  - `trading.place`도 함께 켜야 매도 주문이 가능합니다
- `trading.kr`
  - `tossctl order place --market kr` 허용 여부
  - `trading.place`도 함께 켜야 국내주식 주문이 가능합니다
- `trading.fractional`
  - `tossctl order place --fractional --amount <KRW>` 허용 여부
  - US 시장가 주문으로만 지원됩니다 (소수점 주문은 금액 기반)
  - `trading.place`도 함께 켜야 소수점 주문이 가능합니다
- `trading.cancel`
  - `tossctl order cancel` 허용 여부
- `trading.amend`
  - `tossctl order amend` 허용 여부
- `trading.allow_live_order_actions`
  - 실계좌 주문 액션(`place`, `cancel`, `amend`) 자체를 허용할지
- `trading.dangerous_automation.complete_trade_auth`
  - trade auth 분기를 자동 완료하도록 허용할지
  - 해당 분기 handler가 구현된 빌드에서만 실제로 효과가 있습니다
- `trading.dangerous_automation.accept_product_ack`
  - product acknowledgement 분기를 자동 수락하도록 허용할지
  - 해당 분기 handler가 구현된 빌드에서만 실제로 효과가 있습니다
- `trading.dangerous_automation.accept_fx_consent`
  - post-prepare FX confirmation branch를 자동 수락하고 같은 주문을 계속 진행하도록 허용할지
  - 현재는 `prepare` 성공 후 `needExchange > 0`인 미국주식 KRW 매수 경로에만 연결됩니다

즉, 각 액션은 config에서 먼저 열려 있어야 하고, 그 다음에도 기존 실행 게이트를 통과해야 합니다.

## 실행 순서

거래 mutation이 실제로 실행되려면 아래 순서를 모두 만족해야 합니다.

1. `config.json`에서 해당 액션 허용
   - live mutation은 `trading.allow_live_order_actions=true`도 필요
2. `tossctl order permissions grant`
3. `--execute`
4. `--dangerously-skip-permissions`
5. `--confirm`

## Legacy Compatibility

기존 `schema_version: 1` 파일과 `trading.allow_dangerous_execute`는 계속 읽을 수 있습니다.

다만 `config show`와 `doctor`는 새 이름 기준으로 해석해서 보여주고, legacy key를 변환해서 읽고 있으면 그 사실을 따로 알려줍니다.

`order preview`는 거래 기능이 꺼져 있어도 계속 사용할 수 있습니다.

## Schema

설정 파일은 아래 JSON Schema를 기준으로 합니다.

- [`schemas/config.schema.json`](../schemas/config.schema.json)

에디터나 LLM이 이 schema를 기준으로 `config.json`을 생성하거나 수정할 수 있습니다.
