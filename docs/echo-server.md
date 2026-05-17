# Echo Server

A built-in [httpbin.org](https://httpbin.org)-style HTTP echo / debug server.
Useful for:

- 本地联调客户端时无需自己搭一个 mock server
- 在 CI / 集成测试里启一个可控的 HTTP 反射端点
- 调试代理、负载均衡、Service Mesh 的请求转发链路
- 作为 `/debug` 子路由嵌入到现有应用，临时查看请求实际收到的样子

实现位于 `server/echo_server.go`，构建在生产级 `server.Server` 之上 —
自带 graceful shutdown、`/healthz`、`/readyz`、lifecycle hooks。

---

## 快速开始

### 方式一：独立服务

```go
package main

import "github.com/gookit/rux/v2/server"

func main() {
    s := server.NewEchoServer()
    s.Addr = ":18080"
    _ = s.Run()
}
```

或者直接跑示例：

```bash
go run ./_examples/echo-server
# 监听 127.0.0.1:18080
```

### 方式二：嵌入到现有 Router

只想在自己的应用里挂一个 `/debug` 子树？用 `MountEchoRoutes`：

```go
r := rux.New()
// ... your own routes ...

r.Group("/debug", func() {
    server.MountEchoRoutes(r)
})
```

完整示例：

```bash
go run ./_examples/echo-server -mode=embed
# 访问 http://127.0.0.1:18080/debug/anything
```

---

## 端点列表

| Method | Path                              | 说明                                                   |
| ------ | --------------------------------- | ------------------------------------------------------ |
| GET    | `/`                               | HTML 首页，列出所有端点                                |
| ANY    | `/anything`                       | 回显完整请求（method/url/headers/body/query/form/json）|
| ANY    | `/anything/*path`                 | 同上，忽略尾部路径                                     |
| GET    | `/get`                            | 方法限定，其它方法 → 405 (Allow: GET)                  |
| POST   | `/post`                           | 方法限定，其它方法 → 405 (Allow: POST)                 |
| PUT    | `/put`                            | 方法限定，其它方法 → 405 (Allow: PUT)                  |
| PATCH  | `/patch`                          | 方法限定，其它方法 → 405 (Allow: PATCH)                |
| DELETE | `/delete`                         | 方法限定，其它方法 → 405 (Allow: DELETE)               |
| GET    | `/headers`                        | 仅回显请求头                                           |
| GET    | `/ip`                             | 回显客户端 IP                                          |
| GET    | `/user-agent`                     | 回显 User-Agent                                        |
| ANY    | `/status/{code}`                  | 返回指定 HTTP 状态码（100~599，越界回退 200）          |
| GET    | `/delay/{seconds}`                | 睡眠 N 秒后回显，N 上限 10                             |
| GET    | `/redirect/{n}`                   | 302 跳转 N 次后落到 `/get`，N 上限 30                  |
| GET    | `/cookies`                        | 回显请求 Cookies                                       |
| GET    | `/cookies/set/{name}/{value}`     | 设置 Cookie 后 302 到 `/cookies`                       |
| GET    | `/basic-auth/{user}/{passwd}`     | Basic Auth 校验                                        |
| GET    | `/bytes/{n}`                      | 返回 N 字节随机数据，N 上限 100KB                      |
| GET    | `/uuid`                           | 生成 RFC 4122 v4 UUID                                  |
| GET    | `/download/{filename}`            | 按参数合成文件供下载                                   |
| POST   | `/upload`                         | 接收 multipart 上传并回显文件元数据                    |
| ANY    | `/*path`                          | 兜底：任何未匹配路径走回显                             |

`/healthz` 与 `/readyz` 来自 `server.Server`，不在 echo 路由表里。

---

## 用法示例

### 回显请求

```bash
curl -X POST http://localhost:18080/anything \
  -H 'Content-Type: application/json' \
  -d '{"hello":"world"}'
```

返回（节选）：

```json
{
  "method": "POST",
  "url": "/anything",
  "headers": { "Content-Type": "application/json" },
  "body": "{\"hello\":\"world\"}",
  "json": { "hello": "world" }
}
```

### 返回指定状态码

```bash
curl -i http://localhost:18080/status/418
# HTTP/1.1 418 I'm a teapot
```

### 故意延迟

```bash
curl http://localhost:18080/delay/2     # 2 秒后返回
curl http://localhost:18080/delay/-1    # 负数视为 0，立即返回
curl http://localhost:18080/delay/100   # 自动限到 10 秒
```

### 跳转链

```bash
curl -i http://localhost:18080/redirect/3
# 302 → /redirect/2 → /redirect/1 → /redirect/0 → /get
```

### Basic Auth

```bash
curl -i http://localhost:18080/basic-auth/alice/s3cret
# 401 + WWW-Authenticate

curl -u alice:s3cret http://localhost:18080/basic-auth/alice/s3cret
# 200 {"authenticated": true, "user": "alice"}
```

### 下载（自动合成）

Echo server 本身不持久化任何文件，`/download/{filename}` 一律按查询参数
**实时合成**内容。

| 参数        | 默认  | 说明                                                  |
| ----------- | ----- | ----------------------------------------------------- |
| `size=N`    | 1024  | 字节数，封顶 100KB                                    |
| `type=`     | `bin` | `bin` 随机字节 / `text` 重复 ASCII / `json` 元数据    |
| `inline=1`  | -     | 用 `Content-Disposition: inline` 替代 `attachment`    |

```bash
# 默认下载 1KB 随机字节
curl -OJ http://localhost:18080/download/blob.bin

# 2KB 可读文本
curl "http://localhost:18080/download/poem.txt?type=text&size=2048"

# JSON 元数据文件
curl "http://localhost:18080/download/meta.json?type=json"

# 内联展示而非触发下载
curl -i "http://localhost:18080/download/show.bin?inline=1" | head
```

### 上传（不落盘）

`POST /upload` 接收 `multipart/form-data`，每个文件**流式过 SHA-256**，
**不写盘**，返回每个文件的元数据和非文件 form 字段。

请求体上限 32 MB。

```bash
curl -F "file=@./README.md" \
     -F "tag=v1" \
     http://localhost:18080/upload
```

返回：

```json
{
  "files": [
    {
      "field": "file",
      "filename": "README.md",
      "size": 12345,
      "mime": "application/octet-stream",
      "sha256": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
    }
  ],
  "form": {
    "tag": ["v1"]
  }
}
```

### 方法限定与 405

`/get` `/post` `/put` `/patch` `/delete` 这 5 个端点是**方法限定**的 —
错误方法会得到 405，**不会**被兜底吞成 200：

```bash
curl -i http://localhost:18080/post                 # POST → 200 回显
curl -i -X GET http://localhost:18080/post          # GET  → 405, Allow: POST
```

这跟 httpbin 的语义一致，便于测试客户端正确处理 405。

### 兜底路由

未注册的**路径**（任何方法）会落到根 `/*path` 回显，方便测试客户端的容错：

```bash
curl http://localhost:18080/foo/bar/baz       # 200 回显
curl -X DELETE http://localhost:18080/random  # 200 回显
```

注意：兜底仅在路径未注册时生效。如上节所述，已注册的方法限定端点
（如 `/post`）即使方法不匹配，也由该端点自己返回 405，而不会回退到兜底。
具体路由按 rux 的优先级 `static > param > wildcard` 生效 —
比如 `/status/418` 仍走对应 handler，不会被吞。

---

## 嵌入到生产应用

```go
import (
    "github.com/gookit/rux/v2"
    "github.com/gookit/rux/v2/server"
)

s := server.New(false)

// 业务路由
s.GET("/api/v1/users", listUsers)
// ...

// 仅在开发 / 调试构建里挂载 echo
if buildIsDebug {
    s.Group("/debug", func() {
        server.MountEchoRoutes(s.Router)
    })
}

_ = s.Run()
```

注意：echo 路由是**完全公开**的（无鉴权），生产环境如要保留请自行加中间件，
比如把 `/debug` group 套一层 `handlers.HTTPBasicAuth` 或 IP allowlist。
