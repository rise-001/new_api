# 二开改动记录

本文档记录此仓库相对上游 `QuantumNous/new-api` 的定制内容，供后续开发者和 AI 快速了解上下文。

## 仓库与分支约定

- `upstream`：原作者仓库 `https://github.com/QuantumNous/new-api.git`
- `origin`：二开仓库 `https://github.com/rise-001/new_api.git`
- `main`：只同步上游代码，不直接编写二开功能
- `custom`：保存和部署二开功能
- 同步上游时，先快进 `main`，再将 `main` 合并到 `custom`

## 2026-07-27：模型性能数据仅管理员可见

### 需求

在“设置 -> 顶部导航设置 -> 模型广场设置”中增加一个开关。开启后：

- 管理员和超级管理员仍可看到全部模型详情
- 普通用户看不到“概览”顶部的 TPS、平均延迟和成功率
- 普通用户看不到“性能”页签及其内容
- 普通用户看不到模型卡片右下角的延迟、吞吐和状态
- 普通用户仍可查看“概览”的其他内容和“API”页签
- 关闭开关时保持原有行为，所有用户都能看到性能数据

### 配置

配置保存在现有的 `HeaderNavModules` JSON 中：

```json
{
  "pricing": {
    "enabled": true,
    "requireAuth": false,
    "overviewMetricsAdminOnly": true
  }
}
```

`overviewMetricsAdminOnly` 默认值为 `false`，以兼容旧配置。

管理员判定规则为用户角色大于或等于 `ROLE.ADMIN`，因此管理员和超级管理员都具有访问权限。

### 实现结构

- `web/src/features/system-settings/maintenance/header-navigation-section.tsx`
  - 在模型广场设置中渲染新开关并保存配置
- `web/src/features/system-settings/maintenance/config.ts`
  - 定义后台设置页使用的配置类型、默认值和兼容解析
- `web/src/lib/nav-modules.ts`
  - 定义前台使用的配置类型、默认值和兼容解析
- `web/src/features/pricing/hooks/use-model-performance-access.ts`
  - 统一计算当前用户是否能查看模型性能数据
- `web/src/features/pricing/components/model-overview-summary.tsx`
  - 独立渲染概览统计；无权限时不挂载，也不会发起概览指标请求
- `web/src/features/pricing/components/model-details-tab-list.tsx`
  - 有权限时显示“概览/性能/API”，无权限时显示“概览/API”
- `web/src/features/pricing/components/model-details.tsx`
  - 根据权限条件挂载性能页签内容
- `web/src/features/pricing/components/model-card-grid.tsx`
  - 根据权限隐藏模型卡片性能摘要，并阻止无权限时请求性能汇总
- `web/src/i18n/locales/*.json`
  - 已补齐英文、简体中文、繁体中文、法语、日语、俄语和越南语文案

### 模型卡片补充改动

补充提交：`89fe12c2 fix: hide model card performance from users`

- 开启 `overviewMetricsAdminOnly` 后，普通用户的模型卡片不会渲染右下角“延迟、吞吐、状态”摘要
- 无权限时 `perf-metrics-summary` 查询被禁用，避免只隐藏界面但仍请求卡片性能数据
- 管理员和超级管理员保持原有显示；关闭开关时普通用户也保持原有显示
- 本次提交只修改 3 个文件：业务组件、回归测试和本文档；实际生产代码改动集中在 `model-card-grid.tsx`
- 对应公开镜像从 `ghcr.io/rise-001/new-api-custom:latest` 获取，固定功能版本可使用 `ghcr.io/rise-001/new-api-custom:sha-89fe12c`

### 测试

相关回归测试：

- `web/src/features/pricing/components/__tests__/model-performance-visibility.test.tsx`
- `web/src/features/system-settings/maintenance/__tests__/header-navigation-config.test.ts`

已验证：

- 9 条回归测试通过
- 本次涉及文件的定向 TypeScript 类型检查通过
- 涉及文件 lint 通过
- 涉及文件格式检查通过
- i18n 同步报告无缺失
- Rsbuild 生产构建通过

当前全量 TypeScript 检查会被仓库其他位置的已有问题阻断：`web/src/features/setup/setup-wizard.tsx` 导入了 `admin-step.tsx` 没有导出的 `AdminStep`。该问题与模型性能权限改动无关；后续合并上游时应重新检查它是否已修复。

### 权限边界

当前功能控制前端展示和前端指标请求挂载。后端 `/api/perf-metrics` 接口本身没有根据该开关增加管理员鉴权。

如果后续要求普通用户即使手动调用接口也不能取得性能数据，需要在后端性能指标接口增加与 `HeaderNavModules.pricing.overviewMetricsAdminOnly` 对应的权限校验，同时确保公开模式关闭该限制。

## 同步上游流程

```powershell
git switch main
git fetch upstream
git merge --ff-only upstream/main
git push origin main

git switch custom
git merge main
git push origin custom
```

同步时重点检查以下冲突位置：

- 模型详情页签结构
- 模型卡片性能摘要
- `HeaderNavModules` 配置类型和解析逻辑
- 模型性能指标组件及 API
- 7 个前端语言文件

## 给后续 AI 的提示词

开始新的二开任务时，可以先发送：

```text
请先完整阅读 AGENTS.md、web/AGENTS.md 和 CUSTOM_CHANGES.md，了解项目规范、分支策略和已有二开功能。后续修改必须保留现有二开行为，并评估与 upstream 更新的兼容性。
```

