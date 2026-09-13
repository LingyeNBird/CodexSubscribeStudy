# 共研 · Codex Subscribe Study

共研是一个收集自愿贡献数据、展示研究统计结果的平台，使用 Go、Vue 3 和 SQLite 构建。

**访问网站：https://codex.nightunderfly.online/**

## 研究项目

- **GPT-6 额度异常归因**：接收 Sub2Pool 提交的统计证据，比较不同成本分项对额度消耗异常的解释。
- **ChatGPT 套餐降智封号问卷**：收集账号情况与使用经历，展示样本分布以及套餐、工具、使用方式等因素与异常状态的关联。

## 如何参与

**参与额度研究**：在 Sub2Pool 的「科研共创」设置中填写本站地址，选择研究并确认授权。自建站点请填写对应的 HTTPS 地址。

**参与问卷调查**：在网站选择问卷项目，填写并提交。无需填写问卷，也可以直接查看统计结果。

## 部署

在仓库根目录执行：

```sh
docker compose up -d --build
```

服务默认监听 `127.0.0.1:8080`。对外提供服务时，请配置 HTTPS 反向代理。

数据库保存在容器内的 `/data/study.sqlite`，通过命名卷 `study-data` 持久化。SQLite 使用 WAL，运行时还可能出现 `study.sqlite-wal`、`study.sqlite-shm`；整个 `/data` 目录必须可写。**`docker compose down -v` 会删除该数据卷。**

从旧版 bbolt 升级时，保留原数据卷并执行 `docker compose up -d --build`。新容器首次启动会只读导入 `/data/study.db`，迁移成功后才启动 HTTP 服务，原文件保留不覆盖。不要让旧容器同时写入旧数据库；Compose 替换原容器时会停止旧服务。导入后的重启不会重复导入，后续数据只写入 SQLite。

服务支持以下环境变量；使用 Docker Compose 时，在 `compose.yaml` 的 `environment` 中配置：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `STUDY_ADDR` | `:8080` | HTTP 监听地址 |
| `STUDY_DB` | `data/study.sqlite` | SQLite 数据库路径；Compose 已设置为 `/data/study.sqlite` |
| `STUDY_LEGACY_DB` | SQLite 文件同目录下的 `study.db` | 首次启动时尝试导入的旧 bbolt 文件；文件不存在则正常创建新库，显式设为空可禁用自动导入 |
| `STUDY_MAX_REPORTERS` | `10000` | GPT-6 研究贡献身份数上限 |
| `STUDY_ADMIN_CONFIG` | `config/admin.json` | 管理面板凭据文件路径；Compose 已设置为 `/config/admin.json`。文件不存在时不启动管理面板 |

## 管理面板

管理面板路径：`#/studies/chatgpt-account-survey/adminPanel`。它是独立的全屏工具，不显示站点的导航与页脚，按 `config/admin.json` 里的凭据登录；未配置凭据时 `/api/admin/*` 一律返回 404，面板不可用。

面板分成左侧常驻条件栏和右侧四个视图，条件在两个视图之间共享：

| 视图 | 用途 |
| --- | --- |
| 明细表 | 一行一份问卷，列可增删、表头可排序，点任意一行从右侧打开详情 |
| 交叉透视 | 任选两个维度（含「异常状态」「提交月份」），看组合分布，可切换份数、占行、占列 |
| 两组对比 | A 组与 B 组逐题比较占比差值，并给出 φ 系数 |
| 时间 | 按天、周、月看问卷量与各异常状态占比，点柱子把该时间段加进条件 |

条件栏支持异常状态、提交时间、任意多道题的「包含 / 排除」条件（同题内为“或”，题与题之间为“且”）、全文搜索（含补充文本与备注）和标签。完整条件会写进地址栏，可以直接收藏或分享给协作者。

列表显示填写者提交的**原始值**：账号地区按代码转成地区名，存活时长、分发人数、最高并发、IP 风险分数按填写时的数值显示。统计接口使用的分组只在需要时以「统计口径」标注，不会替代原值。

可以给任意问卷加标签和备注，也可以勾选多行批量加标签。标签只写入管理端自己的 `survey_annotations` 表，不改动问卷原文，也不会进入公开统计。

导出支持 CSV 与 JSON，范围默认是当前筛选结果（勾选行时只导出选中的问卷）。CSV 一行一份、每题一列，带 UTF-8 BOM，Excel 直接打开不乱码；JSON 保留原始值、统计口径和存储原文。

启用方式：

```sh
cp config/admin.json.sample config/admin.json
# 修改 username 与 password
```

`config/admin.json` 已加入 `.gitignore` 和 `.dockerignore`，只有示例文件会进入版本库。字段如下：

```json
{ "username": "admin", "password": "换成你自己的强密码" }
```

