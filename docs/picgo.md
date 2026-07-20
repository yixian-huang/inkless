# 用 PicGo 对接 Inkless 媒体库

把 [PicGo](https://github.com/Molunerfinn/PicGo) 配成上传到 **本站 `/admin/media/upload`**，图片进入 CMS 媒体库，返回可写进文章的 URL。

**推荐鉴权**：长期 **API Key**（`ink_…`），见下文。也可用登录 JWT，但会过期，不适合桌面客户端常驻配置。

---

## 1. 创建 API Key

1. 用有 `media:create` 权限的账号登录后台。  
2. 打开 **设置 → API Key**（`/admin/api-keys`）→ **新建 Key**，复制明文（仅一次）。  
3. 也可用 API（session JWT，不可用 `ink_` 自管 Key）：

```bash
# 先 JWT 登录拿 access token（字段名以 /auth/login 实际响应为准）
ACCESS=$(curl -sS -X POST 'https://YOUR_HOST/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"..."}' | jq -r '.accessToken // .token // .data.accessToken')

curl -sS -X POST 'https://YOUR_HOST/admin/api-keys' \
  -H "Authorization: Bearer $ACCESS" \
  -H 'Content-Type: application/json' \
  -d '{"name":"picgo","scopes":["media:create"]}' | jq .
```

响应示例：

```json
{
  "token": "ink_0123abcd…",
  "key": {
    "id": 1,
    "name": "picgo",
    "tokenPrefix": "ink_0123",
    "scopes": ["media:create"],
    "createdAt": "…"
  }
}
```

- **`token` 仅此一次**，请立即保存。  
- 默认 scope：`media:create`（只允许上传媒体，不能用 key 改文章等）。  
- 列表：`GET /admin/api-keys`；吊销：`DELETE /admin/api-keys/:id`。  
- **管理 Key 必须用登录 JWT**（`ink_…` 不能自管 Key，防泄露后横向扩权）。

---

## 2. 上传接口

| 项 | 值 |
|---|---|
| URL | `https://YOUR_HOST/admin/media/upload` |
| Method | `POST` |
| Body | multipart，字段名 **`file`** |
| Header | `Authorization: Bearer ink_…` |
| 成功 | HTTP **201**，JSON 含 **`url`** |

```bash
export INKLESS_KEY='ink_…'
curl -sS -X POST 'https://YOUR_HOST/admin/media/upload' \
  -H "Authorization: Bearer $INKLESS_KEY" \
  -F 'file=@./test.png' | jq .
# 期望 status 201 且 .url 可打开
```

---

## 3. PicGo（Web 图床插件）

| 配置项 | 填法 |
|---|---|
| API / URL | `https://YOUR_HOST/admin/media/upload` |
| 文件字段 | `file` |
| JSON Path | **`url`**（不是 `data.links.url`） |
| 自定义头 | `Authorization: Bearer ink_…` |

与 **img.li** 的差异：路径、JSON Path、鉴权模型均不同，**不要共用一份配置**。图床侧说明见 imgli 仓库 `docs/ops/picgo.md`。

---

## 4. 安全

- API Key 等同「该用户在 scope 内的机器身份」，泄露请立即吊销。  
- 上传仍校验用户 RBAC：用户本身须具备 `media:create`，key scope 再收窄一层。  
- 不要用无鉴权反代绕过 admin。  

预研背景：`docs/picgo-integration-preresearch.md`。