## 2026-07-27：发布二开 Docker 镜像

`.github/workflows/custom-ghcr.yml` 会在 `custom` 分支每次推送后自动构建并发布以下多架构镜像（支持 `linux/amd64` 和 `linux/arm64`）：

- `ghcr.io/rise-001/new-api-custom:latest`
- `ghcr.io/rise-001/new-api-custom:custom`
- `ghcr.io/rise-001/new-api-custom:sha-<提交短哈希>`

工作流使用 GitHub 自动提供的 `GITHUB_TOKEN`，不需要配置 Docker Hub 密钥。首次发布后，如果希望未登录的服务器直接拉取镜像，需要在 GitHub 仓库的 Packages 页面把软件包可见性设置为 Public。

当前 `ghcr.io/rise-001/new-api-custom` 已验证可以匿名拉取。不要改回 `ghcr.io/rise-001/new_api`：该名称曾因账号下同名包权限冲突返回 `permission_denied: write_package`，因此二开镜像使用独立名称 `new-api-custom`。

使用现有完整 Compose 配置部署二开镜像：

```powershell
docker compose -f docker-compose.yml -f docker-compose.custom.yml pull
docker compose -f docker-compose.yml -f docker-compose.custom.yml up -d
```

部署前必须修改 `docker-compose.yml` 中 PostgreSQL 和 Redis 的默认密码，并同步修改 `SQL_DSN`、`REDIS_CONN_STRING`。生产环境还应设置随机的 `SESSION_SECRET`。

如果镜像保持 Private，需要先使用具有 `read:packages` 权限的 GitHub Personal Access Token 登录：

```powershell
$env:GHCR_TOKEN | docker login ghcr.io -u rise-001 --password-stdin
```

### 当前服务器部署方式

- 对外端口：`8899`
- SQLite 数据目录：`/opt/new-api/data`
- 日志目录：`/opt/new-api/logs`
- 容器名称：`new-api`

首次部署：

```bash
docker run -d --name new-api --restart always -p 8899:3000 -e TZ=Asia/Shanghai -e ERROR_LOG_ENABLED=true -v /opt/new-api/data:/data -v /opt/new-api/logs:/app/logs ghcr.io/rise-001/new-api-custom:latest --log-dir /app/logs
```

更新镜像并重建容器：

```bash
cd /opt/new-api && docker pull ghcr.io/rise-001/new-api-custom:latest && docker stop new-api && docker rm new-api && docker run -d --name new-api --restart always -p 8899:3000 -e TZ=Asia/Shanghai -e ERROR_LOG_ENABLED=true -v /opt/new-api/data:/data -v /opt/new-api/logs:/app/logs ghcr.io/rise-001/new-api-custom:latest --log-dir /app/logs
```

仅执行 `docker restart new-api` 不会切换到新镜像，必须拉取镜像并重建容器。删除容器不会删除上述宿主机挂载目录中的数据和日志。

## 2026-08-09：个人消费清单导出

初始提交：`9811970a feat: add personal consumption exports`

初始异步版本镜像（已被当前实现替代）：`ghcr.io/rise-001/new-api-custom:sha-9811970`

### 功能说明

在个人板块增加“消费清单”入口，用户可以按时间范围和 API Key 筛选自己的消费记录，并直接下载生成的 Excel 文件。

- 导出内容包含请求时间、API Key、模型、输入 Token、输出 Token、消费金额等消费明细
- Excel 按 API Key 分工作表，并附带每日汇总、退款、总计和模型统计
- 单次导出时间范围最长 31 天，最多导出 300,000 条消费记录
- 后端分页读取消费日志并流式写入工作表，避免一次性在内存中保存全部明细
- Excel 先写入系统临时文件，HTTP 响应完成后立即关闭并删除，不上传 S3/MinIO，也不写入数据库
- 同一进程内每个用户同时只能生成一个导出，接口继续使用用户认证和关键操作限流
- 启动后每分钟清理超过 24 小时的同名前缀遗留临时文件，用于处理进程异常退出场景

### 实现结构

- `web/src/features/consumption-exports/`
  - 消费清单页面、创建导出对话框、Blob 响应解析和浏览器下载逻辑
- `web/src/routes/_authenticated/profile/index.tsx`
  - 增加 `/profile?view=consumption` 个人消费清单视图
- `web/src/hooks/use-sidebar-data.ts`
  - 在个人板块增加“消费清单”入口
- `web/src/features/profile/components/sidebar-modules-card.tsx`
  - 支持在个人侧边栏模块设置中控制消费清单入口
- `controller/consumption_export.go`
  - 同步生成导出文件并通过附件响应返回，响应结束后删除临时文件
- `service/consumption_export.go`
  - 范围与数量校验、临时文件生命周期、消费统计和遗留临时文件清理
- `service/xlsx_writer.go`
  - 将工作表 XML 流式写入 XLSX ZIP，支持大批量明细
- `model/consumption_export.go`
  - 消费日志计数、游标分页查询和旧异步导出数据清理
- `model/system_task.go`
  - 已移除旧异步导出专用的用户任务、取消状态和 `user_id` 模型字段
- `web/src/i18n/locales/*.json`
  - 已同步英文、简体中文、繁体中文、法语、日语、俄语和越南语文案

### API

