# IAST 开启后 migration panic 修复

## 现象

`data/conf/iast/iast.conf` 中 `IAST_ENABLED=true` 后重建 api/consumer，启动报错：

```
panic: new with database instance failed : failed to open source,
file:///tmp/moresec-.../store/pg/migration": open .: no such file or directory
```

## 原因

`backend/store/pg/pg.go` 原先用 `runtime.Caller` 推算源码旁的 `migration/` 目录，再以 `file://` 加载 SQL。普通版路径是编译时的 `/src/store/pg/migration`（镜像里曾拷贝过去）；雳鉴插桩后 Caller 被改写到 `/tmp/moresec-...`，该目录不存在。

## 修复

migration SQL 改为 `embed.FS` 打进二进制（`go:embed migration/*.sql` + `migrate/source/iofs`），不再依赖磁盘源码路径。Dockerfile 中对 `/src/store/pg/migration` 的拷贝已删除。

106 验证：开启插桩后进程为 `/app/panda-wiki-api.iast`，migration 通过，管理端/站点 HTTP 200。

## 现场

加载新的 api/consumer 镜像后 `force-recreate` 即可；app/nginx 及其他中间件无需更新。
