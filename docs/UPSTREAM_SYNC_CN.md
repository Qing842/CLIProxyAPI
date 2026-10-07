# 上游同步流程

## 原则

Qing842 main 是维护版主线，不允许用 upstream/main 强行覆盖。

禁止：

~~~bash
git reset --hard upstream/main
git push --force
~~~

## 后端

首次配置：

~~~bash
git remote add upstream https://github.com/router-for-me/CLIProxyAPI.git
~~~

每次同步：

~~~bash
git fetch upstream
git checkout main
git pull origin main
git checkout -b sync/upstream-YYYYMMDD
git merge upstream/main
~~~

解决冲突后运行：

~~~bash
gofmt -w .
go build -o cli-proxy-api ./cmd/server
go test ./...
go build -o test-output ./cmd/server && rm test-output
~~~

然后推送同步分支并通过 PR 合并回 Qing842/main。

## 冲突重点

同步时优先检查：

~~~
internal/quotadrain/
sdk/cliproxy/auth/capacity.go
sdk/cliproxy/auth/quota_pin.go
sdk/cliproxy/auth/selector.go
sdk/cliproxy/service*.go
config.example.yaml
.github/workflows/qing-ghcr-image.yml
README*
docs/
AGENTS.md
~~~

上游对 auth selection、quota、provider billing、Management Asset updater 有改动时，需要重新评估 Quota Drain 兼容性，不能机械选 ours 或 theirs。

## 管理中心

上游：

~~~
https://github.com/router-for-me/Cli-Proxy-API-Management-Center
~~~

同样使用 sync/upstream-YYYYMMDD 分支合并 upstream/main，完成冲突处理后运行：

~~~bash
bun install --frozen-lockfile
bun run verify
~~~

特别检查 RoutingStrategy、路由策略 UI、本地化文案、管理中心 Release workflow。

## 合并后

两个仓库同步后都应观察各自 main CI。后端还要确认 GHCR 新镜像；管理中心还要确认新的 management.html Release。
