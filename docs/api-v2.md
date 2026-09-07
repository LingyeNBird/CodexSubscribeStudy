# v2 原始证据 API

默认接收根地址：https://codex.nightunderfly.online/。测试只用隔离本机服务，不向生产站提交合成数据。

| 路径 | 用途 |
| --- | --- |
| GET /api/v2/protocol | 固定方法、候选与摘要依据 |
| GET /api/v2/studies | 项目列表 |
| GET /api/v2/studies/gpt6-components | 公开联合统计 |
| POST /api/v2/reports | 更新单个统计批次 |
| POST /api/v2/withdraw | 明确撤回同签名身份的全部 v1/v2 贡献 |

报告无最小请求数、周期数、区间数、来源数或局部秩要求。零请求的质量摘要和单请求贡献均可接收；格式、签名与数值完整性校验不是科研准入门槛。

## 请求与签名

JSON 报告严格包含 `protocol`、`study_id`、`method`、`method_digest`、`public_key`、`revision`，提交额外包含 `batch_id`、`summary`。撤回不带后二项。`method_digest` 是规范文件 `protocol/method-v2.json` 去掉首尾空白的 SHA-256，与 Sub2Pool `pooled_descriptor.json` 一致。方法为 `pooled-profile/raw-only-2`。

`X-Study-Signature` 为 Ed25519 标准 base64 签名。被签名字节为 `CodexSubscribeStudy/2\nPOST\n` + 路径 + `\n` + 原始 JSON 字节。重新排版必须重新签名。公钥为32字节标准base64，revision为正安全整数，batch_id为随机UUIDv4，不编码账号/时间。

`summary` 的完整允许字段：

```text
requests, gpt6_requests, other_requests, raw_usd, gpt6_raw_usd,
quota_points, intervals, groups, contrasts, gateway_only, quality,
log_evidence, gpt6_quota, information
```

三条 `log_evidence` 曲线均为1311点，分别对应漂移0/0.15/0.4；在全1倍率处以零为参考，不带先验。`gpt6_quota` 为1311点候选额度归属总量，`information` 为4×4对称半正定信息矩阵。仅有单区间片段时主证据平坦，不伪造信息。

**不存在容量估值字段或辅助证据。** `capacity_context`、`auxiliary_evidence`、PF/恒定估值以及任何额外字段都会被拒绝，而不是接受后隐藏。原始文本、Token明细、调用时间线、真实账号/参与者、IP也不在schema中。大小写别名、重复键、null、非有限数、错误维度等均拒绝。请求体上限256KiB；科学样本量无下限，资源与安全上限仍存在。

可读的 Python 签名与独立双区间计算示例见 `scripts/pooled_fixture.py`。它只产生合成测试数据，不是生产贡献者SDK或真实研究结果。

## 更新、历史与撤回

同安装使用递增 revision，同 batch_id 较新报告原子替换，重复相同报告幂等；不同批次保留。迟到的其他批次旧版本不能越过已用版本或撤回下界。身份容量及10万个批次上限用于防止资源耗尽，不自动删除历史腾空间。

v1路由和原报告保留。旧数据单独归档，不与v2重复加总；不将v1七个评分冒充完整新网格证据。旧客户端的明确签名撤回也适用于同身份的v2批次。停止科研、迁移、身份重置均不是远端撤回。原120天过期清理函数现为兼容no-op。

返回200含 `accepted`、`revision`、`duplicate`；错误400表示字段/签名无效，409表示版本冲突，413体积超限，415非JSON，429限流，503存储或容量不可用。拒绝时不回显请求正文或凭据。

公开GET不返回公钥、批次ID、单个来源曲线或时间线。应用不记录IP/访问日志，直接网络和反代仍可见出口IP；单请求贡献不保证k匿名。撤回为逻辑删除，不保证旧数据库页、备份或下载副本被物理擦除。
