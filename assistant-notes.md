# 助手工作笔记

- 2026-09-12：用户要求在 CodexSubscribeStudy 项目中新增管理员面板，路径为 `问卷调查路径\adminPanel`，需登录、查看全部问卷细节、按问题筛选答案，并禁止搜索引擎收录。用户要求一步一步做、思考不要过长、每思考一会就记笔记，做完在本地启动测试。当前工作目录 `D:/projects/ai/CodexSubscribeStudy`。

- 读取结果：仓库当前 HEAD 为 `aea9c4d feat: add conditional survey analysis`，工作区干净。存储已迁移到 SQLite。后端 `internal/study/server.go` 的 `ServeHTTP` 用 `switch r.URL.Path` 分派 API，静态资源从 `assets fs.FS` 读取，未知路径回退 `index.html`（SPA 单页）。

- 读取结果：前端无 vue-router，`web/src/App.vue` 用 `location.hash` 手动路由，未知路由显示“这个页面还不存在”。`web/vite.config.ts` 把 `/api` 代理到 `127.0.0.1:8080`，因此本地可以同时跑 Go 后端与 Vite。

- 读取结果：数据模型 `SurveySubmission{status[], answers{key:[]string}, details{key:string}, usagePattern[]*int, ipRisk*int}`；表 `survey_submissions(id, submitted_at, status_mask, payload)`。`SurveySubmission.normalized()` 把 details 里的 country/exitCountry/duration/people/concurrency/toolMode:* 与 ipRisk 折算成标准选项标签，是统计与筛选的统一口径，管理端应直接复用。

- 读取结果：`survey_catalog.json` 有 23 道题（key/title/description/multiple/eligibility）与全部 choices，可作为管理端“问题下拉框 + 选项分布”的权威来源，后端直接原样返回即可。

- 读取结果：`submitted_at` 只由 `insertSurveyRecord` 以固定布局 `2006-01-02T15:04:05.000000000Z` 写入（UTC）；旧 bbolt 记录可能为空。读取时用布局列表解析，失败则回退到 payload 内的 `submittedAt`。

- 读取结果：`NewServer(store, assets)` 在约 15 处测试中被调用，因此改用可变参数 Option 模式（`NewServer(store, assets, ...Option)`），避免改动既有调用点。

- 读取结果：`compose.yaml` 的服务是 `read_only: true` + 仅 `study-data:/data` 卷；`Dockerfile` 最终阶段没有 `/config` 目录。要挂载配置文件，必须在镜像里先建 `/config`，否则只读根文件系统下挂载会失败。

- 用户决定：管理员凭据放在仓库内的配置文件（提交 `config/admin.json.sample` 示例，真正的 `config/admin.json` 加入 git 忽略），并在 Compose 里把该文件只读挂载进容器。管理面板路径为 `#/studies/chatgpt-account-survey/adminPanel`。

- 决策：安全默认值是“未配置凭据 → `/api/admin/*` 一律 404”，不提供默认密码。会话用服务端随机 token + HttpOnly/SameSite=Strict Cookie（Path=/api/admin），2 小时滑动过期；登录窗口 1 分钟最多 10 次尝试。禁止收录用三重手段：`robots.txt` 的 `Disallow: /api/admin/`、`/api/admin/*` 的 `X-Robots-Tag: noindex, nofollow`、以及管理页动态插入 `<meta name="robots" content="noindex,nofollow">`。

- 下一步：写 `internal/study/admin.go`（配置加载、会话、三个只读接口），再改 `server.go` 路由与 `cmd/study/main.go`，然后补前端页面。

- 实现结果：新增 `internal/study/admin.go`（配置加载、会话、`/api/admin/session|catalog|submissions`）、`internal/study/admin_test.go`；`server.go` 增加 `Option`/`WithAdmin` 可变参数与 `/api/admin/*` 分派；`cmd/study/main.go` 读取 `STUDY_ADMIN_CONFIG`（默认 `config/admin.json`）并在日志里说明启用状态。

- 实现结果：前端新增 `web/src/data/adminApi.ts`、`web/src/pages/AdminPanelPage.vue` 与 `.css`，`App.vue` 增加 `adminRoute` 分支；`web/public/robots.txt` 产出到 `dist`；`Dockerfile` 建 `/config` 并加 `STUDY_ADMIN_CONFIG`；`compose.yaml` 只读挂载 `./config/admin.json`；`.gitignore` 与 `.dockerignore` 忽略真实配置文件。

- 测试结果：`TestLoadAdminConfig` 一开始把尾部内容校验写反（`Decode` 返回 nil 才表示有剩余内容，正确判据是 `!errors.Is(err, io.EOF)`），已修正；另一处误把 `details.concurrency = "1"` 的期望写成 `"0"`，实际分桶就是选项 `1`，已按实现口径修正断言。`go test ./...` 全绿，`go vet` 通过，CI 同名 gofmt 门禁通过。