- `POST /api/consumption-export/`：生成并直接返回当前用户的 Excel 导出文件

旧版任务列表、下载、取消和删除接口均已移除。

### 数据库影响

当前导出实现不会新增数据库表或字段，也不会把 Excel 内容写入数据库。应用启动迁移时会清理旧异步版本遗留内容：

- 删除 `system_task_locks` 中类型为 `consumption_export` 的旧锁记录
- 删除 `system_tasks` 中类型为 `consumption_export` 的旧任务记录
- 执行 `DROP TABLE IF EXISTS consumption_export_files`

清理语句兼容 SQLite、MySQL 和 PostgreSQL，不会删除其他类型的系统任务，也不会修改用户和消费日志。未部署过初始异步版本的环境中，旧任务和旧文件表均不存在，因此清理操作为空操作。旧版本曾加入的 `system_tasks.user_id` 已从 GORM 模型移除，新数据库不会创建该字段；已经运行过旧异步版本的数据库可能继续保留这个未使用的物理列，本次不执行跨数据库删列。

### 部署注意

- 临时目录必须可写，并需要预留单个大体量 XLSX 的磁盘空间
- 30 万条导出可能耗时较长，反向代理和网关的请求超时应覆盖文件生成时间
- 多实例环境应同步替换所有旧实例，避免旧实例继续访问已删除的文件表

### 测试与验证

已验证：

- 前端 TypeScript 类型检查通过
- Blob 下载响应单元测试通过
- 涉及文件 lint 和格式检查通过
- 7 个前端语言文件同步检查无缺失、无多余键和未翻译项
- Rsbuild 生产构建检查通过
- `git diff --check` 通过

当前本地终端的 PATH 中没有 Go 和 Bun；前端检查使用仓库现有二进制和 Node 执行，未能运行 Go 格式化、后端编译及回归测试。

## 2026-08-10：默认开启 IP 记录

提交：`205e6096 fix: enable IP logging by default`

### 功能说明

- 新用户默认在用量日志和错误日志中记录客户端 IP
- 历史用户的设置中如果缺少 `record_ip_log` 字段，按开启处理
- 用户主动关闭后会显式保存 `"record_ip_log":false`，后续读取仍保持关闭
- 更新其他个人设置时，如果请求未携带 `record_ip_log`，保留用户已有选择
- 前端开关的初始值和缺省回退值与后端统一为开启

### 实现位置

- `dto/user_settings.go`：定义统一的用户设置默认值
- `model/user.go` 和 `model/user_cache.go`：读取及初始化用户设置时应用默认值
- `controller/user.go`：区分“未传递字段”和“显式传递 `false`”
- `web/src/features/profile/components/tabs/notification-tab.tsx`：默认开启“记录 IP 地址”开关
- `model/user_update_test.go`：覆盖历史设置默认开启和用户主动关闭的回归场景

### 影响范围

本次不需要数据库迁移，只调整用量日志和错误日志的 IP 记录默认值。登录、操作审计和充值等日志的 IP 记录逻辑不受影响。

### 测试与验证

- 已通过 `git diff --check`

## 2026-08-24：Gemini 图片模型无图不收费

提交：`f16e64b0 fix: refund Gemini image requests without image output`

### 背景

Gemini 图片模型可能返回 token 使用量，但响应中没有实际图片。token 数量不能作为图片生成成功的依据；如果继续走按次计费流程，会在没有图片结果时扣费。

### 成功判定

对于 `gemini-3.1-flash-image`、`gemini-3.1-flash-image-preview` 等 Gemini 图片模型，只有满足以下条件才算生成成功：

```text
candidates[].content.parts[].inlineData 存在
inlineData.mimeType 以 image/ 开头
inlineData.data 非空
```

对于 `imagen-*` 模型，要求至少有一个 `predictions[]` 项同时满足：

```text
raiFilteredReason 为空
bytesBase64Encoded 非空
```

### 失败处理

如果没有可用图片，relay 返回 `502 Bad Gateway`，不进入成功计费流程。控制器发现 relay 错误后，会调用 `BillingSession.Refund` 退还钱包、订阅和令牌预扣额度。

流式和非流式 Gemini 响应均执行相同判断；流式响应统计的是实际收到的、带有非空图片数据的图片数量。

### 相关代码

- `relay/channel/gemini/relay-gemini.go`
  - `geminiResponseInlineImageCount`：统计可用 inline image
  - `isGeminiImageModel`：识别 Gemini 图片模型
  - `GeminiChatHandler`、`geminiStreamHandler`：无图失败处理
  - `GeminiImageHandler`：处理 `imagen-*` 的 predictions
- `relay/channel/gemini/relay-gemini-native.go`
  - Gemini 原生接口的无图失败处理
- `controller/relay.go`
  - relay 失败时触发预扣额度退款
- `relay/channel/gemini/relay_gemini_usage_test.go`
  - 无图片、过滤图片和无效图片数据的回归测试

### 验证

已通过 `git diff --check`。开发环境未安装 Go 工具链，因此未执行 `go test`；部署环境应执行：

```bash
go test ./relay/channel/gemini
```
- 当前本地终端的 PATH 中没有 Go 和 Bun，未能执行后端回归测试和前端 lint

## 2026-08-10：隐藏普通用户日志中的模型映射

提交：`32506343 fix: hide model mapping from user logs`

### 功能说明

