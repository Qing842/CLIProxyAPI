# 架构说明

## 总体结构

~~~
AI Client
   |
   v
CLIProxyAPI
   |
   +-- OAuth / file credentials
   |     +-- Antigravity
   |     +-- Codex
   |     +-- xAI
   |
   +-- Routing
   |     +-- round-robin
   |     +-- weighted-round-robin
   |     +-- fill-first
   |     +-- quota-drain   <- Qing842 自定义
   |
   +-- Management API / Control Panel
             |
             +-- Qing842 Management Center Release
~~~

## Quota Drain 数据流

~~~
Provider quota endpoint
        |
        v
internal/quotadrain Collector
        |
        v
Auth.Capacity runtime snapshot
        |
        v
QuotaDrainSelector
        |
        +-- priority 分层
        +-- 短周期额度门控
        +-- 长周期额度紧迫度排序
        +-- pin 固定账号
        +-- 数据缺失时 round-robin 回退
~~~

CapacityState 只存在于运行时，不持久化到 auth 文件。

## 管理中心更新

后端管理面板 updater 根据 management.panel-github-repository 请求该仓库的 latest Release，查找 management.html，并按 digest 与本地文件比较。启动时会立即检查一次，之后周期检查。

## 发布架构

后端 main：
GitHub Actions -> AMD64 原生构建 + ARM64 原生构建 -> GHCR 架构镜像 -> multi-arch manifest。

管理中心 main：
GitHub Actions -> Bun build -> dist/index.html 重命名为 management.html -> GitHub Release。

## 生产挂载

生产 Compose 将 config.yaml、auth、plugins、logs、data 挂载到容器。因此重建容器不会删除这些持久化内容，但执行部署前仍应备份配置文件。
