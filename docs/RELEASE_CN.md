# 发布流程

## 后端 GHCR

工作流：

~~~
.github/workflows/qing-ghcr-image.yml
~~~

触发条件：push 到 main。

构建：

- AMD64：ubuntu-latest
- ARM64：ubuntu-24.04-arm
- 不使用 QEMU 构建 ARM64
- 两个架构成功后再生成 multi-arch manifest

主要标签：

~~~
ghcr.io/qing842/cliproxyapi:latest
ghcr.io/qing842/cliproxyapi:sha-<full-commit-sha>
~~~

另有架构临时标签 latest-amd64、latest-arm64 和对应 SHA 架构标签。

生产部署优先固定 sha-<full-commit-sha>。

## 管理中心

管理中心由独立仓库发布：

https://github.com/Qing842/Cli-Proxy-API-Management-Center

后端 production config 通过 management.panel-github-repository 指向该仓库。

后端 updater 会从 latest Release 中寻找 management.html。

## 发布验收

后端 main 更新后至少确认：

1. qing-ghcr-image 成功。
2. build-amd64 成功。
3. build-arm64 成功。
4. publish-manifest 成功。
5. 目标 sha 镜像可以被生产主机 pull。
6. 部署后日志中的 Commit 与目标 SHA 一致。
7. Quota Drain 的相关 provider 至少做一次真实调用验证。