- 普通用户查询自己的日志时，不再返回 `Other` 中的 `is_model_mapped` 和 `upstream_model_name`
- 管理员查询日志的原始数据不受影响，仍可用于排查模型映射和上游路由问题
- `model_price` 等非管理员专属的计费字段继续保留

### 实现位置

- `model/log.go`
  - 在现有 `formatUserLogs` 用户日志格式化流程中移除模型映射字段
- `model/log_format_test.go`
  - 覆盖模型映射字段被移除、普通计费字段仍保留的回归场景

### 影响范围与升级注意

本次不需要数据库迁移，也不改变日志入库内容，只调整普通用户日志接口的返回结果。同步上游时需要重点检查 `formatUserLogs` 及日志脱敏字段列表，避免合并后重新暴露模型映射信息。

## 2026-08-10：自定义菜单管理

提交：`ba808740 feat: add custom menu management`

### 功能说明

- 在系统设置中增加“自定义菜单管理”页面，仅管理员可以维护菜单配置
- 菜单可以显示在聊天区域侧边栏或公共顶部导航栏
- 聊天区域菜单支持站内嵌入或新标签页打开；顶部导航菜单只能在新标签页打开
- 菜单名称直接使用管理员填写的文本，不作为固定 i18n 文案翻译
- 单次最多配置 50 个菜单项，后端校验 ID、名称、URL、显示位置和打开方式

### 配置

配置保存在系统选项 `CustomMenuItems` 中，值为 JSON 数组：

```json
[
  {
    "id": "support",
    "url": "https://support.example.com",
    "name": "Support",
    "location": "chat",
    "open_mode": "embed"
  }
]
```

支持的字段值：

- `location`：`chat` 或 `top`
- `open_mode`：`embed` 或 `new_tab`
- `top` 位置只接受 `new_tab`

### 实现结构

- `setting/custom_menu.go`
  - 定义配置结构、数量限制和后端 URL/字段校验
- `model/option.go`
  - 初始化 `CustomMenuItems`，并在更新系统选项时执行校验
- `controller/misc.go`
  - 将菜单配置加入 `/api/status` 响应，供公共顶部导航和登录后侧边栏读取
- `web/src/features/system-settings/custom-menus/`
  - 自定义菜单列表、编辑对话框、删除确认和保存逻辑
- `web/src/lib/custom-menus.ts`
  - 前端配置解析、序列化和安全 URL 检查
- `web/src/hooks/use-custom-menus.ts`
  - 从系统状态中读取菜单配置
- `web/src/hooks/use-sidebar-data.ts` 和 `web/src/hooks/use-top-nav-links.ts`
  - 分别挂载聊天区域菜单和顶部导航菜单
- `web/src/routes/_authenticated/custom-menu/$menuId.tsx`
  - 渲染登录后的嵌入页面

### 权限与安全边界

菜单配置由管理员维护，但当前完整 `CustomMenuItems` 会通过公开的 `/api/status` 返回，因此菜单 URL 不能视为秘密。嵌入页面仍要求用户登录。后续如果需要配置仅登录用户可见的内部地址，应将聊天区域菜单改由登录后接口返回，而不是继续放在公共状态接口中。

### 测试与升级注意

- `setting/custom_menu_test.go` 覆盖 JSON、重复 ID、URL 协议、位置和打开方式校验
- `web/src/lib/__tests__/custom-menus.test.ts` 覆盖前端解析和序列化
- 已补齐 7 个前端语言文件中的固定管理界面文案
- 同步上游时重点检查 `/api/status`、公共头部、侧边栏数据、系统设置导航、路由树和 7 个语言文件
- `web/src/routeTree.gen.ts` 属于生成文件，解决路由冲突后应通过项目路由工具重新生成

## 2026-08-10：允许嵌入菜单使用来源存储

提交：`34d3ca6d fix: support stateful embedded custom menus`

### 功能说明

在自定义菜单 iframe 的 `sandbox` 中增加 `allow-same-origin`，使被嵌入应用能够以自身来源访问 Cookie、Local Storage 等来源存储，用于保持登录状态和应用配置。

### 安全边界

当前 iframe 同时允许 `allow-same-origin` 和 `allow-scripts`，并允许弹窗逃离 sandbox。该组合只适用于管理员明确信任的嵌入地址，不应嵌入用户可控或来源不明的页面。同步上游或继续扩展菜单功能时，不要在缺少域名限制和安全评估的情况下继续放宽 iframe 权限。

本次只修改 `web/src/routes/_authenticated/custom-menu/$menuId.tsx`，不涉及后端接口或数据库迁移。

## 2026-08-10：错误页反馈入口改动（待纠正）

提交：`615bfe39 chore(web): remove issue feedback from error page`

### 当前改动

- 从通用错误页移除了持续出错时前往 GitHub Issues 反馈的提示
- 移除了“Report an issue”按钮及其项目 Issues 链接
- 保留“Go Back”和“Back to Home”操作

### 项目规范冲突

当前 `AGENTS.md` 明确保护所有与项目和作者组织相关的引用、归属和元数据，不允许删除或替换。该提交删除了错误页中受保护的项目 Issues 引用，因此不能作为后续更新需要保留的合法二开行为；在下一次合并或发布前必须恢复该项目引用，并确保上游更新不会再次删除其他受保护信息。

本次只修改 `web/src/features/errors/general-error.tsx`，不涉及后端接口或数据库迁移。

