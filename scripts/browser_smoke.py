"""Real Go + Vue browser checks with SYNTHETIC signed submissions only."""
import base64
import json
import os
from pathlib import Path
import subprocess
import time
import traceback
import urllib.request
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization
from playwright.sync_api import sync_playwright, expect

ROOT=Path(__file__).resolve().parents[1]
OUTPUT=Path(os.environ.get('STUDY_REVIEW_OUTPUT',ROOT/'review-output')).resolve()
OUTPUT.mkdir(parents=True,exist_ok=True)
BASE='http://127.0.0.1:18088'

def body(seed=80,revision=1,withdraw=False):
    fixture=json.loads((ROOT/'protocol/testdata/synthetic-report.json').read_text())
    report=json.loads(base64.b64decode(fixture['body_base64']))
    key=Ed25519PrivateKey.from_private_bytes(bytes([seed])*32)
    report['public_key']=base64.b64encode(key.public_key().public_bytes(serialization.Encoding.Raw,serialization.PublicFormat.Raw)).decode()
    report['revision']=revision
    if withdraw: report.pop('summary')
    raw=json.dumps(report,sort_keys=True,separators=(',',':')).encode()
    path='/api/v1/withdraw' if withdraw else '/api/v1/reports'
    sig=base64.b64encode(key.sign(b'CodexSubscribeStudy/1\nPOST\n'+path.encode()+b'\n'+raw)).decode()
    return path,raw,sig

def submit(seed=80,revision=1,withdraw=False):
    path,raw,sig=body(seed,revision,withdraw)
    request=urllib.request.Request(BASE+path,data=raw,headers={'Content-Type':'application/json','X-Study-Signature':sig})
    with urllib.request.urlopen(request,timeout=5) as r:
        assert r.status==200
        return json.load(r)

def read():
    with urllib.request.urlopen(BASE+'/api/v1/studies/gpt6-components',timeout=5) as r: return json.load(r)

def check_width(page,width):
    page.set_viewport_size({'width':width,'height':980 if width>600 else 900})
    assert page.evaluate('document.documentElement.scrollWidth <= innerWidth + 1'),f'overflow at {width}'
    page.screenshot(path=str(OUTPUT/f'study-detail-{width}.png'),full_page=True)


def smoke():
    errors=[];requests=[]
    with sync_playwright() as p:
        browser=p.chromium.launch(executable_path=os.environ.get('STUDY_BROWSER_EXECUTABLE') or None)
        page=browser.new_page(viewport={'width':1440,'height':1000})
        page.on('pageerror',lambda e:errors.append(str(e)))
        page.on('request',lambda r:requests.append(r.url))
        page.set_default_timeout(15000)
        try:
            page.goto(BASE)
            expect(page.get_by_role('heading',name='正在进行的研究')).to_be_visible()
            expect(page.get_by_text('等待第一份证据')).to_be_visible()
            page.screenshot(path=str(OUTPUT/'study-home-empty.png'),full_page=True)
            page.get_by_role('link',name='进入 GPT-6 研究').click()
            expect(page.get_by_role('heading',name='第一份证据，还在路上。')).to_be_visible()
            assert read()['totals']['requests']==0
            submit(80);submit(81)
            page.get_by_role('button',name='刷新统计 ↻').click()
            expect(page.get_by_role('heading',name='样本在积累，结论不抢跑。')).to_be_visible()
            assert all(c['support'] is None for c in read()['causes'])
            submit(82)
            assert submit(82)['duplicate']
            assert read()['totals']['contributors']==3
            assert read()['totals']['requests']==15000
            page.get_by_role('button',name='刷新统计 ↻').click()
            expect(page.get_by_text('当前支持较多：',exact=False)).to_be_visible()
            for width in (1440,894,390,320): check_width(page,width)
            page.get_by_label('选择一个解释族').select_option('input')
            expect(page.locator('.factor-grid')).to_be_visible()
            page.locator('.score-details summary').click()
            expect(page.locator('table tbody tr')).to_have_count(7)
            page.get_by_role('link',name='研究方法',exact=True).click()
            expect(page.get_by_role('heading',name='哪些数据进入研究？')).to_be_visible()
            page.get_by_role('link',name='隐私与参与',exact=True).click()
            expect(page.get_by_role('heading',name='“匿名”有明确边界')).to_be_visible()
            assert page.evaluate('document.documentElement.scrollWidth <= innerWidth + 1')
            page.screenshot(path=str(OUTPUT/'study-privacy-mobile.png'),full_page=True)
            page.goto(BASE+'/#/')
            page.set_viewport_size({'width':1440,'height':1000})
            expect(page.get_by_role('heading',name='正在进行的研究')).to_be_visible()
            page.screenshot(path=str(OUTPUT/'study-home-contributed.png'),full_page=True)
            submit(80,2,withdraw=True)
            assert read()['totals']['contributors']==2
            assert all(c['support'] is None for c in read()['causes'])
            assert not errors,errors
            assert all(url.startswith(BASE) for url in requests),requests
            assert browser.contexts[0].cookies()==[]
            (OUTPUT/'browser-results.json').write_text(json.dumps({'synthetic_only':True,'passed':True,'page_errors':errors,
                'checks':['honest empty state','insufficient-contributor gate','real signed submission','duplicate replacement','seven hypotheses','four responsive widths','score intervals','method and privacy','withdrawal','no external requests or cookies']},indent=2)+'\n')
        except Exception:
            (OUTPUT/'failure.txt').write_text(traceback.format_exc())
            page.screenshot(path=str(OUTPUT/'failure.png'),full_page=True)
            raise
        finally: browser.close()

if __name__=='__main__':
    executable=os.environ.get('STUDY_BINARY',str(ROOT/'study'))
    db=OUTPUT/'isolated-study.db'
    if db.exists(): db.unlink() # isolated test-only DB, never production path
    env={**os.environ,'STUDY_ADDR':'127.0.0.1:18088','STUDY_DB':str(db)}
    with (OUTPUT/'go-server.log').open('w') as out:
        process=subprocess.Popen([executable],cwd=ROOT,env=env,stdout=out,stderr=out)
        try:
            for _ in range(100):
                try: read();break
                except Exception: time.sleep(.1)
            else: raise RuntimeError('Test receiver failed to start')
            smoke()
        finally:
            process.terminate()
            try: process.wait(timeout=10)
            except subprocess.TimeoutExpired: process.kill();process.wait()
