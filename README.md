# Qing842 CLIProxyAPI

> **中文说明**  
> 这是 Qing842 维护的 CLIProxyAPI Fork，用于持续同步上游能力，同时保留 Quota Drain、GHCR 多架构构建、生产部署与交接流程等维护版能力。  
> 完整中文总览请看 [README_CN.md](README_CN.md)，生产交接请看 [docs/HANDOVER_CN.md](docs/HANDOVER_CN.md)。

This repository is the maintained Qing842 fork of CLIProxyAPI.

The fork tracks upstream CLIProxyAPI while preserving local production features and release automation. The primary handover documentation is maintained in Chinese because the production environment and operating procedures are maintained by Qing842.

## Fork-specific features

- Quota Drain routing for Antigravity, Codex, and xAI OAuth credentials.
- Native multi-architecture GHCR builds for linux/amd64 and linux/arm64.
- Integration with the Qing842 fork of CLI Proxy API Management Center.
- Production handover, release, rollback, and upstream-sync documentation.

## Documentation

- Chinese overview: README_CN.md
- Handover: docs/HANDOVER_CN.md
- Architecture: docs/ARCHITECTURE_CN.md
- Deployment: docs/DEPLOYMENT_CN.md
- Quota Drain: docs/QUOTA_DRAIN_CN.md
- Release process: docs/RELEASE_CN.md
- Upstream synchronization: docs/UPSTREAM_SYNC_CN.md
- Troubleshooting: docs/TROUBLESHOOTING_CN.md

## Repositories

Maintained fork:
https://github.com/Qing842/CLIProxyAPI

Upstream:
https://github.com/router-for-me/CLIProxyAPI

Management Center fork:
https://github.com/Qing842/Cli-Proxy-API-Management-Center

## License and attribution

This project remains under the upstream MIT license. Keep LICENSE and its copyright notices intact.