## 2026-08-20：自定义菜单支持个人板块

### 功能说明

- 自定义菜单的“展示板块”新增“个人中心”选项。
- 选择个人中心后，菜单项显示在侧栏个人板块的现有菜单下方。
- 个人板块菜单支持内嵌打开和新标签页打开。
- 个人板块菜单遵循个人侧栏模块的整体可见性配置。

### 配置格式

`CustomMenuItems` 中的菜单项新增以下位置值：

```json
{
  "id": "account",
  "url": "https://account.example.com",
  "name": "Account",
  "location": "personal",
  "open_mode": "embed"
}
```

`location` 支持 `chat`、`personal` 和 `top`；`personal` 与 `chat` 一样支持 `embed` 和 `new_tab`，`top` 仍只支持 `new_tab`。

### 实现位置

- `setting/custom_menu.go`：扩展后端位置校验，允许 `personal`。
- `web/src/lib/custom-menus.ts`：扩展前端配置解析。
- `web/src/features/system-settings/custom-menus/`：新增个人中心展示板块选项及列表标签。
- `web/src/hooks/use-sidebar-data.ts`：将个人菜单追加到个人侧栏分组末尾。
- `web/src/routes/_authenticated/custom-menu/$menuId.tsx`：允许个人板块菜单使用内嵌页面。
- `setting/custom_menu_test.go`、`web/src/lib/__tests__/custom-menus.test.ts`：补充个人板块配置回归覆盖。

### 验证

- 前端 TypeScript 类型检查通过。
- 相关文件 lint 和格式检查通过。
- 自定义菜单单元测试通过。
- i18n 同步通过，未新增缺失翻译键。
- Rsbuild 生产构建通过。
- 当前环境未安装 Go 工具链，未运行后端 Go 测试。

## 2026-08-26：邀请额度划转记录（已被后续版本替代）

> 历史记录：该版本曾将“划转记录”用于展示用户把邀请奖励额度划转到可用额度的操作。自 2026-09-04 起，页面和接口已改为展示管理员增加用户额度的管理日志；本节仅保留用于说明历史实现和数据库兼容背景。

提交：`ae20ef4f feat: add affiliate transfer records`

### 历史功能说明

- 用户将邀请额度划转到可用额度后，系统新增一条划转审计日志，记录用户、时间和划转额度。
- 管理员可从个人板块的“划转记录”入口查看全部用户的邀请额度划转历史。
- 列表支持按用户名、开始时间和结束时间筛选，并按每页 20 条分页展示。
- “划转记录”侧栏模块默认开启，也可通过现有个人侧栏模块配置控制入口显示。
- 页面和固定文案已同步英文、简体中文、繁体中文、法语、日语、俄语和越南语。

### 历史 API 与权限

- `GET /api/user/transfer`：分页查询邀请额度划转日志。
- 查询参数包括 `p`、`page_size`、`username`、`start_timestamp` 和 `end_timestamp`。
- 接口位于现有 `AdminAuth` 路由组中，仅管理员可以访问；前端路由入口同样要求 `ROLE.ADMIN`。

### 历史数据存储与一致性

- 本次不新增数据库表或字段，划转记录复用日志数据库中的 `logs` 表。
- 日志类型为 `LogTypeManage`，`content` 固定为 `Affiliate quota transfer`，划转额度同时写入 `quota` 和结构化的 `Other.op.params.quota`。
- 查询兼容主日志数据库和 ClickHouse 日志数据库，并沿用现有用户名精确过滤和 ClickHouse 展示 ID 处理。
- 审计日志在额度划转事务提交成功后写入。日志写入失败会记录系统错误，但不会回滚已经完成的额度划转。
- 部署后只能记录新发生的划转，既有划转不会自动补录。

### 历史实现位置

- `model/user.go`：额度划转事务提交成功后触发审计日志写入。
- `model/log.go`：创建划转日志，并提供筛选、计数和分页查询。
- `controller/log.go`、`router/api-router.go`：增加管理员查询接口。
- `web/src/features/transfer-records/`：实现查询请求、筛选条件、记录表格和分页。
- `web/src/routes/_authenticated/transfer-records/index.tsx`：注册 `/transfer-records` 页面。
- `web/src/hooks/use-sidebar-data.ts`、`web/src/hooks/use-sidebar-config.ts`：增加管理员入口及侧栏模块配置。
- `web/src/features/profile/components/sidebar-modules-card.tsx`：增加“划转记录”模块开关。
- `web/src/i18n/locales/*.json`：补齐 7 种前端语言文案。

### 历史升级注意

同步上游时重点检查 `TransferAffQuotaToQuota` 的事务提交位置、日志表筛选逻辑、用户管理路由组、个人侧栏配置和生成的 `web/src/routeTree.gen.ts`，避免划转成功后漏记日志或放宽查询权限。

## 2026-09-04：划转记录改为管理员加额度日志

提交：`17b9e931 fix: show manual quota addition records`

### 变更说明

