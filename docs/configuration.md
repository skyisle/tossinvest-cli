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
    "place": false,
    "sell": false,
    "kr": false,
    "fractional": false,
    "cancel": false,
    "amend": false,
    "allow_live_order_actions": false,
    "dangerous_automation": {
      "accept_fx_consent": false
    }
  }
}
```

## 필드 설명

토글은 두 종류로 나뉩니다.

**경로 게이트 (broker API 분기별)**
- `trading.place` — `tossctl order place` 허용 여부
- `trading.cancel` — `tossctl order cancel` 허용 여부
- `trading.amend` — `tossctl order amend` 허용 여부

**스코프 선언 (유저 자가 제한)**
- `trading.sell` — `tossctl order place --side sell` 허용 여부. `trading.place`도 함께 켜야 합니다. 끄면 매수만 가능
- `trading.kr` — `tossctl order place --market kr` 허용 여부. `trading.place`도 함께 켜야 합니다. 끄면 US only
- `trading.fractional` — `tossctl order place --fractional --amount <KRW>` 허용 여부. US 시장가 주문으로만 지원. `trading.place`도 함께 켜야 합니다

**마스터 / 자동화**
- `trading.allow_live_order_actions` — 실계좌에 도달하는 주문 액션(`place`, `cancel`, `amend`) 자체를 허용. **마스터 킬스위치**로 위 경로 게이트가 켜져 있어도 이 값이 false면 broker에 닿지 않음
- `trading.dangerous_automation.accept_fx_consent` — post-prepare FX confirmation branch를 자동 수락하고 같은 주문을 계속 진행하도록 허용. 현재는 `prepare` 성공 후 `needExchange > 0`인 미국주식 KRW 매수 경로에만 연결됨

즉, 각 액션은 config에서 먼저 열려 있어야 하고, 그 다음에도 기존 실행 게이트를 통과해야 합니다. **`order permissions grant`는 별도 토글 없이 `place`/`cancel`/`amend` 중 하나라도 켜져 있으면 실행 가능합니다** (grant 자체는 TTL 부여만 담당).

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

`v0.4.3`에서 제거된 필드(`trading.grant`, `trading.dangerous_automation.complete_trade_auth`, `trading.dangerous_automation.accept_product_ack`)는 설정에 남아있어도 무시되며, `doctor`의 `legacy_config` 체크에서 감지해 알려줍니다. 해당 필드들은 실제로 어떤 동작도 제어하지 않던 죽은 토글이었습니다.

다만 `config show`와 `doctor`는 새 이름 기준으로 해석해서 보여주고, legacy key를 변환해서 읽고 있으면 그 사실을 따로 알려줍니다.

`order preview`는 거래 기능이 꺼져 있어도 계속 사용할 수 있습니다.

## Schema

설정 파일은 아래 JSON Schema를 기준으로 합니다.

- [`schemas/config.schema.json`](../schemas/config.schema.json)

에디터나 LLM이 이 schema를 기준으로 `config.json`을 생성하거나 수정할 수 있습니다.
