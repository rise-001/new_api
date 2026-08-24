# 二次开发记录

## Gemini 图片模型无图不收费

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

本次修改已通过 `git diff --check`。提交时开发环境未安装 Go 工具链，因此未执行 `go test`；部署环境应执行：

```bash
go test ./relay/channel/gemini
```
