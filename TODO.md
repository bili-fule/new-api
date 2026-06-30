# QQ 绑定功能实现清单

## 背景

已有外部 AstrBot QQ 机器人服务提供以下 API：
- `GET /api/plug/api/v1/bind/code?user_id=xxx` → `{"code":"200262","expire_seconds":300}`
- `GET /api/plug/api/v1/bind/query?user_id=xxx` → `{"code":"200262","bound":true,"user_id":"abc123","qq":"123456789"}`

AstrBot 地址：`http://192.168.10.128:6185`（管理员后台可配置）

new-api 需要调用这些 API 实现 QQ 绑定，并在 TokenAuth 中检查。

---

## 1. 后端：添加配置常量

**文件：`common/constants.go`**

在 `var QQRequiredEnabled = false` 旁边添加：
```go
var QQBotBaseURL = ""
```

---

## 2. 后端：注册配置到选项系统

### 2.1 `model/option.go` — `InitOptionMap()` 函数

添加：
```go
common.OptionMap["QQBotBaseURL"] = ""
```

### 2.2 `model/option.go` — `UpdateOption()` 的 switch 分支

在 `case "TurnstileSecretKey":` 后面添加：
```go
case "QQBotBaseURL":
    common.QQBotBaseURL = value
```

---

## 3. 后端：创建 QQ 绑定控制器

**新建文件：`controller/qq_bind.go`**

实现两个 handler：

### 3.1 `GetQqBindCode`

- 路由：`GET /api/user/self/bind/qq/code`
- 认证：需要 `middleware.UserAuth()`
- 行为：
  1. 从 session/token 获取当前用户 ID
  2. 调用 `common.QQBotBaseURL + "/api/plug/api/v1/bind/code?user_id=" + strconv.Itoa(userId)`
  3. 解析 AstrBot 返回的 JSON
  4. 返回 `{"code":"200262","expire_seconds":300}` 到前端

```go
func GetQqBindCode(c *gin.Context) {
    userId := c.GetInt("id")
    resp, err := http.Get(common.QQBotBaseURL + "/api/plug/api/v1/bind/code?user_id=" + strconv.Itoa(userId))
    // ... 错误处理
    var result struct {
        Code          string `json:"code"`
        ExpireSeconds int    `json:"expire_seconds"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    result,
    })
}
```

### 3.2 `ConfirmQqBind`

- 路由：`POST /api/user/self/bind/qq`
- 认证：需要 `middleware.UserAuth()`
- 请求体：`{"code":"200262"}`
- 行为：
  1. 从 session 获取用户 ID
  2. 用当前用户 ID 调用 AstrBot 的 `/api/plug/api/v1/bind/query?user_id=xxx`
  3. 检查 `bound` 是否为 `true`
  4. 如果是，把 `qq` 字段值存到当前用户的 `QQId` 字段
  5. 调用 `user.Update(false)` 保存
  6. 清除用户缓存
  7. 返回成功

注意：调用 AstrBot 时需要验证返回的 `user_id` 是否与当前登录用户一致（防止 A 用 B 的码）。

---

## 4. 后端：注册路由

**文件：`router/api-router.go`**

在 `selfRoute` 组（已有 `/api/user/self` 相关路由的组）中添加：

```go
selfRoute.GET("/bind/qq/code", middleware.CriticalRateLimit(), controller.GetQqBindCode)
selfRoute.POST("/bind/qq", middleware.CriticalRateLimit(), controller.ConfirmQqBind)
```

`selfRoute` 已有 `middleware.UserAuth()`，所以无需重复加。

---

## 5. 后端：TokenAuth 添加 QQ 绑定检查

**文件：`middleware/auth.go`** — `TokenAuth` 函数

在用户状态检查后面（`userEnabled` 检查之后，`userCache.WriteContext(c)` 之前）添加：

```go
if common.QQRequiredEnabled && userCache.QQId == "" {
    abortWithOpenAiMessage(c, http.StatusForbidden, common.TranslateMessage(c, "bind.qq_required"))
    return
}
```

确保 `common/constants.go` 中已声明：
```go
var QQRequiredEnabled = false
```

确保 `i18n/keys.go` 中有：
```go
const BindQQRequired = "bind.qq_required"
```

确保 `i18n/locales/en.yaml` 和 `zh-CN.yaml` / `zh-TW.yaml` 中有翻译：
```yaml
bind.qq_required: "Please bind your QQ account before using the API"
bind.qq_required: "请先绑定 QQ 账号后再使用 API"
```

---

## 6. 后端：User 模型已有字段

`model/user.go` 中已有 `QQId` 字段，缓存 `model/user_cache.go` 的 `UserBase` 中已有 `QQId`。无需重复创建。

`model/user.go` 中 `ClearBinding` 的 `bindingColumnMap` 已有 `"qq": "qq_id"`。

---

## 7. 前端：添加 API 调用

**文件：`web/default/src/features/profile/api.ts`**

添加两个函数：

```typescript
export async function getQqBindCode(): Promise<ApiResponse<{code: string; expire_seconds: number}>> {
  const res = await api.get('/api/user/self/bind/qq/code')
  return res.data
}

