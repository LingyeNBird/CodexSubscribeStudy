# 开发与持续集成

在 web 中运行 `pnpm install --frozen-lockfile`、`pnpm check`、`pnpm build`；仓库根目录运行 `go vet ./...`、`go test -race ./...`。浏览器回归入口为 `scripts/browser_smoke.py`，只使用隔离数据库与本机合成接收服务，不在生产环境运行。

所有 PR 共用 `.github/workflows/ci.yml`：安装锁定依赖、Go 测试与静态检查、Vue 构建、真实 HTTP/浏览器回归和镜像构建。main 验证通过后仍自动发布 GHCR；PR 不发布。验证 job 只读仓库，发布 job 才有 packages:write。

新增功能应增加测试用例，必要时扩展现有 job/step 或矩阵；不按 PR 编号或一次性任务生成 review 工作流。验证流程只检出、测试并报告，不负责恢复/解包/提交业务源码。只有独立且长期的自动化职责才考虑新工作流，不以 PR 数量决定文件数量。

一个 YAML 可以产生任意多次运行，Actions 中的历史运行不是新工作流文件。已删除的临时恢复工作流不再属于当前源码树；保留历史提交不等于继续使用它们。大型研究实验单独复现，常规 CI 运行其轻量数值回归，不每次重复论文级全套模拟。

参考：[GitHub 持续集成](https://docs.github.com/en/actions/get-started/continuous-integration)、[复用工作流配置](https://docs.github.com/en/actions/concepts/workflows-and-actions/reusing-workflow-configurations)。
