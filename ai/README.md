# Dubbo Admin AI

Dubbo Admin AI 是面向 Dubbo Admin 场景的智能运维服务模块，用于通过 AI 对话辅助排障、分析与治理决策。

项目 wiki: https://stringl1l1l1l.github.io/dubbo-admin-ai-wiki/

## JWT 访问校验与会话归属

`component/server/server.yaml` 中的两组配置共同生效：

- `auth` 控制可选的 JWT 访问校验。启用时，需要配置 Admin 的 `jwks_url`、
  `issuer` 和 `audience`，并在 Admin 侧启用 token 签发。AI API 请求必须携带
  有效的 Bearer token；`/health` 和 CORS 预检无需 token。
- `authentication` 控制会话归属使用的服务端身份。当前为 `static` 模式，
  会话仍归属于配置中的 `tenant_id` 和 `user_id`，不会根据 JWT 的 `sub`
  自动切换用户。JWT 访问校验本身不提供多用户会话隔离。

关闭 `auth.enabled` 只关闭 JWT 校验，会话归属检查仍然生效。
静态身份模式默认仅监听 `127.0.0.1`；非回环监听仍需显式设置
`authentication.allow_insecure_static_auth: true`，仅用于开发环境。
旧会话的归属迁移仍由 `authentication.adopt_legacy_sessions` 显式控制，默认关闭。

浏览器跨域访问时，在 `cors_origins` 中列出可信的前端来源（默认不允许跨域）。
允许的来源可以读取 `WWW-Authenticate` 响应头，供前端在 Bearer 401 后刷新
token 并重试一次。
