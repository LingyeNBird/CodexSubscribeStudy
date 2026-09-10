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

- `POST /api/survey/submissions`：JSON，请求上限 32 KiB；成功返回 HTTP 201 `{ "accepted": true }`。每次成功提交追加一份问卷，并记录服务器 UTC 提交时间，不接受客户端指定提交时间。
- `GET /api/survey/statistics`：公开汇总，不返回逐份问卷或补充文本。请求时将新增问卷纳入统计，并返回对应的统计范围；HTTP 响应不缓存。

提交结构：

```json
{
  "status": ["正常"],
  "answers": { "plans": ["Plus"], "usage": ["直登"] },
  "details": { "country": "JP", "duration": "24", "durationUnit": "小时" }
}
```

`status` 为非空字符串数组：正常必须单独选择；降智、封号、风控（限流）可以任意组合，不得重复。`answers.plans` 必填。其他问题允许未作答，未作答不进入对应问题分母。所有提交值经后端白名单及条件关系验证。

`answers` 接受目录 `internal/study/survey_catalog.json` 中的 models、plans、tools、activation、usage、proxy、network、ipStability、official、desktopMode、cliMode、shared、warning、truncated、discovery、limitedDiscovery；值为选项字符串数组，单选必须仅一项，多选不得重复。network 选项为家宽、机房、机场；ipStability 选项为固定 IP、IP 乱飞。网络类型、IP 稳定性和 IP 地区适用于直登或反代，反代工具仅适用于反代。limitedDiscovery 仅在选择风控（限流）时接受，选项为容量达到上限、服务不可用、周限额度明显骤降、其他。

`details` 仅接受：

- country、exitCountry：ISO alpha-2 地区码；出口可为 `unknown`。
- duration、durationUnit：账号存活时长，非负有限数值字符串和天/小时/星期/月/年，换算约定小时÷24、星期×7、月×30、年×365。
- people：整数且至少2，只在分发为“是”时有效；concurrency：非负整数或 `unknown`。数值上限为1000000。
- degradationTime、banTime：对应异常发生时的 `YYYY-MM-DD`、`YYYY-MM-DDTHH:mm` 或 `unknown`。不附加时区推断。
- discoveryOther、limitedDiscoveryOther、proxyOther、thirdPartyOther：仅对应“其他”选项有效的补充文本。
- `toolMode:<工具名>`：已选第三方工具的直登（OAuth）或反代；Claude Code 不接受此项。

补充值每项不超过2000字节。账号时长、人数、并发、地区、连接模式的统计分组由服务端计算，不接受客户端直接上传分组结论。

可选 `usagePattern` 为24个整数，依次对应当地时间0点至23点，每项取值0至24，表示该小时选中的格数（24对应100%，12表示通常约有30分钟在使用）。未填写时省略，不计入使用规律分母。

可选 `ipRisk` 为0至100的整数，在使用方式包含直登或反代时接受。0为最低风险，100为最高风险；未填写时省略。统计按0–14、15–24、25–39、40–49、50–69、70–100分组。

统计接口返回 `version: 2`、`catalogDigest`、`statuses`、`factors`、`usagePattern` 和 `range`，只包含汇总计数及题目口径，不返回派生比例或相关系数。

- `statuses`：长度为8的互斥状态计数数组。数组下标为状态掩码：降智=1、封号=2、风控=4；0为正常，3为降智且封号，5为降智且风控，6为封号且风控，7为三者都有。一份问卷只进入一个状态。
- `factors`：按题目目录排序的数组，每项包含 `id`、`title`、`description`、`multiple`、`distributionScope`、`associationScope`、`comparable`、`applicable` 和 `options`。`applicable` 是该题有效回答的8格状态计数；`options` 每项包含 `label` 和同样的8格 `counts`。多选题的选项计数不能相加充当有效回答总数。
- `comparable`：该题是否允许关联比较。模型、降智发现方式、风控识别方式为 false；前端只展示它们的样本分布。
- `usagePattern`：包含有效回答 `total` 及24个小时的格数总和 `sums`，前端按 `sums[hour] / total` 计算平均值；无回答时不作除法。

`range` 包含 firstSubmissionId、lastSubmissionId（已纳入统计的提交编号范围）、firstSubmittedAt、lastSubmittedAt（已知提交时间的最小值和最大值）、unknownTimeCount（提交时间未知的历史问卷数）、computedAt（本次结果计算时间）。无问卷时编号为0、提交时间为null；时间未知不等于问卷未被统计。

前端统一计算分布、比例和 φ。异常组合采用并集：状态下标与所选异常掩码按位与不为0时，累加该格一次。未选择某个因素选项的人群计数等于该题 `applicable` 减去该选项的 `counts`。分布分母只含满足异常组合且回答该题的问卷；关联对照在该题全部有效回答中计算，包括正常样本。φ 分母为0时前端生成 null；未进行显著性检验、多重比较校正或混杂调整。单因素汇总不支持任意跨因素联合筛选。

`POST /api/survey/statistics/query` 按前提条件重新计算相同格式的统计汇总。请求体为：

```json
{
  "prerequisites": [
    { "factor": "country", "options": ["菲律宾"] },
    { "factor": "plans", "options": ["Plus", "Pro 5X"] }
  ]
}
```

`factor` 和 `options` 必须来自当前问卷目录。相同因素内的选项采用“或”，不同因素之间采用“且”；未回答前提因素的问卷不匹配。接口最多接受12个因素、合计32个选项，从 SQLite 原始问卷计算条件样本，不返回单份问卷。至少需要一个前提，未知、重复或空选项返回400。

问卷最多保存100000份。备份及数据库导入方式见 [部署说明](../README.md)。
