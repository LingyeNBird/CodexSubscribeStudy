"""Small, auditable Python fixtures for the signed raw-only wire contract.

Synthetic costs only. The two-row profile is independently evaluated using a
2x2 inverse, rather than copying the client's vectorized NumPy implementation.
No network access and no capacity estimates. Used by browser/integration tests.
"""
import base64
import hashlib
import itertools
import json
import math
from pathlib import Path
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

ROOT = Path(__file__).resolve().parents[1]
DESCRIPTOR_BYTES = (ROOT / "protocol/method-v2.json").read_bytes().strip()
DESCRIPTOR = json.loads(DESCRIPTOR_BYTES)


def grid():
    points = set(itertools.product(DESCRIPTOR["grid"], repeat=4))
    for v in DESCRIPTOR["extra_single_and_global"]:
        points.add((v,) * 4)
        for i in range(4):
            p = [1.] * 4
            p[i] = v
            points.add(tuple(p))
    return sorted(points)


def summary(informative=False):
    points = grid()
    n = len(points)
    result = {"requests": 1, "gpt6_requests": 1, "other_requests": 0,
              "raw_usd": 1., "gpt6_raw_usd": 1., "quota_points": 1.,
              "intervals": 1, "groups": 1, "contrasts": 0, "gateway_only": True,
              "quality": {key: 0 for key in DESCRIPTOR["quality_keys"]},
              "log_evidence": [[0.] * n for _ in range(3)],
              "gpt6_quota": [1.] * n, "information": [[0.] * 4 for _ in range(4)]}
    if not informative:
        return result
    # Two raw intervals; their unknown common scale is profiled separately for
    # every candidate. The true cache-read factor in this SYNTHETIC case is two.
    z = [[80., 5., 4., 2., 4.], [80., 5., 4., 160., 4.]]
    y = [97 / 30, 413 / 30]
    result.update(requests=4, gpt6_requests=2, other_requests=2, raw_usd=348., gpt6_raw_usd=188.,
                  quota_points=sum(y), intervals=2, groups=1, contrasts=1)
    null = points.index((1, 1, 1, 1))
    for d, drift in enumerate(DESCRIPTOR["drifts"]):
        aa, ab = 1 / 6 + .01, -1 / 12
        bb = aa + drift * drift * max(y[1], .5)**2 * .1
        determinant = aa * bb - ab * ab
        w = [[bb / determinant, -ab / determinant], [-ab / determinant, aa / determinant]]

        def inner(a, b):
            return sum(a[i] * w[i][j] * b[j] for i in range(2) for j in range(2))

        curve = []
        for p in points:
            cost = [row[0] + sum(row[j+1] * p[j] for j in range(4)) for row in z]
            residual = max(0., inner(y, y) - max(inner(cost, y), 0.)**2 / inner(cost, cost))
            curve.append(-.5 * (4 + 2 - 1) / 2 * math.log1p(residual / 4))
        result["log_evidence"][d] = [round(v - curve[null], 10) for v in curve]
        if d == 1:
            cost = [sum(row) for row in z]
            denom = inner(cost, cost)
            scale = max(inner(cost, y), 0.) / denom
            cols = [[scale * row[j+1] for row in z] for j in range(4)]
            result["information"] = [[round(inner(a, b) - inner(a, cost) * inner(cost, b) / denom, 8)
                                      for b in cols] for a in cols]
    result["gpt6_quota"] = [round(sum(y[i] * sum(z[i][j+1] * p[j] for j in range(4)) /
                                     (z[i][0] + sum(z[i][j+1] * p[j] for j in range(4))) for i in range(2)), 8)
                           for p in points]
    return result


def packet(seed=80, revision=1, batch="11111111-1111-4111-8111-111111111111", *, informative=False, extra=None):
    key = Ed25519PrivateKey.from_private_bytes(bytes([seed]) * 32)
    report = {"protocol": DESCRIPTOR["protocol"], "study_id": DESCRIPTOR["study_id"],
              "method": DESCRIPTOR["method"], "method_digest": hashlib.sha256(DESCRIPTOR_BYTES).hexdigest(),
              "public_key": base64.b64encode(key.public_key().public_bytes(serialization.Encoding.Raw, serialization.PublicFormat.Raw)).decode(),
              "revision": revision}
    report.update(batch_id=batch, summary=summary(informative))
    if extra:
        report["summary"].update(extra)
    body = json.dumps(report, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()
    path = "/api/v2/reports"
    signed = b"CodexSubscribeStudy/2\nPOST\n" + path.encode() + b"\n" + body
    return path, body, base64.b64encode(key.sign(signed)).decode()
