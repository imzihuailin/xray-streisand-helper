# xray-streisand-helper

利用 xray-installer 在 Debian/Ubuntu VPS 上安装 Xray Reality，并把由 xray-installer (https://github.com/manateelazycat/xray-installer) 生成的 proxy.yaml
转换成可由 Streisand 扫码导入的 VLESS 链接。

## 要求

- 装有 Debian 或 Ubuntu 的 VPS 
- root 权限和公网 IPv4
- TCP 443 未被其他程序占用
- 详见教程： https://manateelazycat.github.io/2026/04/09/best-proxy/

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/imzihuailin/xray-streisand-helper/main/install.sh -o /tmp/xsh-install.sh
sudo bash /tmp/xsh-install.sh
```

流程会下载并校验，安装成功且 DNS、服务、端口和配置全部一致后，
终端会输出 VLESS 链接和二维码。用 Streisand 扫码导入。

如果之前已经通过 xray-installer 获得过 proxy.yaml 或者再次运行 `setup` 就会默认复用现有 UUID 和 Reality 密钥：

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
xray-streisand-helper doctor           检查平台、文件、服务、443、DNS 和配置一致性
xray-streisand-helper --version
```

`show` 在 DNS A 记录不包含安装时探测到的公网 IPv4 时不会生成二维码。
修正 DNS 后重新执行 `sudo xray-streisand-helper show`。

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


## License

[MIT](https://github.com/imzihuailin/xray-streisand-helper/blob/main/LICENSE)

## 致谢
[xray-installer](https://github.com/manateelazycat/xray-installer)

[go-qrcode](https://github.com/skip2/go-qrcode)

详见[致谢文档](https://github.com/imzihuailin/xray-streisand-helper/blob/main/THIRD_PARTY_NOTICES.md)
