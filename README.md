# 共研 · Codex Subscribe Study

面向 Sub2Pool 的自愿、去标识化科研统计平台。Go 后端 + Vue 3 SPA + bbolt 单文件数据库；一个进程、一个数据卷，无需 MySQL/Redis。

首个研究项目是 **GPT-6 额度异常归因**：比较无需额外修正、整体倍率、缓存读、缓存创建、输出、输入和混合七类解释。支持度表示候选内的预测胜率，不是平台真实计费机制的概率。无数据时展示真实空状态，不内置虚构贡献或排行。

## 部署

```sh
docker compose up -d --build
```

默认监听宿主机 `127.0.0.1:8080`，在前面配置你自己的HTTPS反向代理。不要公开无TLS的提交地址。持久化数据位于容器 `/data/study.db`，命名卷 `study-data`。备份、恢复与代理日志配置由部署者管理；`docker compose down -v` 会删除数据，请勿误用。

随后在 Sub2Pool「科研共创」设置填写本服务的 **HTTPS根地址**，选择研究并阅读/确认授权。Sub2Pool默认关闭，placeholder不发网络请求。平台不需要配置Sub2API地址或管理员Token，也不主动访问参与者服务器。

环境变量：

| 变量 | 默认 | 用途 |
|---|---|---|
| `STUDY_ADDR` | `:8080` | HTTP监听（生产放在HTTPS代理后） |
| `STUDY_DB` | `data/study.db` | bbolt文件路径 |
| `STUDY_MAX_REPORTERS` | `10000` | 匿名键总上限，包括撤回墓碑 |

应用不记录客户端IP。**代理、CDN和网络层仍可看到IP**，请关闭或限制相应访问日志、禁止POST正文日志并向参与者准确说明。贡献使用可撤回随机公钥，而不是绝对不可关联匿名。完整隐私说明见 [API与隐私文档](docs/api.md)。

## 本地开发

Go >=1.23，Node.js22，pnpm12.3.4。

```sh
cd web
corepack enable
pnpm install --frozen-lockfile
pnpm check
pnpm build
cd ..
go test -race ./...
go vet ./...
go run ./cmd/study
```

SPA构建后通过 `go:embed` 编入二进制，开发 `web/pnpm dev` 的API请求代理至8080。源码中的 `web/dist/.keep` 仅使纯Go测试无需预构建，运行完整网页前必须构建SPA。

备份（先停止占用数据库的服务进程，再执行）：

```sh
STUDY_DB=data/study.db ./study -backup /safe/path/study-backup.db
```

CLI使用一致性事务副本，模式0600；不提供公开数据库下载接口。撤回是逻辑删除，页式数据库/备份的物理数据保留需要单独规划。

## 研究与协议

- [方法、公式、先验与可识别性](docs/method.md)
- [提交API、签名、字段、去重和撤回](docs/api.md)
- [预声明的方法描述](protocol/method.json)

公开GET：`/api/v1/studies` 和 `/api/v1/studies/gpt6-components`。提交/撤回为Ed25519签名POST；收到最新统计覆盖同安装历史，不累加轮询次数。至少3个合格安装才显示公共支持度。120天未更新的贡献过期；不提供单个贡献者的公开浏览。

**研究范围有意保守**：v1仅纳入普通档位、非长上下文、原始请求连续覆盖并可与真实额度时间对齐的 GPT-5.6/GPT-6 区间。原始成本分项只在 Sub2Pool 用户同意后随现有采集保存，不额外调用模型。原因置信受混杂、网关价卡差异和匿名提交真实性限制；详见方法。它不是付费额度预测保证，更不是自动计费调整工具。

## 视觉与测试

圆角、柔和彩色色块、清晰边框和轻微偏移阴影；无点阵、追踪脚本或第三方字体。提供项目总览、原因排序、规模汇总、可识别性、代表参数、评分区间、方法与隐私页面，支持移动端和减少动态效果偏好。

CI执行Go竞态测试、静态检查、Vue类型检查与构建，以及真实HTTP/Chromium提交、幂等、撤回和移动端测试。Python生成的合成签名fixture用于验证与Sub2Pool跨语言协议一致。合成实验不会写入生产展示。

本项目代码以 MIT 许可证发布；Sub2Pool 集成代码继续遵守其原有 AGPL-3.0-only 许可证。bbolt和前端依赖按各自许可证使用。
