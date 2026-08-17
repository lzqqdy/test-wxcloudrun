// 部署成功后，把 env 和服务名换成你的，粘到小程序任意页面里试。
// 基础库需要 >= 2.13.1

async function pingCloudRun() {
  wx.cloud.init()

  const env = '你的云托管环境ID'
  const service = 'golang' // 控制台服务名，当前是 golang

  const ping = await wx.cloud.callContainer({
    config: { env },
    path: '/api/ping',
    method: 'GET',
    header: { 'X-WX-SERVICE': service },
  })
  console.log('ping', ping.data)

  const who = await wx.cloud.callContainer({
    config: { env },
    path: '/api/whoami',
    method: 'GET',
    header: { 'X-WX-SERVICE': service },
  })
  console.log('whoami', who.data)

  const echo = await wx.cloud.callContainer({
    config: { env },
    path: '/api/echo',
    method: 'POST',
    header: {
      'X-WX-SERVICE': service,
      'content-type': 'application/json',
    },
    data: { hello: 'chuquwan' },
  })
  console.log('echo', echo.data)
}

module.exports = { pingCloudRun }
