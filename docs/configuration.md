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
  "schema_version": 1,
  "trading": {
    "grant": false,
    "place": false,
    "cancel": false,
    "amend": false,
    "allow_dangerous_execute": false
  }
}
```

## 필드 설명

- `trading.grant`
  - `tossctl order permissions grant` 허용 여부
- `trading.place`
  - `tossctl order place` 허용 여부
- `trading.cancel`
  - `tossctl order cancel` 허용 여부
- `trading.amend`
  - `tossctl order amend` 허용 여부
- `trading.allow_dangerous_execute`
  - `--dangerously-skip-permissions` 자체를 사용할 수 있는지

즉, 각 액션은 config에서 먼저 열려 있어야 하고, 그 다음에도 기존 실행 게이트를 통과해야 합니다.

## 실행 순서

거래 mutation이 실제로 실행되려면 아래 순서를 모두 만족해야 합니다.

1. `config.json`에서 해당 액션 허용
2. `tossctl order permissions grant`
3. `--execute`
4. `--dangerously-skip-permissions`
5. `--confirm`

`order preview`는 거래 기능이 꺼져 있어도 계속 사용할 수 있습니다.

## Schema

설정 파일은 아래 JSON Schema를 기준으로 합니다.

- [`schemas/config.schema.json`](../schemas/config.schema.json)

에디터나 LLM이 이 schema를 기준으로 `config.json`을 생성하거나 수정할 수 있습니다.
