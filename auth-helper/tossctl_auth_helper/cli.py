from __future__ import annotations

import argparse
import json
import sys
import time
from pathlib import Path


DEFAULT_URL = "https://www.tossinvest.com/account"
REQUIRED_COOKIE_NAMES = {
    "SESSION",
    "XSRF-TOKEN",
    "UTK",
    "LTK",
    "FTK",
    "browserSessionId",
}
# Toss's current web app still sets WTS-DEVICE-ID and login-method after a
# successful QR login, but DEVICE_INFO is no longer guaranteed to appear.
REQUIRED_LOCAL_STORAGE_KEYS = {
    "WTS-DEVICE-ID",
    "login-method",
}

# When the user confirms "이 기기 로그인 유지" on their phone after QR auth,
# Toss calls POST /api/v1/wts-login-device/check-with-login, whose response
# re-issues the SESSION cookie with Max-Age=31536000 (1 year). If we save
# storage state before that second confirmation, the SESSION is only
# session-scoped (expires=-1) and the server invalidates it after ≈1h idle.
# We wait for the persistent cookie before saving so the CLI inherits the
# long-lived session.
PERSISTENT_SESSION_MIN_TTL_SECONDS = 7 * 24 * 3600  # > 1 week out = definitely persistent


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="python -m tossctl_auth_helper",
        description="Browser-assisted login helper for tossctl.",
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    login_parser = subparsers.add_parser(
        "login",
        help="Open a real browser and capture Toss Securities storage state.",
    )
    login_parser.add_argument(
        "--storage-state",
        required=True,
        help="Path where Playwright storage state JSON will be written.",
    )
    login_parser.add_argument(
        "--url",
        default=DEFAULT_URL,
        help=f"Login entry URL. Defaults to {DEFAULT_URL}.",
    )
    login_parser.add_argument(
        "--timeout-seconds",
        type=int,
        default=300,
        help="How long to wait for the user to complete login.",
    )

    return parser


def emit(payload: dict) -> int:
    json.dump(payload, sys.stdout, indent=2)
    sys.stdout.write("\n")
    sys.stdout.flush()
    return 0


def log(message: str) -> None:
    sys.stderr.write(message + "\n")
    sys.stderr.flush()


def local_storage_keys(storage_state: dict) -> set[str]:
    for origin in storage_state.get("origins", []):
        if origin.get("origin") != "https://www.tossinvest.com":
            continue
        return {item["name"] for item in origin.get("localStorage", [])}
    return set()


def has_persistent_session_cookie(storage_state: dict) -> bool:
    threshold = time.time() + PERSISTENT_SESSION_MIN_TTL_SECONDS
    for cookie in storage_state.get("cookies", []):
        if cookie.get("name") != "SESSION":
            continue
        expires = cookie.get("expires")
        if isinstance(expires, (int, float)) and expires >= threshold:
            return True
    return False


def command_login(args: argparse.Namespace) -> int:
    try:
        from playwright.sync_api import Error as PlaywrightError
        from playwright.sync_api import sync_playwright
    except ImportError:
        return emit(
            {
                "status": "error",
                "message": "python package 'playwright' is required. Install it with 'pip install playwright'. Google Chrome must also be installed on your system.",
            }
        )

    storage_state_path = Path(args.storage_state).expanduser().resolve()
    storage_state_path.parent.mkdir(parents=True, exist_ok=True)

    try:
        with sync_playwright() as playwright:
            browser = playwright.chromium.launch(headless=False, channel="chrome")
            context = browser.new_context()
            page = context.new_page()
            log("Opened browser for Toss Securities login.")
            page.goto(args.url, wait_until="domcontentloaded")
            log("Complete QR login in the browser window. The helper will continue after account cookies appear.")

            deadline = time.monotonic() + args.timeout_seconds
            initial_auth_notified = False
            while time.monotonic() < deadline:
                storage_state = context.storage_state()
                cookies = {cookie["name"]: cookie["value"] for cookie in storage_state.get("cookies", [])}
                storage_keys = local_storage_keys(storage_state)
                current_url = page.url

                initial_auth_done = (
                    REQUIRED_COOKIE_NAMES.issubset(cookies.keys())
                    and REQUIRED_LOCAL_STORAGE_KEYS.issubset(storage_keys)
                    and "signin" not in current_url
                )

                if initial_auth_done and not initial_auth_notified:
                    log(
                        "First-step login captured. "
                        "Now confirm '이 기기 로그인 유지' on your phone "
                        "to obtain the persistent (long-lived) session."
                    )
                    initial_auth_notified = True

                if initial_auth_done and has_persistent_session_cookie(storage_state):
                    page.wait_for_timeout(1500)
                    storage_state = context.storage_state()
                    final_cookie_count = len(storage_state.get("cookies", []))
                    storage_state_path.write_text(
                        json.dumps(storage_state, indent=2),
                        encoding="utf-8",
                    )
                    browser.close()
                    return emit(
                        {
                            "status": "ok",
                            "message": (
                                "Captured authenticated Toss Securities storage state "
                                "with persistent SESSION cookie."
                            ),
                            "storage_state_path": str(storage_state_path),
                            "cookie_count": final_cookie_count,
                            "origin_count": len(storage_state.get("origins", [])),
                        }
                    )

                page.wait_for_timeout(1000)

            browser.close()
            if not initial_auth_notified:
                timeout_message = (
                    f"Timed out after {args.timeout_seconds} seconds. "
                    "QR login was not completed — scan the QR code in the browser window first."
                )
            else:
                timeout_message = (
                    f"Timed out after {args.timeout_seconds} seconds. "
                    "QR scan completed but no persistent SESSION was captured. "
                    "Make sure you confirmed '이 기기 로그인 유지' on your phone."
                )
            return emit({"status": "error", "message": timeout_message})
    except PlaywrightError as exc:
        message = str(exc)
        if "Executable doesn't exist" in message:
            message = (
                "Google Chrome is not installed or could not be found. "
                "Install Chrome from https://www.google.com/chrome/ and try again."
            )
        return emit({"status": "error", "message": message})


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "login":
        result = command_login(args)
        if result != 0:
            return result
        return 0

    return emit(
        {
            "status": "error",
            "message": f"Unsupported command: {args.command}",
        }
    )
