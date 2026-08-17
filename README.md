# test-wxcloudrun

给微信云托管用的最小 Go HTTP 测试服务。无数据库，上传即可部署。

## 本地先跑

```bash
PORT=8080 go run .
curl http://127.0.0.1:8080/api/ping
curl http://127.0.0.1:8080/api/whoami
curl -X POST http://127.0.0.1:8080/api/echo \
  -H 'content-type: application/json' \
  -d '{"hello":"chuquwan"}'
```

## 部署到微信云托管

1. 打开 [微信云托管控制台](https://cloud.weixin.qq.com/cloudrun)，用小程序/公众号登录，没有环境就新建一个。
2. **服务列表 → 新建服务**。名称建议 `demo`（和小程序示例里的 `X-WX-SERVICE` 一致）。测试阶段打开「允许公网访问」。
3. 进入服务 → **部署发布 → 手动上传代码包 → 文件夹**，选中本仓库根目录（里面要有 `Dockerfile`、`main.go`、`go.mod`）。
4. 发布。构建大约 2～3 分钟。
5. 部署成功后点公网域名，应看到「微信云托管测试服务已启动」。再访问 `/api/ping` 应返回 JSON。

也可以把本仓库推到 GitHub，在发布里选「绑定代码仓库」，后续用流水线发版。

## 接口

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| GET | `/` | 浏览器看一眼服务是否活着 |
| GET | `/health` | 健康检查 |
| GET | `/api/ping` | 返回 pong |
| GET | `/api/whoami` | 打印微信注入头。公网 curl 时 openid 为空；小程序 `callContainer` 会带上 |
| POST | `/api/echo` | 原样返回 JSON body |
| GET/POST | `/api/notes` | 内存笔记，重启丢失，只用来确认读写 |

## 小程序调用

代码见 `examples/miniprogram.js`。核心是：

```js
wx.cloud.init()
const res = await wx.cloud.callContainer({
  config: { env: '云托管环境ID' },
  path: '/api/whoami',
  method: 'GET',
  header: { 'X-WX-SERVICE': 'demo' },
})
console.log(res.data)
```

`env` 在云托管控制台环境概览里，不是云开发环境 ID。`X-WX-SERVICE` 必须等于服务名。

## 说明

- 容器监听 `PORT`，默认 `80`，和云托管约定一致。
- `/api/notes` 存在进程内存里，缩到 0 或重新发布后数据会没。
- 这是连通性测试，不是业务模板。确认 ping / whoami 通了，再往里面加 Trip、Agent。