- 验证结果：本地用 `data/survey-preview-20260908.sqlite`（45 份问卷）启动 `go run ./cmd/study`（日志 `administrator panel enabled`）与 `pnpm dev`。浏览器实测：未登录显示登录表单；登录后显示 45 份问卷，状态计数 降智17/封号2/风控11/正常23；选“账号地区”得到选项分布 菲律宾25/日本4/美国12/玻利维亚0/其他4（合计45），点选“菲律宾”后列表收敛为 25 份且摘要全部为“菲律宾”；展开单份问卷能看到按目录排序的答案与原始字段（含 `proxyOther` 网址）；“退出登录”回到登录表单。

- 验证结果：`curl` 复核未带 Cookie 的 `/api/admin/submissions` 返回 401，错误密码 401、正确密码 200 并下发 `HttpOnly; SameSite=Strict; Path=/api/admin` Cookie，`DELETE` 后同一 Cookie 再访问返回 401；`/api/admin/session` 响应带 `X-Robots-Tag: noindex, nofollow`；`/robots.txt` 返回 200 `text/plain` 且含 `Disallow: /api/admin/`。

- 风险与说明：管理面板展示的是已去标识化的问卷原文（含用户手填的“其他”补充文本），属于用户明确要求的“查看所有问卷细节”，未做二次脱敏。配置文件里的密码是明文，因此依赖 `.gitignore` + 只读挂载 + 宿主机文件权限；已把“宿主机文件需可读（如 `chmod 644`）”写进 README。管理端仍会在页面顶部渲染站点导航。

- 下一步：等待用户查看本地效果；未提交、未推送。

- 用户反馈：面板把「账号地区」显示成“其他国家或地区”，看不到原始数据。查证 SQLite 后确认：`details.country` 里实际有 TR/SG/KR/CN/TW 各 1 份，被 `normalized()` 合并成“其他国家或地区”；`concurrency` 是 3/5/15/20 等精确数字，`duration` 是 34 天、3.1 天、2 年等精确值，也都被分桶。原实现只在折叠的“原始填写字段”里露出代码，主视图确实丢失了原始信息。

- 决策：新增 `SurveySubmission.rawAnswers()`，返回填写者提交的原值（地区码、`时长+单位`、人数、并发、数值 `ipRisk`、`toolMode:*`），随 `/api/admin/submissions` 的 `raw` 字段返回；前端改为按 `raw` 展示与筛选，`normalized` 只在两者不同的时候作为“统计口径”标注。选项芯片规则：若该题所有原始值都命中目录选项（如套餐），保留完整选项列表（含 0 值）；否则只列出实际提交过的原始值，按出现次数降序、再按数值/字典序排列，避免出现一屏 0 值的目录项。

- 测试结果：`TestAdminSubmissionsExposeFullNormalizedRecords` 增加原始值断言（`raw.country = "SG"` 同时 `normalized.country = "其他国家或地区"`、`raw.concurrency = "15"` → `11 及以上`、`raw.duration = "45 天"` → `30–89 天`、`raw.ipRisk = "20"` → `15–24`）。`go test ./...`、`go vet` 全绿。

- 验证结果：重启后端后 `curl` 复核接口：`raw.country` 分布为 PH25/US12/JP4/TR1/KR1/SG1/CN1，而 `normalized.country` 仍是 菲律宾25/美国12/日本4/其他国家或地区4。浏览器实测：账号地区芯片为 `PH 菲律宾 25`、`US 美国 12`、`JP 日本 4`、`CN/KR/SG/TR 其他国家或地区 1`；最高并发显示 `3`、`5`、`15 11 及以上`、`unknown 我不知道 11` 等原值；存活时长显示 `1 年`、`3.1 天`、`34 天` 等原值；套餐这类纯选项题仍保留 `Free 免费 0`、`Go 0` 等完整列表。明细里“账号地区 US”下方标注“统计口径：美国”。

- 工具结果：浏览器 relay 标签页后来被切到后台，`page.evaluate`/`screenshot` 会超时或报“tab is not visible”，这属于 relay 限制，不是页面问题；DOM 文本证据已足够。

- 下一步：等用户确认效果；改动仍未提交、未推送。

- 用户反馈：ISO 地区代码看不懂，而且按钮上重复显示“其他国家或地区”很莫名其妙。这是对的：区域合并只是统计页为了减少长尾做的处理，放在原始数据视图里既看不懂也没意义。