- “划转记录”不再查询邀请奖励额度划转日志，改为查询管理员通过用户管理执行“增加额度”产生的管理日志。
- 因此当前页面名称暂时仍为“划转记录”，实际含义是“管理员增加用户额度记录”，不是用户邀请奖励额度划转记录。
- 列表中的用户名为被增加额度的目标用户，而不是执行操作的管理员；按用户名搜索时同样按目标用户精确匹配。
- 页面默认查询本地时间当天 `00:00` 至次日 `00:00`，后端采用开始时间包含、结束时间不包含的区间，避免跨日边界重复。
- 页面仍支持自定义用户名、开始时间、结束时间以及每页 20 条分页查询。
- 列表额度直接显示审计日志写入时的格式化金额，保持与管理日志页面展示一致。

### 日志兼容

- 当前管理审计日志通过英文 `content` 前缀 `Increased user quota by ` 识别，目标用户 ID 从 `Other.op.params.target_user_id` 读取。
- 兼容旧版中文 `content` 前缀 `管理员增加用户额度 `；旧日志继续使用日志自身的用户 ID 和用户名作为目标用户。
- 管理员给自己增加额度时没有 `target_user_id`，查询逻辑会按日志归属识别为本人操作，不会把该管理员给其他用户增加额度的记录混入本人结果。
- 查询继续支持主日志数据库和 ClickHouse 日志数据库，不新增数据表或迁移。
- 仅统计 `user.quota_add`（增加额度）日志，不包含扣减额度、覆盖额度或旧的邀请额度划转日志。

### 实现位置

- `model/log.go`：筛选管理员增加额度日志，兼容新旧格式，解析目标用户和格式化额度。
- `controller/log.go`、`router/api-router.go`：`GET /api/user/transfer` 改用管理员增加额度日志查询。
- `web/src/features/transfer-records/`：调整接口记录结构、默认时间范围及表格展示。
- `web/src/features/profile/components/sidebar-modules-card.tsx`：更新侧栏模块说明。
- `web/src/i18n/locales/*.json`：同步 7 种语言的页面说明。

### 回归覆盖

- 后端用例覆盖新版结构化审计日志、旧版中文日志、目标用户名筛选、管理员本人加额度以及结束时间排除边界。
- 前端用例覆盖默认时间范围为当天零点至次日零点。

### 功能确认

- 当前接口为 `GET /api/user/transfer`，位于 `AdminAuth` 路由组，仅管理员及超级管理员可访问。
- 当前页面入口位于个人侧栏，页面和侧栏模块名称仍为“划转记录”，说明文字为“管理员增加额度记录”。
- 列表展示目标用户、审计时间和审计日志写入时的格式化额度；支持目标用户名、开始时间、结束时间和每页 20 条分页。
- 个人设置「侧栏个性化」中的“划转记录”模块开关仅对管理员及超级管理员显示，详见下面 2026-09-21 的条目。

## 2026-09-19：消费导出改为单遍扫描并补齐复合索引

### 问题

消费导出在反向代理上返回 `504 Gateway Time-out`。原因有两个，都在服务端：

1. `logs` 表没有同时覆盖 `user_id` 和 `created_at` 的索引。导出查询是 `WHERE user_id = ? AND type IN (2,6) AND created_at BETWEEN ? AND ? ORDER BY created_at, id`，优化器只能在 `idx_user_id_id` 和 `idx_created_at_id` 之间二选一：走前者要读该用户全部历史日志并对 `created_at` 做 filesort，且游标分页的每一批都要重做一次；走后者要扫该时间段内全站所有用户的日志再过滤。
2. 同一批数据被读了两遍以上。先全量扫一遍算令牌、模型和每日汇总，再扫一遍写明细；勾选“每个 API 令牌单独生成一个工作表”时，还要按令牌名各扫一遍。

### 变更说明

- `logs` 表新增复合索引 `idx_user_id_created_at_id`（`user_id`, `created_at`, `id`）。等值定位用户、范围收窄时间、排序顺序与 `ORDER BY` 一致，游标分页变成索引内 seek，消除 filesort 和跨用户扫描。
- 导出改为单遍扫描：一次遍历同时累加三张汇总表并写出明细行，不再回查数据库。
- 明细行在扫描过程中直接渲染成工作表 XML，写入每工作表一个临时文件；扫描结束后再组装 ZIP。内存占用与记录数无关。
- 勾选按令牌分表时，先用一次 `GROUP BY token_name` 聚合查询确定工作表清单和每表行数，因此序号列仍然跨工作表连续，导出内容与改动前一致。
- 新增上限：按令牌分表时最多 200 个工作表，超出时返回提示要求关闭该选项。此前没有上限，但那种规模本来就必定超时。

### 行为与产物

- 生成的 Excel 内容不变：工作表顺序、名称、列、合计行、序号连续性均与改动前一致。
- 导出期间的临时文件从 1 个变为 1 个工作簿加每工作表 1 个行文件，全部在响应结束后删除。
- 导出中若检测到记录数与预先统计不一致（生成期间有新日志写入），仍然报错要求重试，错误信息中增加了工作表名。

### 实现位置

- `model/log.go`：新增 `idx_user_id_created_at_id` 复合索引标签。
- `model/consumption_export.go`：新增 `CountConsumptionExportLogsByToken` 聚合查询；移除重构后不再使用的 `TokenName` 过滤字段。
- `service/xlsx_writer.go`：`xlsxSheet` 以 `RawRows`/`RowCount` 接收预渲染的行；新增 `xlsxRowWriter` 与共享的 `appendXLSXRow`。
- `service/consumption_export.go`：新增 `consumptionExportDetail`/`consumptionExportDetailSheet` 负责在单遍扫描中分流明细行；移除 `consumptionExportDownloadSheets`、`buildStreamingConsumptionDetailSheet` 和 `writeConsumptionExportRecords`。

