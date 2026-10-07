# CLIProxyAPI 交接总览

最后更新：2026-10-07

## 1. 仓库关系

维护版后端：
https://github.com/Qing842/CLIProxyAPI

上游后端：
https://github.com/router-for-me/CLIProxyAPI

维护版管理中心：
https://github.com/Qing842/Cli-Proxy-API-Management-Center

上游管理中心：
https://github.com/router-for-me/Cli-Proxy-API-Management-Center

维护原则：持续吸收上游更新，但不得覆盖 Qing842 自定义功能和生产发布流程。

## 2. 生产部署基线

生产目录：

~~~
/opt/cliproxyapi
~~~

关键路径：

~~~
/opt/cliproxyapi/compose.yaml
/opt/cliproxyapi/config.yaml
/opt/cliproxyapi/auth
/opt/cliproxyapi/plugins
/opt/cliproxyapi/logs
/opt/cliproxyapi/data
~~~

截至 2026-10-07 的已验证生产镜像：

~~~
ghcr.io/qing842/cliproxyapi:sha-1d290e232cb3c9ec667962a928bed1f5287e93b7
~~~

对应多架构 manifest digest：

~~~
sha256:b86d22b93d6a59bb4bed86bb203ce99e13acfd53a9828ba46da362c3ed4bf2bc
~~~

生产关键配置：

~~~yaml
routing:
    strategy: "quota-drain"

management:
    panel-github-repository: "https://github.com/Qing842/Cli-Proxy-API-Management-Center"
~~~

不要把 management.secret-key、OAuth token、API key 或 auth 文件内容写入仓库、Issue、PR 或公开日志。

## 3. 当前自定义功能

### Quota Drain

用途：在多个付费订阅账号之间优先消耗更容易在重置前浪费的长期额度。

核心实现：

~~~
internal/quotadrain/
sdk/cliproxy/auth/capacity.go
sdk/cliproxy/auth/quota_pin.go
sdk/cliproxy/auth/selector.go
~~~

规则和限制见 QUOTA_DRAIN_CN.md。

### GHCR 多架构发布

工作流：

~~~
.github/workflows/qing-ghcr-image.yml
~~~

main 更新后并行原生构建 AMD64 和 ARM64，再生成多架构 manifest。

### 自定义管理中心

生产后端通过 panel-github-repository 拉取 Qing842 管理中心最新 Release 中的 management.html。

## 4. 日常维护原则

- 生产镜像优先固定到 sha-<完整 commit>，不要长期依赖 latest。
- 改生产配置前先备份 compose.yaml 和 config.yaml。
- 上游同步必须走 sync/upstream-* 临时分支。
- 禁止用 reset --hard upstream/main 覆盖维护版 main。
- 禁止 force push main。
- 上游同步完成后必须验证 Quota Drain、管理中心发布和 GHCR 构建工作流。
- LICENSE 保留上游 MIT 许可和版权声明。

## 5. 交接检查清单

接手者至少需要确认：

1. 能访问两个 Qing842 Fork。
2. 能查看 GitHub Actions 和 GHCR Package。
3. 知道生产 Compose 与 config 路径。
4. 知道生产使用固定 SHA 镜像。
5. 知道 Quota Drain 的排序、刷新和回退规则。
6. 知道如何从 upstream/main 安全同步更新。
7. 知道如何回滚到上一镜像和上一份配置备份。
8. 知道管理中心必须发布 management.html。

详细操作分别见 DEPLOYMENT_CN.md、RELEASE_CN.md 和 UPSTREAM_SYNC_CN.md。