- 决策：前端用 `Intl.DisplayNames(["zh-CN"], {type:"region"})` 把地区代码转成中文地区名（CN→中国、KR→韩国、SG→新加坡、TR→土耳其），`unknown` 显示为“我不知道”；不需要在代码里维护国家表。地区类问题（账号地区、出口地区）完全不再显示“统计口径”，因为合并规则对原始数据没有解释价值；存活时长、分发人数、最高并发、IP 风险这些按数值分组的仍然显示分组区间，因为那是有意义的数据区间。筛选仍然按原始值进行，只是显示成名字。

- 验证结果：浏览器实测“账号地区”按钮为 菲律宾25 / 美国12 / 日本4 / 中国1 / 韩国1 / 新加坡1 / 土耳其1；“出口地区”为 美国27 / 日本12 / 新加坡2 / 韩国1 / 台湾1 / 未回答2（合计 45）。明细里“账号地区 美国”不再有“统计口径”小字。存活时长仍显示「34 天 30–89 天」，点选后“已选：34 天”，列表收敛为 2 份。前端构建通过。

- 用户要求重做，指出三点：功能太少（只能看）、结构不好、标题下一大段废话。用户确认的需求：交叉透视 + 两组人群对比 + 筛选后导出 + 逐份查阅与标记 + 时间维度；标签用预置加自定义；异常状态与提交月份要能当交叉维度；几十到几百份规模；左侧常驻筛选 + 右侧结果；去掉站点外壳；**一次做完不分期**。

- 决策：后端只加最小写入面——`0003_survey_annotations.sql` 新建标注表（submission_id 主键外键、tags JSON、note、updated_at），`internal/study/annotations.go` 提供 `PUT /api/admin/annotations`，请求为 `{ids, addTags, removeTags, note}`，支持批量与“清空即删行”。前端拆成 `components/admin/`：`adminModel.ts`（筛选、选项统计、交叉、对比、趋势、CSV/JSON、URL 状态）+ 条件栏 / 明细表 / 交叉 / 对比 / 时间 / 抽屉六个组件，页面只负责装配与状态。`App.vue` 改为按 hash 的路径部分匹配路由，并让管理面板整页脱离 `.site-shell`（也不再拉取公开研究接口）。

- 决策：统计口径只在数值分组（存活时长、人数、并发、IP 风险）标注；地区合并对原始数据没有解释价值，完全不再显示。对比视图的 φ 用与公开页相同的公式，并明确写“未做显著性检验、未控制混杂”。

- 测试结果（后端）：`TestAnnotationsAreStoredWithoutTouchingTheSubmission` 验证标签顺序、备注、批量和问卷原文未被改动；`TestAnnotationRemovalAndClearing` 验证移除单个标签不清空其他字段、全清后删行；`TestAnnotationRequestsAreValidated` 覆盖空请求、非法 id、超长标签/备注、未知字段、尾随内容与不存在的问卷编号；`TestAnnotationRequiresTheAdminSession` 验证匿名写入被拒。发现并修正一处路由遗漏：`serveAdmin` 里加了 `annotations` 分支，但外层 `ServeHTTP` 的 case 列表没加，导致 404。

- 修正：备注与超长输入最初在 `normalize()` 里静默截断，改为校验并返回 400——截断是调用方无法察觉的数据丢失。

- 前端修掉的真实缺陷：① 交叉表页脚总计取的是“样本数”，而多选维度的行列合计会大于样本数，已改为列合计之和（`grand`）；② 地址栏写入用了 `#${location.hash}`，每写一次多一个 `#`，刷新后路由失配导致白屏；③ 对比页播种 A 组时用了 `structuredClone`，对 Vue 响应式代理抛异常，导致切页签无反应，改为 JSON 深拷贝 `cloneFilters`；④ 对比页签直接点击时不会播种 A 组，补了 `selectTab`。

- 验证结果（前端，浏览器实测）：未登录只显示登录卡片、无站点导航；登录后 45 行、表头可排序；点“降智”→17/45，再加“账号地区=菲律宾”→10/45，刷新后条件与结果从地址栏完整恢复；交叉透视显示 套餐级别 × 异常状态，页脚合计 10/6/16 与行合计一致并给出多选重复计数提示；时间视图按天出柱、点击可加时间段；两组对比 A=10（降智·菲律宾）B=35（其余全部），23 道题按差值排序并给出 φ；详情抽屉打开第 35 份，预置标签“可疑”和备注均写入成功且表格行同步显示；导出 CSV 与 JSON 通过真实按钮验证（文件名、BOM、23 列、45 条记录、完整字段）。

- 工具结果：一开始用 `page.evaluate` 给 `URL.createObjectURL`/`Blob` 打桩来观察导出，一直观察不到，误以为按钮坏了；实际是 `page.evaluate` 运行在隔离世界，桩打在了隔离世界的全局对象上。改用 `tab.evaluate`（页面全局）后 `probe=1`、文件名与内容都正确。这次的教训：验证页面副作用要打在页面主世界的全局对象上。