### 数据库影响

- 仅新增一个索引，不新增表或列，不改写任何数据。
- ClickHouse 日志库不受影响，它走 `migrateClickHouseLogDB` 自己的建表语句，GORM 索引标签只作用于 SQLite、MySQL 和 PostgreSQL 的 `AutoMigrate`。
- 索引创建发生在 `InitResources()` 阶段，HTTP 端口尚未监听，表现为启动变慢而不是运行时阻塞。MySQL 5.7+ 为 online DDL 不阻塞读写；PostgreSQL 的普通 `CREATE INDEX` 会阻塞写入；SQLite 为全表锁。若在 PostgreSQL 上且 `logs` 表规模很大，应改为手动执行 `CREATE INDEX CONCURRENTLY` 后再升级。
- 索引本身会略微增加日志写入开销并占用磁盘（约每行 30–45 字节）。新索引的最左前缀就是 `user_id`，理论上使 `logs.user_id` 单列索引成为冗余，本次不删除。

### 测试与验证

- 重写 `service/consumption_export_test.go`：覆盖单遍扫描下按令牌分表的序号连续性、生成期间记录增减的报错、四种工作表布局、工作簿组装和临时文件清理（含新的行文件）。
- 原 `TestWriteWorksheetXMLRejectsUnexpectedStreamingRowCount` 保护的“导出期间数据变化”契约已由 `TestConsumptionExportDetailRejectsRecordsAddedAfterPlanning` 和 `TestConsumptionExportDetailFinishRejectsMissingRecords` 在新边界上直接覆盖。
- 当前本地终端的 PATH 中没有 Go，未能执行 `go build`、`go vet` 和后端回归测试，需要在有 Go 环境的机器上补跑。

## 2026-09-21：划转记录侧栏开关仅管理员可见

### 变更说明

- 个人设置「侧栏个性化」的“个人中心区域”里，“划转记录”模块开关只对管理员及超级管理员渲染。
- 原因：侧栏入口本身带 `requiredRole: ROLE.ADMIN`（`web/src/hooks/use-sidebar-data.ts`），普通用户永远看不到该入口，这个开关对他们是无效 UI。
- 普通用户点击“恢复默认”时保存的配置不含 `transfer` 键；`use-sidebar-config.ts` 中缺失的模块键按可见处理，所以日后升级为管理员不会被历史配置挡住入口。

### 实现位置

- `web/src/features/profile/components/sidebar-modules-card.tsx`：按 `auth.user.role >= ROLE.ADMIN` 条件拼接 `transfer` 模块项。

## 2026-09-21：取消用户自助删除账户

### 变更说明

- 个人设置「安全」卡片不再提供“删除账户”入口，安全操作只剩“修改密码”和“访问令牌”，卡片栅格从三列改为两列。
- 删除 `DELETE /api/user/self` 路由和 `controller.DeleteSelf`，用户绕过界面直接调用该接口会得到 404，无法再自助注销账号。
- 管理员删除用户的能力不变，仍然走 `DELETE /api/user/:id`（`controller.DeleteUser`），根用户仍受 `i18n.MsgUserCannotDeleteRootUser` 保护。

### 实现位置

- `web/src/features/profile/components/profile-security-card.tsx`：移除删除账户动作、对话框挂载和 `Trash2` 图标。
- `web/src/features/profile/components/dialogs/delete-account-dialog.tsx`：删除该组件。
- `web/src/features/profile/api.ts`、`web/src/features/profile/types.ts`：移除 `deleteUserAccount` 和 `DeleteAccountRequest`。
- `router/api-router.go`、`controller/user.go`：移除自助删除路由与处理函数。

### 遗留说明

- 各语言 locale 文件中“删除账户”相关的翻译键未删除，未被引用不影响运行；其中 `Delete Account` 等文案也可能被后续功能复用。
- 当前本地终端没有 Go 和前端依赖（`web/node_modules` 缺少 typescript/oxlint），未能执行 `go build` 和前端 typecheck/lint，需要在完整环境补跑。

## 2026-09-21：配额警告阈值按展示货币输入

### 变更说明

- 个人设置「通知」里的“配额警告阈值”原来直接显示和编辑原始额度单位（默认 `500000`，显示为 100000 之类的裸数字），现在改为按站点展示货币输入，例如 `USD` 下填 `0.2`。
- 标签后追加当前货币标识（`USD` / `CNY` / 自定义符号 / `Tokens`），占位符复用 `Enter amount in {{currency}}` 和 `Enter amount in tokens`。
- 提交给后端的 `quota_warning_threshold` 仍是原始额度单位，由 `parseQuotaFromDollars` 换算，`service/quota.go` 的比较逻辑和后端 `> 0` 校验不变。
- 站点设置为 Tokens 展示时，`quotaUnitsToDollars` 与 `parseQuotaFromDollars` 均为恒等变换，界面与改动前一致。

### 实现位置

- `web/src/features/profile/components/tabs/notification-tab.tsx`：新增 `formatThresholdAmount`，输入框改为展示货币金额的受控字符串，订阅 `useSystemConfigStore` 的 `config.currency` 以便货币配置加载后重新归一化。

### 遗留说明

