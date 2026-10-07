# 生产部署与回滚

## 生产路径

~~~
/opt/cliproxyapi/compose.yaml
/opt/cliproxyapi/config.yaml
~~~

持久化目录包括 auth、plugins、logs、data。

## 部署前备份

~~~bash
sudo cp /opt/cliproxyapi/compose.yaml /opt/cliproxyapi/compose.yaml.bak-$(date +%Y%m%d-%H%M%S)
sudo cp /opt/cliproxyapi/config.yaml /opt/cliproxyapi/config.yaml.bak-$(date +%Y%m%d-%H%M%S)
~~~

## 推荐部署方式

生产使用固定 SHA 镜像：

~~~
ghcr.io/qing842/cliproxyapi:sha-<full-commit-sha>
~~~

不要把 latest 作为长期生产基线。

先拉取目标镜像：

~~~bash
sudo docker pull ghcr.io/qing842/cliproxyapi:sha-<full-commit-sha>
~~~

确认 compose 中 image 指向目标镜像，随后：

~~~bash
sudo docker compose -f /opt/cliproxyapi/compose.yaml config -q
sudo docker compose -f /opt/cliproxyapi/compose.yaml up -d --force-recreate
~~~

## 部署后检查

~~~bash
sudo docker ps --filter name=cliproxyapi
sudo docker inspect cliproxyapi --format 'Image={{.Config.Image}} Status={{.State.Status}}'
sudo docker logs --tail 100 cliproxyapi
~~~

启动日志应显示 VERSION、COMMIT 与预期提交一致。

同时检查：

~~~bash
sudo grep -nE '^(routing:|    strategy:|management:|    panel-github-repository:)' /opt/cliproxyapi/config.yaml
~~~

生产维护版应保持 quota-drain，并指向 Qing842 管理中心 Fork。

## 回滚

回滚优先使用上一次已验证的固定 SHA 镜像，不要回到未知 latest。

步骤：

1. 把 compose image 改回上一个 SHA。
2. 如配置也变更过，恢复对应时间点的 config.yaml.bak-*。
3. 执行 compose config -q。
4. 重建容器。
5. 检查日志和关键 API。

持久化 auth、plugins、logs、data 不应在普通镜像回滚中删除。