- 收尾：预览库里测试用的标注已通过接口清空（`survey_annotations` 无残留行）；`go test ./...`、`go vet`、CI 同款 gofmt、`vue-tsc` + Vite 构建全部通过；README 与 `docs/api.md` 已按新面板与标注接口重写。

- 下一步：等用户体验；改动仍未提交、未推送。

- 用户反馈：排序「只看数字」。查证属实，`Number.parseFloat("2 年")` 得 2，比 `parseFloat("34 天")` 的 34 小，于是「2 年」排在「34 天」前面；同一个 `parseFloat` 还让「10 天」排在「2 天」前面，且数值与文本混在一起比较（`NaN` 落到 `localeCompare`）不是全序。

- 决策：`adminModel` 新增 `magnitude()`，把 `数值 + 单位` 按小时/天/星期/月/年换算到天（与统计页同一套换算：1/24、1、7、30、365），认不出的单位返回 null；新增 `columnKind()` 按整列决定是数值列还是文本列——**按列决定而不是按单元格两两决定**，避免同列内数字和词混比；`compareCells()` 数值列把无值项（`unknown`）固定放末尾。表格比较器里「空单元格」和「无值」都在正反两个方向都排末尾。

- 修正：`columnKind` 阈值最初定 80%，实测「最高账号并发」是 27 个数字 + 11 个 `unknown`（71%），被判成文本列，于是又出现 `10 < 2`；改为 50%（真文本列的数字占比是 0%，不受影响）。

- 修正：降序时把 `compareCells` 的返回值整体取反，会把原本固定在末尾的 `unknown` 顶到最前面，改为在取反之前先处理无值分支。

- 修正：点击一个新列默认给了降序，改为默认升序（再点一次翻转）；提交时间的初始状态仍是最新在前。

- 验证结果：`magnitude("2 年")=730`、`magnitude("34 天")=34`、`magnitude("24 小时")=1`、`magnitude("unknown")=null`、`magnitude("Plus")=null`。浏览器实读表格：存活时长升序 `0 天 / 0 月 / 1 天×3 / 2 天 / 3.1 天 / 4 天 / 5 天 / 8 天`，降序 `3 年 / 2 年×2 / 1 年×3`；最高并发升序 `2,2,2,3…`，降序 `20 / 15 / 10×3 / 8`，两个方向的「未回答」都在末尾；账号地区按拼音升序（菲律宾、韩国、美国…）；提交时间按时间升序；编号 1,2,3。筛选侧同类选项也修正：存活时长芯片在相同份数下按 1 天 → 1 年 → 34 天 → 3 月 → 2 年 的真实时长排序。

- 工具结果：中途一次读到 `body: false` 一度以为登录态坏了，实为改动前端文件时 HMR 留下的陈旧状态，刷新后 45 行正常，不是缺陷。

- 用户提供远端导出 `study-export.sqlite`（83 份，2026-09-09 至 09-12，迁移停在 0002，说明远端仍是旧构建）。已复制为 `data/study-remote-20260912.sqlite` 并把 8080 后端切到它；`0003_survey_annotations.sql` 在真实远端数据上应用成功，公开统计的 `lastSubmissionId` 从 45 变成 83。原 `survey-preview-20260908.sqlite` 未删除，保留作对比。

- 分析结论：唯一在分层后依然稳健的因素是**套餐级别（Pro 20X）**——降智 29/55=52.7% vs 其他 4/28=14.3%，φ=+0.371，Fisher p=0.0008，OR 6.7（95% CI 2.0–21.9）；按「是否分发」分层 MH OR=5.8，按「提交日期」分层 MH OR=7.9，且存在剂量关系（Plus 5.9% → Pro 5X 37.5% → Pro 20X 52.7%）。「是否分发」单独看有弱关联（46.4% vs 37.5%），但一控制套餐就归零（Pro 20X 内 52.2% vs 53.6%）。

- 三条必须一起说出口的限制：① 我是扫描了 45 个两臂均≥10 的选项才挑出这个的，置换检验下单次重排的最大 |φ| 达到 0.371 的比例约 0.40，所以它是强假设而非已确立的发现；② 问卷无法区分「掉智」和「额度用光」，Pro 20X 用量本来就最大，即使模型质量毫无变化也会看到更高的自报降智率——现有题目排除不掉这个解释；③ 降智与风控高度共现（φ=+0.410，p=3.1e-4，风控者中 67.9% 同时报降智），当作两个独立结果会重复计数；日间波动 47.9pp 与套餐极差 46.8pp 同量级，提示外部事件（服务端波动）可能是主导。

- 封号只有 7 例，任何因素都达不到稳定显著，不应下结论。







