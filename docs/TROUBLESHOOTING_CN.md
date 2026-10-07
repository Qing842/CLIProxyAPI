# 故障排查

## Quota Drain 看起来没有生效

先确认配置：

~~~bash
sudo grep -nE '^(routing:|    strategy:)' /opt/cliproxyapi/config.yaml
~~~

应为 quota-drain。

随后检查：

- 账号是否为 Antigravity、Codex 或 xAI OAuth。
- 同类账号 priority 是否不同；高 priority 会先于额度紧迫度。
- 相关短周期窗口是否已耗尽。
- quota 数据是否抓取失败或已超过 10 分钟未刷新。
- 如果任一候选缺少有效长期数据，策略可能安全退回 round-robin。

需要更详细诊断时临时开启 debug 日志，完成后恢复，避免长期产生过多日志。

## 管理中心没有更新

确认：

~~~bash
sudo grep -nE '^(management:|    panel-github-repository:|    disable-auto-update-panel:)' /opt/cliproxyapi/config.yaml
~~~

panel-github-repository 应指向 Qing842 Fork。

确认最新管理中心 Release 中存在 management.html。后端启动时会主动执行一次面板更新检查，之后周期检查。

## GHCR 镜像拉取失败

先确认目标 tag 存在以及 Package 可访问。生产推荐使用 sha-<full-commit-sha>，避免 latest 在排障期间变化。

## GitHub Actions 构建异常慢

当前维护版 ARM64 已改为 ubuntu-24.04-arm 原生 Runner。若日志出现 QEMU 路径，说明运行的可能是旧 workflow 或旧 commit。

## 容器启动失败

执行：

~~~bash
sudo docker compose -f /opt/cliproxyapi/compose.yaml config -q
sudo docker logs --tail 200 cliproxyapi
~~~

优先检查 YAML、挂载文件权限、config 兼容性和目标镜像架构。

## 快速回滚

将 compose image 恢复为上一已验证 SHA，并按 DEPLOYMENT_CN.md 重建。不要删除 auth、plugins、logs 或 data。
