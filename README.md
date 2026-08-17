# test-wxcloudrun

微信云托管 Go 服务：接同环境 MySQL，带 Notes CRUD 和可点的前端页。

## 看效果

部署后打开服务公网域名根路径，页面上可以直接新增 / 编辑 / 删除笔记。数据在 MySQL `testdb.notes`。

## 数据库环境变量

同环境 MySQL 一般会自动注入 `MYSQL_ADDRESS`、`MYSQL_USERNAME`、`MYSQL_PASSWORD`。若页面提示密码未配置，到 **服务设置 → 环境变量** 补上：

| 变量 | 说明 |
| --- | --- |
| `MYSQL_ADDRESS` | 内网地址，例如 `10.35.108.30:3306` |
| `MYSQL_USERNAME` | 账号，一般是 `root` |
| `MYSQL_PASSWORD` | 控制台里的数据库密码，不要写进代码仓库 |
| `MYSQL_DATABASE` | 可选，默认 `testdb` |

改环境变量后要重新发布或重启服务才生效。库有「10 分钟自动暂停」，第一次连可能要等几秒。

## 接口

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| GET | `/` | CRUD 前端页 |
| GET | `/health` | 存活检查，不因数据库暂停而失败 |
| GET | `/api/db` | 数据库连接状态 |
| GET | `/api/notes` | 列表 |
| POST | `/api/notes` | 新增 `{"text":"..."}` |
| GET | `/api/notes/{id}` | 单条 |
| PUT | `/api/notes/{id}` | 更新 `{"text":"..."}` |
| DELETE | `/api/notes/{id}` | 删除 |
| GET | `/api/ping` | pong |
| GET | `/api/whoami` | 微信注入头 |
| POST | `/api/echo` | 回显 JSON |

## 本地

本机连不到云托管内网 IP。本地只验证页面和接口形状时，可以起一个本地 MySQL，再：

```bash
export MYSQL_ADDRESS=127.0.0.1:3306
export MYSQL_USERNAME=root
export MYSQL_PASSWORD=本地密码
PORT=8080 go run .
```

## 部署

控制台手动上传本仓库文件夹，或绑定 GitHub `main` 后重新发布。