Docker Compose 已经把该文件以只读方式挂载进容器（`./config/admin.json:/config/admin.json:ro`）。容器以非 root 用户 10001 运行，宿主机上的配置文件需要可读，例如 `chmod 644 config/admin.json`；否则容器启动后管理面板会处于禁用状态并在日志中提示。

该页面不会出现在导航中，也不参与站点路由的其他分支。它通过三种方式避免被搜索引擎收录：`robots.txt` 的 `Disallow: /api/admin/`、管理接口的 `X-Robots-Tag: noindex, nofollow`、以及页面运行时插入的 `<meta name="robots" content="noindex,nofollow">`。**地址本身不是安全边界，凭据才是。**

修改凭据后重启服务即可生效；已登录的会话保存在内存中，重启后需要重新登录。登录窗口为 1 分钟最多 10 次尝试，会话空闲 2 小时过期。标签与备注保存在数据库里，随 `-backup` 快照一起备份。

## 本地开发

需要 Go 1.25 或更高版本、Node.js 22.12 或更高版本，以及 pnpm 12.3.4。SQLite 驱动使用纯 Go，无需 CGO、系统 SQLite 或新增数据库容器。

先安装依赖并构建前端：

```sh
cd web
corepack enable
pnpm install --frozen-lockfile
pnpm build
cd ..
```

启动后端：

```sh
go run ./cmd/study
```

访问 http://127.0.0.1:8080/。后端会嵌入构建后的前端文件。

修改前端时，可在另一个终端启动开发服务器：

```sh
cd web
pnpm dev
```

开发服务器会将 `/api` 请求转发至本地 8080 端口。

运行检查：

```sh
go test ./...
go vet ./...
cd web
pnpm check
```

## 备份

可在服务运行期间使用内置命令生成完整的 SQLite 快照，备份文件包含已提交的 WAL 数据。目标文件必须不存在，命令不会覆盖已有文件。

本地运行：

```sh
go run ./cmd/study -backup /path/to/backup.sqlite
```

如果使用自定义数据库路径，备份时也需设置相同的 `STUDY_DB`。

Docker Compose 部署：

```sh
docker compose exec study /usr/local/bin/study -backup /data/study-export.sqlite
docker compose cp study:/data/study-export.sqlite ./study-export.sqlite
```

再次备份时换一个新文件名。**不要只复制运行中的 `study.sqlite` 文件**，最新提交可能仍在 WAL 中。恢复时停止服务，将备份放到新的路径并设置 `STUDY_DB` 指向它；不要将旧库的 WAL/SHM 文件与恢复的快照混用。

## 导入旧 bbolt 数据库

除首次启动自动导入外，也可以显式指定源文件和 SQLite 目标：

```sh
STUDY_DB=data/imported.sqlite go run ./cmd/study -import-bbolt /path/to/study.db
```

此命令完成导入后退出。源 bbolt 文件只读打开，源库的写入服务须先停止；SQLite 目标必须没有业务数据。问卷、记录编号、序号、提交时间、证据批次、贡献身份和最高修订号均保留，历史字段格式在导入时转换，统计缓存从问卷重新构建。整个导入在一个 SQLite 事务中提交，失败会回滚；同一来源重复导入不会重复计数。不能通过重新导入旧库来合并 SQLite 启用后的新数据。

回滚程序时，保留 SQLite 数据库，另行使用仍保留的旧 bbolt 文件；旧文件不包含切换到 SQLite 后的新增数据，不能把它视为最新备份。

## 数据库结构与迁移

- `internal/database`：连接池、WAL、事务、一致性备份和完整性检查。
- `internal/database/migrations/*.sql`：按编号执行的 SQL 迁移。`schema_migrations` 记录版本、文件名、校验值和执行时间；修改已执行的迁移或使用旧程序打开更新的数据库会报错。后续结构变化新增递增编号文件，不修改历史文件。
- `internal/study/import_bbolt.go`、`survey_legacy.go`：仅用于旧 bbolt 数据导入，正常服务读写不再使用 bbolt。
- `survey_submissions`：问卷原文 JSON，以及可索引的提交时间、异常状态。时间索引统一使用 UTC、固定九位小数格式；未知时间保留 NULL。
- `survey_answers`、`survey_details`：展开 JSON 的只读查询视图，可用于按题目、选项和补充信息查询；不是重复保存的数据。
- `evidence_identities`、`evidence_batches`：贡献身份和证据批次。修订号以十进制文本保存，完整保留协议的 uint64 范围。
- `survey_statistics`：带统计截止位置的增量缓存；`legacy_imports`：旧库导入记录。
- `survey_annotations`：管理面板的标签与备注，按问卷编号关联，可为空；与问卷原文、公开统计完全分离。

## 文档

- [研究方法](docs/method.md)
- [API 与提交协议](docs/api.md)
- [研究记录](research/README.md)
- [固定方法规范](protocol/method.json)

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
