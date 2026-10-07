# Qing842 CLIProxyAPI 维护版

这是 Qing842 维护的 CLIProxyAPI Fork。目标不是替代上游，而是在持续同步上游能力的同时，稳定保留生产环境需要的自定义功能、构建流程和交接资料。

## 当前自定义能力

1. Quota Drain（额度消耗优先）路由。
   - 支持 Antigravity、Codex、xAI OAuth 账号。
   - 同优先级账号中，优先消耗“更容易在重置前浪费”的长期额度。
   - 短周期额度负责可用性门控，长期额度负责排序。
   - 额度数据不可用时安全退回普通轮询。

2. 自定义 GHCR 镜像。
   - 镜像仓库：ghcr.io/qing842/cliproxyapi
   - AMD64 使用 ubuntu-latest 原生构建。
   - ARM64 使用 ubuntu-24.04-arm 原生构建。
   - main 每次更新后自动生成 latest 和 sha-<commit> 多架构镜像。

3. 自定义管理中心。
   - 仓库：https://github.com/Qing842/Cli-Proxy-API-Management-Center
   - 生产配置通过 management.panel-github-repository 指向该 Fork。
   - 管理中心包含 Quota Drain 选项并独立发布 management.html。

## 文档入口

- 交接总览：docs/HANDOVER_CN.md
- 架构：docs/ARCHITECTURE_CN.md
- 部署与回滚：docs/DEPLOYMENT_CN.md
- Quota Drain 规则：docs/QUOTA_DRAIN_CN.md
- 发布流程：docs/RELEASE_CN.md
- 同步上游：docs/UPSTREAM_SYNC_CN.md
- 故障排查：docs/TROUBLESHOOTING_CN.md

## 上游关系

维护版：
https://github.com/Qing842/CLIProxyAPI

上游：
https://github.com/router-for-me/CLIProxyAPI

同步上游时不要使用 reset --hard 或 force push 覆盖维护版 main。统一使用临时同步分支合并 upstream/main，完成冲突处理和测试后再通过 PR 回到 main。

## License

本项目沿用上游 MIT License。LICENSE 必须保留原版权声明和许可文本。
