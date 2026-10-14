# 本机验证记录

环境：2026-10-14，macOS ARM64，Go 1.27.1。Godwit 宿主固定在 #68 的 `0686d91f248302f085e799d13e990c2805348856`。

- `go test -race ./...`、`go vet ./...` 通过。
- `scripts/package.py` 成功生成 c-shared 动态库、手动安装 ZIP、SHA-256 和目录发布片段。
- `scripts/smoke.py` 通过真实 Godwit `ServeNative` 加载，验证 describe 与 manifest 一致、空配置、空白段、非法语言、未知方法，以及错误 SHA-256 / ABI 版本拒绝。
- `--live` 使用公开示例 `Hello, world!`，返回 `你好，世界！`，来源 `en`；没有读取用户邮件或凭据。
- 语言映射、UTF-16 分批、emoji、空白段、顺序恢复、限流与响应不完整均有离线测试。
- 独立代码审查完成，修复了 Windows 读取 manifest 和解码中文响应时的默认编码问题。

首轮测量的库约 10.88 MiB，后续构建的准确大小及哈希以产物内 `registry-release.json` 为准。一次真实 Godwit 测试宿主进程完成 describe、configure、空白翻译的峰值 RSS 为 32,342,016 字节（约 30.84 MiB），使用 `/usr/bin/time -l` 测量。该测量不含联网翻译、Godwit 主应用或长时间运行，不能当作增量内存或泄漏检查。

macOS Intel CI 曾在真实 Go 宿主加载时触发 `bad sweepgen in refill`。Go 的 Darwin AMD64 链接器固定使用 GS+0x30 保存 goroutine 指针，两份运行时直接复用宿主线程会读到对方的指针。ABI 初始化改为纯 C，该平台的 Go 回调改由独立 C 线程执行，不更改 API。宿主测试并发发起 40 个本地调用，覆盖此边界。

跨平台结果见 GitHub Actions；工作流在五个平台构建并加载实际动态库。未提供 Windows ARM64 产物，未验证签名、公证后的桌面应用加载或长期稳态内存。服务使用非官方 DeepL 接口，可用性不保证。
