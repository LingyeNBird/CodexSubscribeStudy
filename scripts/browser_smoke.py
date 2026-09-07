"""Real Go + Vue regression using only local, synthetic, signed submissions."""
import json
import os
from pathlib import Path
import subprocess
import time
import traceback
import urllib.error
import urllib.request
from playwright.sync_api import sync_playwright, expect
from pooled_fixture import packet

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = Path(os.environ.get("STUDY_REVIEW_OUTPUT", ROOT / "review-output")).resolve()
OUTPUT.mkdir(parents=True, exist_ok=True)
BASE = "http://127.0.0.1:18088"


def submit(*args, **kwargs):
    path, body, signature = packet(*args, **kwargs)
    request = urllib.request.Request(BASE + path, data=body, headers={
        "Content-Type": "application/json", "X-Study-Signature": signature})
    with urllib.request.urlopen(request, timeout=10) as response:
        assert response.status == 200
        return json.load(response)


def read():
    with urllib.request.urlopen(BASE + "/api/studies/gpt6-components", timeout=10) as response:
        return json.load(response)


def smoke():
    errors, requests = [], []
    with sync_playwright() as p:
        browser = p.chromium.launch(executable_path=os.environ.get("STUDY_BROWSER_EXECUTABLE") or None)
        page = browser.new_page(viewport={"width": 1440, "height": 1000})
        page.on("pageerror", lambda e: errors.append(str(e)))
        page.on("request", lambda r: requests.append(r.url))
        page.set_default_timeout(15000)
        try:
            page.goto(BASE)
            expect(page.get_by_role("heading", name="正在进行的研究")).to_be_visible()
            expect(page.get_by_text("等待第一份证据")).to_be_visible()
            page.screenshot(path=str(OUTPUT / "study-home-empty.png"), full_page=True)
            page.get_by_role("link", name="进入 GPT-6 研究 ↗").click()
            expect(page.get_by_role("heading", name="第一份证据，还在路上。")).to_be_visible()
            assert read()["totals"]["requests"] == 0
            submit()
            assert submit()["duplicate"]
            result = read()
            assert result["totals"]["contributors"] == result["totals"]["requests"] == 1
            assert all(c["support"] is None for c in result["causes"])
            page.get_by_role("button", name="刷新统计 ↻").click()
            expect(page.get_by_role("heading", name="贡献已收到，等待互补信息。")).to_be_visible()
            # A single installation with two raw intervals may participate in
            # inference without a three-installation or 200-request gate.
            submit(revision=2, informative=True)
            result = read()
            assert result["totals"]["requests"] == 4 and result["totals"]["batches"] == 1
            assert all(c["support"] is not None for c in result["causes"])
            assert abs(sum(c["support"] for c in result["causes"]) - 1) < 1e-10
            page.get_by_role("button", name="刷新统计 ↻").click()
            expect(page.get_by_text("当前支持较多：", exact=False)).to_be_visible()
            expect(page.locator(".cause")).to_have_count(7)
            for width in (1440, 894, 390, 320):
                page.set_viewport_size({"width": width, "height": 980 if width > 600 else 900})
                assert page.evaluate("document.documentElement.scrollWidth <= innerWidth + 1"), width
                page.screenshot(path=str(OUTPUT / f"study-detail-{width}.png"), full_page=True)
            page.get_by_label("选择一个解释族").select_option("input")
            expect(page.locator(".factor-grid")).to_be_visible()
            page.locator(".score-details summary").click()
            expect(page.locator(".score-details table").first.locator("tbody tr")).to_have_count(4)
            expect(page.locator(".score-details table").nth(1).locator("tbody tr")).to_have_count(7)
            page.get_by_role("link", name="研究方法", exact=True).click()
            expect(page.get_by_role("heading", name="哪些数据进入研究？")).to_be_visible()
            page.get_by_role("link", name="隐私与参与", exact=True).click()
            expect(page.get_by_role("heading", name="贡献会长期保留")).to_be_visible()
            assert page.evaluate("document.documentElement.scrollWidth <= innerWidth + 1")
            page.screenshot(path=str(OUTPUT / "study-privacy-mobile.png"), full_page=True)
            submit(revision=3, batch="22222222-2222-4222-8222-222222222222")
            assert read()["totals"]["requests"] == 5
            assert read()["totals"]["batches"] == 2
            for forbidden in ("capacity_context", "auxiliary_evidence", "particle_capacity", "constant_capacity", "ip"):
                try:
                    submit(revision=4, extra={forbidden: 1})
                except urllib.error.HTTPError as e:
                    assert e.code == 400
                else:
                    raise AssertionError("accepted forbidden field: " + forbidden)
            assert read()["totals"]["requests"] == 5
            public = json.dumps(read())
            for forbidden in ("public_key", "batch_id", "log_evidence", "capacity_context", "auxiliary"):
                assert forbidden not in public
            assert not errors, errors
            assert all(url.startswith(BASE) for url in requests), requests
            assert browser.contexts[0].cookies() == []
            (OUTPUT / "browser-results.json").write_text(json.dumps({
                "synthetic_only": True, "passed": True, "page_errors": errors,
                "checks": ["empty state", "one request accepted", "single installation inference", "Python Ed25519 to Go",
                           "2x2 raw profile evidence", "replace not accumulate", "retain separate batches", "seven hypotheses",
                           "four responsive widths", "parameter ranges", "method and privacy", "reject capacity estimates",
                           "durable retention", "no external requests or cookies"]}, indent=2) + "\n")
        except Exception:
            (OUTPUT / "failure.txt").write_text(traceback.format_exc())
            page.screenshot(path=str(OUTPUT / "failure.png"), full_page=True)
            raise
        finally:
            browser.close()


if __name__ == "__main__":
    executable = os.environ.get("STUDY_BINARY", str(ROOT / "study"))
    db = OUTPUT / "isolated-study.db"
    # The test refuses to reuse an existing database; it never deletes a file.
    if db.exists():
        raise RuntimeError("Use an empty STUDY_REVIEW_OUTPUT directory for this isolated test")
    env = {**os.environ, "STUDY_ADDR": "127.0.0.1:18088", "STUDY_DB": str(db)}
    with (OUTPUT / "go-server.log").open("w") as out:
        process = subprocess.Popen([executable], cwd=ROOT, env=env, stdout=out, stderr=out)
        try:
            for _ in range(100):
                try:
                    read()
                    break
                except Exception:
                    time.sleep(.1)
            else:
                raise RuntimeError("Test receiver failed to start")
            smoke()
        finally:
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
