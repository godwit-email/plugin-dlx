# DLX plugin for Godwit

为 Godwit 提供邮件与选中文本的多语言翻译，安装后即可使用，无需额外配置。

这是一个独立的社区插件，使用 [DLX](https://github.com/OwO-Network/DLX) 的翻译能力。插件直接调用翻译包，不启动 HTTP 服务，无需部署服务器或填写账号。

需要支持 [Godwit #68](https://github.com/godwit-email/godwit/pull/68) 原生插件 API 的版本。ABI 固定为 v1，接口来源提交 `0686d91f248302f085e799d13e990c2805348856`。DLX 依赖固定在 `0f2b50dd69fea3012d95a8d0e7a4ae8dca9ff5b2`，不会在运行时下载代码。

## 安装与使用

进入 Godwit 的扩展页面，安装并启用 DLX，然后在邮件或选区翻译中选择它。正式的一键安装需要维护者发布二进制，并将其 URL、大小、SHA-256 和 manifest 加入 Godwit 的签名插件目录；本仓库不会自行加入可信名单。

本地开发时，运行下面的构建命令，将 `dist/<平台>/` 作为插件目录导入 Godwit。目录包含 `plugin.json` 和当前平台的 `plugin.dylib` / `plugin.so` / `plugin.dll`。未进入可信目录的原生插件会触发 Godwit 的授权提示。

没有插件设置项。代理沿用宿主传入的 `HTTPS_PROXY` / `HTTP_PROXY` / `NO_PROXY` 环境变量。

## 构建与验证

需要 Go 1.27.1、C 编译器和 Python 3；Windows 使用 MinGW-w64 GCC。

```sh
go test -race ./...
go vet ./...
python3 scripts/package.py
cd tools/host && go mod download && cd ../..
python3 scripts/smoke.py dist/darwin-arm64/plugin.dylib
# 可选：发送一条公开示例到真实服务，不会读取邮件或凭据
python3 scripts/smoke.py dist/darwin-arm64/plugin.dylib --live
```

将 smoke 命令的路径换成当前平台的库。测试宿主直接使用 #68 的真实 `ServeNative`，检查描述、配置、翻译契约及 SHA-256 拒绝；不会将整个 Godwit 放进插件依赖。Windows 的竞态检查不在 CI 中运行。

CI 在 macOS ARM64 / AMD64、Linux ARM64 / AMD64、Windows AMD64 原生 runner 上测试、构建、加载。Windows ARM64 暂未提供产物。Linux 构建使用 Ubuntu 24.04，其 glibc 基线不保证兼容更旧发行版。

修改 `plugin.json` 的版本后推送相同的 `v<版本>` 标签，CI 会创建**草稿 release**：包含裸动态库、用于手动安装的 ZIP、校验文件和 `registry-release.json`，供维护者检查、发布及提交插件目录。ZIP 包含 manifest、固定名称的库和完整依赖许可证。

## 行为与边界

- `describe`、`configure`、`translate` 严格使用 ABI v1 的 JSON envelope；未知方法或非法请求返回 `invalid_request`。
- BCP 47 标签映射到 DLX 支持的语言；`zh-CN` / `zh-SG` → 简体，`zh-TW` / `zh-HK` / `zh-MO` → 繁体。英语默认美式，葡萄牙语默认巴西。
- 每批最多 1500 个 UTF-16 单位，emoji 计两个。长段优先在句子或单词边界切开，顺序发送，再恢复原段落数量和空白段。切句会损失跨句上下文。
- 遇到限流、网络错误或不完整响应，整次翻译失败，不返回部分翻译；不自动重试。每次调用超过 30 秒后不再开始新批次，给 Godwit 的 60 秒宿主超时留余量。长邮件可能需要选区翻译。
- 自动识别时，只有单个非空段且所有批次识别一致才返回 `source`；多个段返回空字符串，避免 Godwit 根据首段语言漏翻混合语言邮件。明确传入来源语言时返回该语言。
- 共享库支持并发调用，没有可变插件配置。请求最多 1 MiB，响应最多 4 MiB；响应由 C 分配，交给宿主释放。
- macOS Intel 的 Go 宿主与 Go 动态库共享线程本地存储槽，因此初始化只使用 C，翻译调用在独立 C 线程进入插件运行时，保持 ABI 不变。其他平台直接调用。

## 隐私与服务风险

翻译内容会发送至 DLX 使用的第三方翻译服务。DLX 还会访问相关网站初始化 cookies，并访问 Apple 的公开 App Store 接口查询客户端版本。插件不提供自建中转、不记录翻译正文。

原生动态库在 Godwit 的子进程中运行，具有当前用户权限；子进程不是权限沙箱。可信名单和哈希授权由 Godwit 管理。DLX 使用非公开接口，可能限流、临时封禁或随上游变更失效。插件是社区扩展，服务可用性取决于上游。

## 许可

插件代码及 Godwit ABI 头文件：Apache-2.0。DLX：MIT，许可原文见 `THIRD_PARTY_NOTICES`。每个平台安装目录附带所链接 Go 模块的许可证 `DEPENDENCY_LICENSES.txt`。
