# xray-streisand-helper

在 Debian/Ubuntu VPS 上安装 Xray Reality，并把上游生成的 Clash YAML
转换成可由 Streisand 扫码导入的 VLESS 链接。

## 要求

- Debian 或 Ubuntu，systemd，Linux amd64/arm64
- root 权限和公网 IPv4
- 一个已将 A 记录解析到 VPS 公网 IPv4 的域名
- 使用 Cloudflare 时必须设为 **DNS Only**，不能开启代理
- TCP 443 未被其他程序占用

首版不支持 NAT VPS、仅 IPv6 VPS、非 systemd 系统或其他发行版。

## 一键安装

安装脚本固定 helper 版本，不跟随 `latest`：

```bash
curl -fsSL https://raw.githubusercontent.com/imzihuailin/xray-streisand-helper/main/install.sh -o /tmp/xsh-install.sh
sudo bash /tmp/xsh-install.sh
```

流程会下载并校验 helper，要求输入一次域名，然后下载并校验固定的
`xray-installer v0.1.10`。安装成功且 DNS、服务、端口和配置全部一致后，
终端会输出 VLESS 链接和二维码。用 Streisand 扫码导入，并实际连接验证。

再次运行 `setup` 默认复用现有 UUID 和 Reality 密钥：

```bash
sudo xray-streisand-helper setup
```

`--force` 会要求上游重新安装，仅在明确需要替换有效配置时使用：

```bash
sudo xray-streisand-helper setup --force
```

残缺安装、无法识别的 `xray` 程序或非 Xray 程序占用 443 时，工具会中止，
不会猜测或覆盖。

## 命令

```text
xray-streisand-helper setup [--force]  安装或复用有效配置
xray-streisand-helper show             验证默认安装并显示链接和终端二维码
xray-streisand-helper link [yaml]      stdout 仅输出 VLESS 链接
xray-streisand-helper serve [yaml]     启动临时本地二维码网页
xray-streisand-helper doctor           检查平台、文件、服务、443、DNS 和配置一致性
xray-streisand-helper --version
```

`show` 在 DNS A 记录不包含安装时探测到的公网 IPv4 时不会生成二维码。
修正 DNS 后重新执行 `sudo xray-streisand-helper show`。

## 临时网页

`serve` 只监听随机的 `127.0.0.1` 端口，使用随机令牌，禁止缓存，不加载第三方
资源，并在 10 分钟后自动关闭。它也响应网页停止按钮、`Ctrl+C` 和终止信号。

网页不是一键主流程。需要在电脑上的第二个终端建立 SSH 隧道，例如网页显示
VPS 地址 `127.0.0.1:41234` 时：

```bash
ssh -N -L 41234:127.0.0.1:41234 root@192.0.2.10
```

然后在本机浏览器打开工具输出的完整令牌 URL。

## 安全与上游

helper 独立调用 GPL-3.0 项目
[`manateelazycat/xray-installer`](https://github.com/manateelazycat/xray-installer)，
不复制或修改其源码。固定版本和提交为：

```text
v0.1.10
022b6ad3d6126fb0f5ffc1db68c035485f8f03a8
```

源码固化官方 amd64/arm64 Release 包的 SHA-256，并检查归档只包含预期二进制，
同时核对 `--version` 输出。

重要限制：该上游程序内部执行
`https://raw.githubusercontent.com/XTLS/Xray-install/main/install-release.sh`。
这是来自 `main` 分支的未固定脚本，helper 对上游 Release 包的哈希校验无法覆盖
这一层网络下载。运行前应理解并接受这个供应链边界。

配置和安装元数据包含凭据，必须保持仅 root 可读。不要把真实的 YAML、IP、
UUID、公钥、short ID 或 VLESS 链接提交到公开仓库或日志。

## 开发

```bash
go test -race ./...
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/xray-streisand-helper
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/xray-streisand-helper
```

GitHub Actions 对每次提交运行测试、vet 和两种架构构建。推送 `v*` 标签后会发布
静态二进制归档、源码标签及 `checksums.txt`。

自动测试不替代最终人工验收：发布前应在全新 Debian 和 Ubuntu VPS 上各完成一次
端到端安装，并用 Streisand 实际扫码导入和连接。

## License

自有代码采用 MIT。第三方信息见
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)。
