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

### 测试

相关回归测试：

- `web/src/features/pricing/components/__tests__/model-performance-visibility.test.tsx`
- `web/src/features/system-settings/maintenance/__tests__/header-navigation-config.test.ts`

已验证：

- 9 条回归测试通过
- TypeScript 类型检查通过
- 涉及文件 lint 通过
- 全仓前端格式检查通过
- i18n 同步报告无缺失
- Rsbuild 生产构建通过

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
