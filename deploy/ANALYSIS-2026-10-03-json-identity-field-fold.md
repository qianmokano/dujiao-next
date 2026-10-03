# 统一身份字段检查与 JSON Unicode 折叠一致

日期：2026-10-03，基线 fork main `f83b5c7`。两站账户面板统一的最终字段检查补充。

## 依据与修改

- 标准库 JSON 字段绑定使用 Unicode SimpleFold，ToLower 无法完整匹配；例如后台 `paſſword: null` 的混合请求可以漏过原始字段检查。
- 参考商城个人原始 JSON 检查、商城后台身份 guard、ginutil 请求参数工具三处模式，新增 `ginutil.HasJSONField` 并共用 `strings.EqualFold` 比较，不新增文件或数据库配置。
- 公共工具测试对照实际 JSON 解码，覆盖 null、ASCII、Unicode long s／Kelvin、重复字段、缺失字段及不同 Unicode 字符。个人和后台 HTTP 回归扩充相应输入，拒绝在业务修改之前发生。
- `.17` 已发布并预拉取但未切换生产；不移动既有标签。当前线上仍为 `.16`，测试账号通行证资料尚未修改，原有业务指纹保持一致。

## 验证与下一步

运行相关 HTTP／工具回归、全量 Go 测试、vet、服务端构建与真实 SQLite 集成测试。PR、合并 main 检查及新的未使用标签镜像全部通过后，先网关后商城完成完整备份、实际恢复、隔离验证和正式切换，随后执行两站真实资料修改与还原验收。
