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

提交：`9811970a feat: add personal consumption exports`

固定版本镜像：`ghcr.io/rise-001/new-api-custom:sha-9811970`

### 功能说明

在个人板块增加“消费清单”入口，用户可以按时间范围和 API Key 筛选自己的消费记录，并异步导出 Excel 文件。

- 导出内容包含请求时间、API Key、模型、输入 Token、输出 Token、消费金额等消费明细
- Excel 按 API Key 分工作表，并附带每日汇总、退款、总计和模型统计
- 创建导出任务后，前端在任务处于等待或运行状态时每 10 秒刷新一次进度
- 支持查看历史任务、取消任务、删除任务和下载已完成文件
- 只能查看、操作和下载当前用户自己的导出任务
- 单次导出时间范围最长 366 天，最多导出 50,000 条消费记录
- 导出文件有效期为 1 小时，过期后无法下载；后台每分钟清理一次过期文件

### 实现结构

- `web/src/features/consumption-exports/`
  - 消费清单页面、创建导出对话框、任务列表、接口调用和数据类型
- `web/src/routes/_authenticated/profile/index.tsx`
  - 增加 `/profile?view=consumption` 个人消费清单视图
- `web/src/hooks/use-sidebar-data.ts`
  - 在个人板块增加“消费清单”入口
- `web/src/features/profile/components/sidebar-modules-card.tsx`
  - 支持在个人侧边栏模块设置中控制消费清单入口
- `controller/consumption_export.go`
  - 导出任务的创建、查询、取消、删除和下载接口
- `service/consumption_export.go`
  - 消费记录查询、异步任务执行、进度更新、文件过期和清理逻辑
- `service/xlsx_writer.go`
  - Excel 明细、汇总和统计工作表生成
- `model/consumption_export.go`
  - 导出文件存储、读取和清理的数据访问逻辑
- `model/system_task.go`
  - 扩展系统任务以支持用户归属、消费导出任务和取消状态
- `web/src/i18n/locales/*.json`
  - 已同步英文、简体中文、繁体中文、法语、日语、俄语和越南语文案

### API

- `GET /api/consumption-export/`：查询当前用户的导出任务
- `POST /api/consumption-export/`：创建导出任务
- `GET /api/consumption-export/:task_id/download`：下载有效期内的导出文件
- `POST /api/consumption-export/:task_id/cancel`：取消等待中或运行中的任务
- `DELETE /api/consumption-export/:task_id`：删除任务及其导出文件

以上接口均要求用户登录，并在后端校验任务归属。

### 数据库影响

应用启动后通过现有 GORM 自动迁移执行以下结构变更，兼容 SQLite、MySQL 和 PostgreSQL：

- 在 `system_tasks` 表增加可空且带索引的 `user_id` 字段，用于标识用户级系统任务
- 新增 `consumption_export_files` 表，使用数据库原生二进制字段保存 Excel 内容和过期时间

迁移不会重写或删除现有用户、消费日志和系统任务记录。过期导出文件会从表中删除，但 SQLite、MySQL 或 PostgreSQL 的物理数据库文件不一定立即缩小。导出文件保存在主数据库是为了保证多实例部署时任意实例都能下载，因此部署前应备份数据库，并关注导出高峰期的数据库空间占用。

### 测试与验证

已验证：

- 消费导出 model 测试通过
- Excel 生成和消费导出 service 测试通过
- 后端全包编译检查通过
- 前端 TypeScript 类型检查通过
- 涉及文件 lint 和格式检查通过
- 7 个前端语言文件同步检查无缺失、无多余键和未翻译项
- Rsbuild 生产构建检查通过
- `git diff --check` 通过

全量 `go test ./service` 仍存在仓库原有的 channel-affinity usage-cache 计数器隔离测试失败，与本次消费导出功能无关。完整版权检查仍会报告 `web/src/features/channels/lib/channel-field-update.ts` 的既有空行格式问题，本次未修改该文件。

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
- 当前本地终端的 PATH 中没有 Go 和 Bun，未能执行后端回归测试和前端 lint
