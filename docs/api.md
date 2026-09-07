# 原始证据 API

默认接收根地址：https://codex.nightunderfly.online/。测试只用隔离本机服务，不向生产站提交合成数据。

| 路径 | 用途 |
| --- | --- |
| GET /api/protocol | 固定方法、候选与摘要依据 |
| GET /api/studies | 项目列表 |
| GET /api/studies/gpt6-components | 公开联合统计 |
| POST /api/reports | 更新单个统计批次 |

报告无最小请求数、周期数、区间数、来源数或局部秩要求。零请求的质量摘要和单请求贡献均可接收；格式、签名与数值完整性校验不是科研准入门槛。

## 请求与签名

JSON 报告严格包含 `protocol`、`study_id`、`method`、`method_digest`、`public_key`、`revision`、`batch_id`、`summary`。`method_digest` 是规范文件 `protocol/method.json` 去掉首尾空白的 SHA-256，与 Sub2Pool `pooled_descriptor.json` 一致。方法为 `pooled-profile/raw-only`。

`X-Study-Signature` 为 Ed25519 标准 base64 签名。被签名字节为 `CodexSubscribeStudy\nPOST\n` + 路径 + `\n` + 原始 JSON 字节。重新排版必须重新签名。公钥为32字节标准base64，revision为正安全整数，batch_id为随机UUIDv4，不编码账号或时间。

`summary` 的完整允许字段：

```text
requests, gpt6_requests, other_requests, raw_usd, gpt6_raw_usd,
quota_points, intervals, groups, contrasts, gateway_only, quality,
log_evidence, gpt6_quota, information
```

三条 `log_evidence` 曲线均为1311点，分别对应漂移0/0.15/0.4；在全1倍率处以零为参考，不带先验。`gpt6_quota` 为1311点候选额度归属总量，`information` 为4×4对称半正定信息矩阵。仅有单区间片段时主证据平坦，不伪造信息。

**不存在容量估值字段或辅助证据。** `capacity_context`、`auxiliary_evidence`、PF/恒定估值以及任何额外字段都会被拒绝，而不是接受后隐藏。原始文本、Token明细、调用时间线、真实账号/参与者、IP也不在schema中。大小写别名、重复键、null、非有限数、错误维度等均拒绝。请求体上限256KiB；科学样本量无下限，资源与安全上限仍存在。

可读的 Python 签名与独立双区间计算示例见 `scripts/pooled_fixture.py`。它只产生合成测试数据，不是生产贡献者SDK或真实研究结果。

## 更新与长期保留

同安装使用递增 revision；同 batch_id 的报告原子替换，重复相同报告幂等；不同批次长期保留。低于已接收 revision 的报告不能覆盖已接收状态。身份容量及10万个批次上限用于防止资源耗尽，不自动删除数据腾空间。

停止参与、关闭科研、身份重置和长期未更新都不会删除已提交数据。返回200含 `accepted`、`revision`、`duplicate`；错误400表示字段或签名无效，409表示revision冲突，413体积超限，415非JSON，429限流，503存储或容量不可用。拒绝时不回显请求正文或凭据。

公开GET不返回公钥、批次ID、单个来源曲线或时间线。应用不记录IP或访问日志，直接网络和反代仍可见出口IP；单请求贡献不保证k匿名。贡献会长期保留，参与者应在授权前理解这一保留规则。
