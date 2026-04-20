from __future__ import annotations

import argparse
import base64
import json
import re
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
# DEVICE_INFO is no longer guaranteed after QR login — see auth-notes.md.
REQUIRED_LOCAL_STORAGE_KEYS = {
    "WTS-DEVICE-ID",
    "login-method",
}

QR_TAB_SEGMENT_SELECTOR = (
    '[data-tossinvest-log="SegmentedControlButton"][data-parent-name="TossCert"]'
)
QR_TAB_LABEL_EXACT = "QR코드로 로그인"
QR_TAB_LABEL_PATTERN = re.compile(r"QR", re.IGNORECASE)

QR_API_PATHS = (
    "/api/v2/login/wts/toss/qr",
    "/api/v2/login/wts/toss/status",
)

# The signin page renders the QR as the only inline base64 PNG, so this
# selector works regardless of the Korean alt attribute ("큐알코드").
QR_IMG_SELECTOR = 'img[src^="data:image/png;base64"]'


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
    login_parser.add_argument(
        "--headless",
        action="store_true",
        help="Run Chrome headless (required for remote/CLI-only login).",
    )
    login_parser.add_argument(
        "--qr-output",
        default=None,
        help="Path to write the current QR PNG (forward to phone via messenger).",
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


def activate_qr_tab(page, qr_state=None) -> bool:
    # Ordered from most to least locale-independent:
    #   1. data-* attributes that Toss uses for analytics (stable across UI copy)
    #   2. Korean label match (current fallback)
    #   3. Generic "QR" regex (last resort)
    candidates = [
        ("segment[data-parent=TossCert]#1", lambda: page.locator(QR_TAB_SEGMENT_SELECTOR).nth(1)),
        ("label-exact", lambda: page.get_by_role("button", name=QR_TAB_LABEL_EXACT).first),
        ("label-regex", lambda: page.get_by_role("button", name=QR_TAB_LABEL_PATTERN).first),
    ]
    for name, factory in candidates:
        try:
            locator = factory()
            locator.wait_for(state="visible", timeout=3000)
            locator.click(timeout=2000)
        except Exception:
            continue

        # Verify the click actually activated QR: the base64 QR image should
        # render, or the API handler should have stashed a qrCode.
        try:
            page.locator(QR_IMG_SELECTOR).first.wait_for(state="visible", timeout=3000)
            log(f"Activated QR login tab via {name}.")
            return True
        except Exception:
            if qr_state is not None and qr_state.qr_code is not None:
                log(f"Activated QR login tab via {name} (confirmed by API).")
                return True
            continue

    log("Could not confirm QR tab activation — falling through.")
    return False


def decode_qr_url(page) -> str | None:
    try:
        return page.evaluate(
            """
            async () => {
              if (typeof BarcodeDetector === 'undefined') return null;
              const img = document.querySelector('img[src^="data:image/png;base64"]');
              if (!img) return null;
              try { await img.decode(); } catch (e) {}
              const detector = new BarcodeDetector({formats: ['qr_code']});
              const bitmap = await createImageBitmap(img);
              const codes = await detector.detect(bitmap);
              return codes.length ? codes[0].rawValue : null;
            }
            """
        )
    except Exception:
        return None


def save_qr_png(data_uri: str, output_path: Path) -> bool:
    try:
        _, _, b64 = data_uri.partition(",")
        if not b64:
            return False
        output_path.write_bytes(base64.b64decode(b64))
        return True
    except Exception as exc:
        log(f"Failed to save QR PNG: {exc}")
        return False


class QRState:
    def __init__(self, qr_output: Path | None):
        self.qr_output = qr_output
        self.qr_code: str | None = None
        self.answer_letter: str | None = None
        self.url: str | None = None
        self.qr_status: str | None = None
        self.user_status: str | None = None
        self.pending_decode = False

    def handle_api_payload(self, result: dict) -> None:
        new_qr = result.get("qrCode")
        new_letter = result.get("answerLetter")
        new_qr_status = result.get("qrStatus")
        new_user_status = result.get("userStatus")

        if isinstance(new_qr, str) and new_qr != self.qr_code:
            self.qr_code = new_qr
            if self.qr_output is not None and save_qr_png(new_qr, self.qr_output):
                log(f"QR code saved to {self.qr_output}")
            self.pending_decode = True

        if isinstance(new_letter, str) and new_letter != self.answer_letter:
            self.answer_letter = new_letter
            log(
                f"Answer letter: '{new_letter}' — select this on your phone after scanning the QR."
            )

        status_key = (new_qr_status, new_user_status)
        if status_key != (self.qr_status, self.user_status) and any(status_key):
            self.qr_status = new_qr_status
            self.user_status = new_user_status
            log(f"Login status: qr={new_qr_status} user={new_user_status}")

    def flush_decode(self, page) -> None:
        # Playwright sync API forbids evaluate() inside response callbacks —
        # defer the decode to the main loop.
        if not self.pending_decode:
            return
        self.pending_decode = False
        decoded = decode_qr_url(page)
        if not decoded:
            return
        if decoded == self.url:
            return
        self.url = decoded
        log(f"QR URL: {decoded}")


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

    qr_output_path: Path | None = None
    if args.qr_output:
        qr_output_path = Path(args.qr_output).expanduser().resolve()
        qr_output_path.parent.mkdir(parents=True, exist_ok=True)

    qr_state = QRState(qr_output_path)

    try:
        with sync_playwright() as playwright:
            browser = playwright.chromium.launch(
                headless=args.headless,
                channel="chrome",
            )
            context = browser.new_context()
            page = context.new_page()

            def on_response(response):
                path = ""
                try:
                    path = response.url.split("?", 1)[0]
                except Exception:
                    return
                if not any(path.endswith(p) for p in QR_API_PATHS):
                    return
                try:
                    payload = response.json()
                except Exception:
                    return
                if not isinstance(payload, dict):
                    return
                result = payload.get("result")
                if isinstance(result, dict):
                    qr_state.handle_api_payload(result)

            page.on("response", on_response)

            log("Opened browser for Toss Securities login.")
            page.goto(args.url, wait_until="domcontentloaded")

            qr_mode = args.headless or qr_output_path is not None
            if qr_mode:
                activate_qr_tab(page, qr_state)
                page.wait_for_timeout(500)
            else:
                log("Complete QR login in the browser window. The helper will continue after account cookies appear.")

            deadline = time.monotonic() + args.timeout_seconds
            while time.monotonic() < deadline:
                storage_state = context.storage_state()
                cookies = {cookie["name"]: cookie["value"] for cookie in storage_state.get("cookies", [])}
                storage_keys = local_storage_keys(storage_state)
                current_url = page.url

                if (
                    REQUIRED_COOKIE_NAMES.issubset(cookies.keys())
                    and REQUIRED_LOCAL_STORAGE_KEYS.issubset(storage_keys)
                    and "signin" not in current_url
                ):
                    page.wait_for_timeout(1500)
                    storage_state = context.storage_state()
                    storage_state_path.write_text(
                        json.dumps(storage_state, indent=2),
                        encoding="utf-8",
                    )
                    browser.close()
                    return emit(
                        {
                            "status": "ok",
                            "message": "Captured authenticated Toss Securities storage state.",
                            "storage_state_path": str(storage_state_path),
                            "cookie_count": len(cookies),
                            "origin_count": len(storage_state.get("origins", [])),
                        }
                    )

                qr_state.flush_decode(page)
                page.wait_for_timeout(1000)

            browser.close()
            return emit(
                {
                    "status": "error",
                    "message": (
                        f"Timed out after {args.timeout_seconds} seconds waiting for login to complete. "
                        "A full authenticated web session was not observed."
                    ),
                }
            )
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
