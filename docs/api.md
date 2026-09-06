# 公共协议、数据最小化和部署隐私

协议 `codex-cost-study/1`，研究 `gpt6-components`，方法 `log-capacity-loco/1`。

## 接口

- `GET /healthz`：服务健康。
- `GET /api/v1/protocol`：机器可读固定方法，去除末尾换行后 SHA-256 为 `method_digest`。
- `GET /api/v1/studies`：`{"studies":[研究总览]}`。
- `GET /api/v1/studies/gpt6-components`：总量、质量状态、七类原因的支持度、评分与90%重抽样区间、代表倍率、方法摘要。有效安装不足3个时支持度/评分字段为null，**不是0%**。
- `POST /api/v1/reports`：签名的滚动摘要，按匿名随机公钥更新，不是累加每次上传。
- `POST /api/v1/withdraw`：签名撤回，版本须更新，不能附带summary。

没有逐安装、公钥列表、原始记录或个人数据的公开查询端点。SPA与API同源，不开放跨域的浏览器提交；服务端客户端不受浏览器CORS限制。

## 签名与幂等

JSON正文UTF-8，Content-Type `application/json`，最大32768字节。`X-Study-Signature` 是64字节 Ed25519签名的标准Base64编码。签名消息的精确字节：

```
CodexSubscribeStudy/1\nPOST\n/api/v1/reports\n<原始JSON正文>
```

撤回时把路径改成 `/api/v1/withdraw`。服务器对原始收到字节验签，不自己重排JSON；拒绝重复键、不认识的字段、非有限值、错误维数、负数、越界和非半正定协方差。参考实现 `internal/study/protocol.go`；跨语言样例 `protocol/testdata/synthetic-report.json` **完全是合成数据**，不应用它向生产贡献。

正文仅有：`protocol,study_id,method,method_digest,public_key,revision,summary`。public_key为随机Ed25519公钥（32字节Base64），不是账号、机器ID或IP。revision为1..2^53-1递增整数。客户端随机种子本地加密保存，按网站和研究隔离公钥，不上传私钥。

同一键同一版本同一正文重试返回duplicate=true。相同版本不同正文、旧版本、已撤回版本返回409；更高版本替换完整摘要。撤回保留键哈希与版本墓碑，旧的已签名报告不能恢复已撤数据。重新明确授权后可发送更高版本。不是保证不同安装来自不同真人。

成功：`{"accepted":true,"revision":3,"duplicate":false}`。
错误：固定代码JSON，400无效、405方法、409版本冲突、413体积、415类型、429全局限流、503容量/忙碌。客户端对错误指数退避；不要无限高速重试。

## summary固定字段

规模：`window_days=90,requests,baseline_requests,gpt6_requests,raw_usd,gpt6_raw_usd,quota_points,cycles,blocks`。成本取整到美元；额度以百分点表示，不同账号的百分点不能解释为一个池子。

质量：`eligible,gateway_only,status,identifiable[4],design_rank,exclusions{固定键计数}`。status只能是 insufficient_data/unidentifiable/model_mismatch/drift_sensitive/exploratory/external_usage_uncontrolled。合格原因排名必须exploratory，达到客户端相同门槛。200次请求以下不接收。

推断：`score_mean[7],score_cov[7][7],support[7],factor_estimates[7][4]`。顺序为 unchanged/global/cache_read/cache_creation/output/input/mixed；分项顺序 input/cache_creation/cache_read/output。评分相对unchanged，故首项与对应协方差为0；支持度eligible时和为1，否则全0。方法摘要不符拒绝写入，避免版本混池。

这些统计是在去标识化最小化策略下选择的摘要，**不是差分隐私**。不发送逐请求、逐区间数据、时间序列、模型别名、IP、姓名、API Key、提示词或回答。发送摘要需用户明确同意目的地和范围。

## 存储、过期与删除

使用bbolt单文件事务数据库，单个安装只保留最新摘要与粗粒度收到小时。正文未存储；公钥包含在私有报告内，用哈希作为键。公开只汇总不同安装。120天不更新则公共读忽略，每日清理逻辑删除统计，保留反重放墓碑。不同方法摘要不进入当前聚合。

撤回从当前统计移除，公共响应缓存最多30秒；备份或外部截图无法立即撤回。bbolt是页式存储，逻辑删除不保证底层磁盘页已擦除；物理清除需运营者安排安全压缩/备份轮换并遵守实际部署的数据保留政策。不要把“撤回”表述为第三方缓存、备份或旧磁盘扇区立即全部消失。

默认最多10000个匿名键（含墓碑），可通过STUDY_MAX_REPORTERS调整。全球写入桶60突发、每2秒补1，最多16并发，不按IP建表。未知键撤回同样受容量限制。匿名公开接口仍有拒绝服务和Sybil风险；需要运营层监测资源，不能声称靠签名杜绝伪造。

## 部署隐私清单

应用没有访问日志、不读取 RemoteAddr/X-Forwarded-For，不保存IP字段，没有分析脚本、远程字体、Cookie或用户登录。HTTP/TLS握手和反向代理天然能看到出口IP。运营者须明确告知并处理代理/CDN/托管平台层的日志；只关闭应用日志不足以保证网络匿名。

推荐反向代理关闭本服务access_log，不把请求体或签名headers写入调试日志。TLS终止在可信入口；生产Sub2Pool仅允许HTTPS公网根地址，禁止重定向并校验/固定DNS目标以防向内网发送统计。客户端既不使用环境代理也不带Sub2API凭据。

禁止把测试数据、密钥、数据库或完整POST日志作为公开CI产物。仓库测试使用合成账号及仅绑定本机的模拟服务。
