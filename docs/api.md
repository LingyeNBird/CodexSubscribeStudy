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

公开GET不返回公钥、批次ID、单个来源曲线或时间线。应用不保存原始IP或访问日志，直接网络和反代仍可见出口IP；单请求贡献不保证k匿名。贡献会长期保留，参与者应在授权前理解这一保留规则。

## 账号情况问卷

- `POST /api/survey/submissions`：JSON，请求上限 32 KiB；成功返回 HTTP 201 `{ "accepted": true }`。每次成功提交追加一份问卷，不去重或覆盖。
- `GET /api/survey/statistics`：公开汇总，不返回逐份问卷或补充文本。响应禁用缓存，提交后的新请求立即可见。

提交结构：

```json
{
  "status": ["正常"],
  "answers": { "plans": ["Plus"], "usage": ["直登"] },
  "details": { "country": "JP", "duration": "24", "durationUnit": "小时" }
}
```

`status` 为非空字符串数组：正常必须单独选择；降智、封号、风控（限流）可以任意组合，不得重复。`answers.plans` 必填。其他问题允许未作答，未作答不进入对应问题分母。所有提交值经后端白名单及条件关系验证。

`answers` 接受目录 `internal/study/survey_catalog.json` 中的 models、plans、tools、activation、usage、proxy、network、quality、official、desktopMode、ciMode、shared、warning、truncated、discovery；值为选项字符串数组，单选必须仅一项，多选不得重复。

`details` 仅接受：

- country、exitCountry：ISO alpha-2 地区码；出口可为 `unknown`。
- duration、durationUnit：非负有限数值字符串和天/小时/星期/月/年，换算约定小时÷24、星期×7、月×30、年×365。
- people：整数且至少2，只在分发为“是”时有效；concurrency：非负整数或 `unknown`。数值上限为1000000。
- degradationTime、banTime：对应异常发生时的 `YYYY-MM-DD`、`YYYY-MM-DDTHH:mm` 或 `unknown`。不附加时区推断。
- discoveryOther、proxyOther、thirdPartyOther：仅对应“其他”选项有效的补充文本。
- `toolMode:<工具名>`：已选第三方工具的直登（OAuth）或反代；Claude Code 不接受此项。

补充值每项不超过2000字节。持续时间、人数、并发、地区、连接模式、事件记录精度的统计分组由服务端计算，不接受客户端直接上传分组结论。降智与封号时间分别进入各自分布。

统计返回 total、degraded、banned、limited、normal、both、statuses、22项 factors、19项 associations、outcomeAssociation。各异常计数包含重叠问卷；both 为同时报告降智与封号的份数；statuses 按八种互斥状态组合返回 label、count、tone。factors 和 associations 分别提供 degraded、banned、limited 三类异常的分布和关联。事件时间仅统计问卷实际询问的降智与封号时间。

分布分母只含该异常状态且回答该题的问卷；关联对照在该题全部有效回答中计算，包括正常样本。仅异常状态展开的模型、发现方式、事件时间不参与因素相关性。φ 分母为0时返回 null；未进行显著性检验、多重比较校正或混杂调整。

问卷使用同一 bbolt 数据库及备份流程，最多保存100000份。
