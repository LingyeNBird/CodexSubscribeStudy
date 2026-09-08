# 共研 · Codex Subscribe Study

共研是一个收集自愿贡献数据、展示研究统计结果的平台，使用 Go、Vue 3 和 bbolt 构建。

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

数据库保存在容器内的 `/data/study.db`，通过命名卷 `study-data` 持久化。**`docker compose down -v` 会删除该数据卷。**

服务支持以下环境变量；使用 Docker Compose 时，在 `compose.yaml` 的 `environment` 中配置：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `STUDY_ADDR` | `:8080` | HTTP 监听地址 |
| `STUDY_DB` | `data/study.db` | 数据库路径；Compose 已设置为 `/data/study.db` |
| `STUDY_MAX_REPORTERS` | `10000` | GPT-6 研究贡献身份数上限 |

## 本地开发

需要 Go 1.23 或更高版本、Node.js 22.12 或更高版本，以及 pnpm 12.3.4。

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

先停止服务，再使用相同的数据库路径执行备份。以下命令适用于本地运行：

```sh
go run ./cmd/study -backup /path/to/backup.db
```

如果使用自定义数据库路径，备份时也需设置相同的 `STUDY_DB`。Docker 部署请先停止容器，再备份 `study-data` 数据卷。

## 文档

- [研究方法](docs/method.md)
- [API 与提交协议](docs/api.md)
- [研究记录](research/README.md)
- [固定方法规范](protocol/method.json)

## 许可证

本项目采用 [MIT 许可证](LICENSE)。