- 系统设置里的全局默认值 `QuotaRemindThreshold`（`web/src/features/system-settings/integrations/monitoring-settings-section.tsx`）仍是原始额度单位，本次未改。
- locale 中的 `Enter threshold` 键不再被引用，未删除。

## 2026-09-21：新增企业微信群机器人通知

### 变更说明

- 个人设置「通知」的通知方式增加“企业微信”，与邮箱、Webhook、Bark、Gotify 并列，选中后填写群机器人 Webhook 地址即可。
- 采用群机器人 Webhook 方式（`https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...`），不涉及 corpid/corpsecret/agentid，也不需要维护 access_token。
- 消息体为文本消息 `{"msgtype":"text","text":{"content":"标题\n正文"}}`。企业微信文本消息不渲染 HTML，因此额度预警内容改用纯文本，充值链接以裸 URL 附在末尾（企业微信客户端会自动识别为链接）。
- 企业微信业务失败时仍返回 HTTP 200，发送逻辑会读取响应体的 `errcode`，非 0 时返回错误，避免把失败当成功。
- 与 Bark/Gotify 一致：Worker 模式走 `DoWorkerRequest`，非 Worker 模式先过 `ValidateSSRFProtectedFetchURL` 再用受 SSRF 保护的 HTTP 客户端发送。

### 实现位置

- `dto/user_settings.go`：新增 `NotifyTypeWeCom` 与 `WeComWebhookUrl` 字段（存在用户 `setting` JSON 中，不新增数据库列）。
- `controller/user.go`：`UpdateUserSettingRequest` 接收 `wecom_webhook_url`，校验非空、URL 格式和 http(s) 前缀，并在通知方式为企业微信时落库。
- `service/user_notify.go`：新增 `sendWeComNotify`，`NotifyUser` 增加企业微信分支。
- `service/quota.go`：额度预警和订阅额度预警各增加一个企业微信纯文本分支。
- `i18n/keys.go`、`i18n/locales/*.yaml`：新增 `setting.wecom_url_empty`、`setting.wecom_url_invalid`。
- `web/src/features/profile/constants.ts`、`types.ts`、`components/tabs/notification-tab.tsx`：通知方式选项、类型、表单字段和图标；选项栅格由 4 列改为 `sm:3 / lg:5`。
- `web/src/i18n/locales/*.json`：7 种语言新增 `WeCom`、`WeCom Bot Webhook URL` 和说明文案。

### 同步上游注意

- `dto/user_settings.go`、`controller/user.go` 的设置校验段、`service/user_notify.go`、`service/quota.go` 的通知分支都是上游文件，合并上游时如果上游自己新增通知方式，这几处需要人工合并。

### 待验证

- 本机没有 Go 和前端依赖，`go build`、`go vet` 和前端 typecheck/lint 未执行。
- 真实企业微信群机器人地址未做端到端发送验证，需要在部署环境用一个群机器人 key 实测一次。

## 2026-09-21：通知设置增加“发送测试通知”

### 变更说明

- 个人设置「通知」底部在“保存设置”旁边增加“发送测试通知”按钮。
- 测试用的是当前表单里的配置，不写库，因此可以先测通再保存；两条入口共用同一套校验，规则不会漂移。
- 新增接口 `POST /api/user/setting/test`，位于 `UserAuth` 自助路由组并挂 `CriticalRateLimit`（默认 20 次 / 20 分钟）。
- 测试通知走 `service.DispatchUserNotify`，跳过按通知类型的频率限制（默认 2 条 / 10 分钟，那是给额度预警这类自动通知用的），避免用户连点两次就被挡。滥用由路由上的 `CriticalRateLimit` 兜底。
- 邮件方式在通知邮箱和账户邮箱都为空时，发送逻辑本来会静默跳过，接口改为直接返回 `setting.test_no_email`，不会假装发送成功。
- 通知标题和正文通过后端 i18n 输出，跟随请求语言（en / zh-CN / zh-TW）。

### 实现位置

- `controller/user.go`：抽出 `validateUserSettingRequest` 和 `applyNotifySettings`，`UpdateUserSetting` 改为复用；新增 `TestUserSettingNotify`。
- `service/user_notify.go`：拆出 `DispatchUserNotify`（不限流的发送路径），`NotifyUser` = 频率限制 + 该函数。
- `dto/notify.go`：新增 `NotifyTypeTest`。
- `router/api-router.go`：注册 `POST /api/user/setting/test`。
- `i18n/keys.go`、`i18n/locales/*.yaml`：新增 `setting.test_sent`、`setting.test_failed`、`setting.test_no_email`、`setting.test_notify_title`、`setting.test_notify_content`。
- `web/src/features/profile/api.ts`、`components/tabs/notification-tab.tsx`：新增 `sendTestNotification` 与按钮、独立的 loading 状态。
- `web/src/i18n/locales/*.json`：7 种语言新增 `Send Test Notification`、`Test notification sent`、`Failed to send test notification`。

### 回归覆盖

- `controller/user_setting_notify_test.go`：覆盖 `applyNotifySettings` 只写入当前选中通知方式的字段（避免把上一个渠道的地址/密钥带进设置）、Gotify 优先级越界回落到 5、企业微信 Webhook 地址的空值/非法格式/非 http(s) 校验。

### 待验证

- 本机没有 Go，`go build`、`go vet` 和上面这个新测试都没跑过，需要在有 Go 环境的机器上补跑。