export async function confirmQqBind(code: string): Promise<ApiResponse> {
  const res = await api.post('/api/user/self/bind/qq', { code })
  return res.data
}
```

---

## 8. 前端：更新 QqBindDialog 组件

**文件：`web/default/src/features/profile/components/dialogs/qq-bind-dialog.tsx`**

需要重新实现（原有文件已在回退时删除）：

1. 打开弹窗时调用 `getQqBindCode()` 获取验证码
2. 显示验证码和说明文字
3. 用户点击"已发送，确认绑定"按钮时调用 `confirmQqBind(code)` 
4. 成功后调用 `onSuccess()` 刷新页面
5. 点按确认绑定前先轮询 `/api/plug/api/v1/bind/query`（直接调 AstrBot），但更简单的方案是让用户手动点击确认按钮

> 建议方案（参考 Telegram 绑定对话框的模式）：
> - 展示验证码 + 说明"请将此验证码发送给 QQ 机器人"
> - 一个"确认绑定"按钮
> - 点击后调 `confirmQqBind()`，new-api 去 AstrBot 查询
> - 成功/失败 toast 提示

---

## 9. 前端：翻译

**文件：`web/default/src/i18n/locales/en.json`**

添加：
```json
"QQ": "QQ",
"Bind QQ Account": "Bind QQ Account",
"Bind your QQ account to enable API access": "Bind your QQ account to enable API access",
"QQ account bound successfully": "QQ account bound successfully",
"Send this code to the QQ bot": "Send this code to the QQ bot",
"Confirm binding": "Confirm binding",
"Failed to generate code": "Failed to generate code",
"Failed to bind QQ": "Failed to bind QQ"
```

**文件：`web/default/src/i18n/locales/zh.json`** 同理添加中文翻译。

---

## 10. 前端：个人资料页添加 QQ 绑定项

**文件：`web/default/src/features/profile/components/tabs/account-bindings-tab.tsx`**

在 bindings 数组中添加 QQ 项（在 WeChat 后面）：

```typescript
{
  id: 'qq',
  label: t('QQ'),
  icon: MessageCircle,
  value: (profile as unknown as Record<string, unknown>).qq_id as string | undefined,
  isBound: Boolean((profile as unknown as Record<string, unknown>).qq_id),
  isEnabled: true,
  onBind: () => dialogs.open('qq'),
},
```

记得添加 `MessageCircle` 到 `lucide-react` 导入，以及 `QqBindDialog` 的导入和渲染。

---

## 架构总览

```
┌─────────────────────────────────────────────────────┐
│ 前端 (React)                                         │
│  QqBindDialog                                        │
│   ① open → GET /api/user/self/bind/qq/code          │
│   ② 用户发送「绑定200262」到 QQ 机器人              │
│   ③ 点确认 → POST /api/user/self/bind/qq {code}     │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│ new-api (Go/Gin)                                     │
│  GET /api/user/self/bind/qq/code                     │
│    → 调 AstrBot: GET /api/plug/api/v1/bind/code?user_id=N    │
│    → 转发结果给前端                                             │
│                                                      │
│  POST /api/user/self/bind/qq                         │
│    → 调 AstrBot: GET /api/plug/api/v1/bind/query?user_id=N    │
│    → bound=true 则存 qq_id → 返回成功                │
│                                                      │
│  TokenAuth 中间件                                    │
│    → if QQRequiredEnabled && qq_id=="" → 403        │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│ AstrBot (QQ 机器人)                                   │
│  192.168.10.128:6185                                 │
│  GET /api/plug/api/v1/bind/code → 生成验证码                  │
│  GET /api/plug/api/v1/bind/query → 返回绑定状态               │
│  用户在 QQ 里发「绑定200262」→ 记录绑定               │
└─────────────────────────────────────────────────────┘
```

> **注意**：`common.QQBindSecret` 和对应的配置处理代码已在回退时清理，不需要了。AstrBot 那边的 confirm 是 AstrBot 自己内部处理的，new-api 只需要调用 query 查询结果即可。
