
# 自动acme.sh 证书申请与安装

自动申请证书 自动部署到云厂商 自动 使用rclon储存证书到网盘或者对象储存

> https://github.com/joyanhui/auto-acme-sh-cloud 仓库不保留commit记录 开发调试在 私有仓库

## 特性
- [x] 通过环境变量指定 命令行参数 以及配置文件
- [x] 支持 github actions，也可以在本地运行
- [x] 并发申请证书
- [x] 可以持久化储存证书 防止频繁申请被证书提供商拒绝 同时支持强制申请证书
- [x] 上传证书到阿里云
- [x] 上传证书到腾讯云
- [x] 部署证书到 阿里云 cdn
- [x] 部署证书到 腾讯云 cdn
- [x] 使用rclone 命令 复制 证书文件到到 s3储存
- [ ] 使用7zip打包加密发送到别处
- [ ] 推送到其他github仓库


## 开发和测试环境
- [x] nixos 24.11
- [x] ubuntu last(github action)
- [x] 持久化储存 ali oss 理论上 rclone支持的 多数可以

## 外部依赖
- [x] acme.sh  以及它需要的 curl/wget等
- [ ] 7z
- [ ] ssh
- [x] rclone
- [x] cat

## 使用方法
### 命令行
```sh
# 安装依赖 其他系统请自行更换命令
apt-get install -y golang rclone p7zip-full curl wget 
# go 依赖
sh go_get.sh
# 参数帮助 
go run *.go -h
Usage of /tmp/go-build728691659/b001/exe/cloud_ali:
  -a string
        acme.sh执行文件路径 如果不指定 默认 为 acme.sh(本地) 或 /home/runner/.acme.sh/acme.sh(github actions) 
  -c string
        配置文件 路径  如果不指定 默认 为 /home/yh/myworkspace/acme.sh/config.toml(本地) 或 /home/runner/work/acme.sh/acme.sh/config.toml(github actions) 
  -dotacme string
        .acme.sh路径 也是工作路径 如果不指定 默认 为 /home/yh/.acme.sh(本地) 或 /home/runner/.acme.sh(github actions) 
  -f    是否强制acme.sh更新证书
  -r prepare
        运行模式: prepare|`p`安装基本工具(acme.sh 7zip rclone),`get`|`g` 获取证书,`deploy`|`d` 推送到cdn,`clean`|`c` 删除过期或重复的证书 (default "prepare")
  -rclone_conf string
        RCLONE 配置文件路径 如果不指定 默认 为 /home/yh/.rclone.conf(本地) 或 /home/runner/.rclone.conf(github actions) 
  -s string
        证书服务器 默认zerossl 可以指定其他 acme.sh支持的证书服务器 (default "zerossl")
  -ssl string
        证书路径 如果不指定 默认 为 /home/yh/.acme.sh/ssl(本地) 或 /home/runner/work/acme.sh/acme.sh/ssl(github actions) 
# 准备安装环境
go run *.go -r prepare
# 获取证书
go run *.go -r get   # -f true
# 上传 到云厂商 服务器 并推送到 对应的服务，目前只有cdn
go run *.go -r deploy  
# 删除过期或重复的证书 
# 腾讯的会按照 导入时间 保留最新的同域名的那个  阿里的 会先按照同域名的过期时间 如果过期时间一样 会按照证书id 保留id编号最大的一个
 go run *.go -r cleanCloudCert

```
### github action
`.github/workflows/AutoACME.yml`
```yml
name: Auto ACME to Cloud 
on:
  workflow_dispatch:
    inputs:
      FORCE_UPDATE_CERT:
        type: boolean
        description: 是否强制更新没过期的证书
        default: false
        required: true
      SERV:
        type: choice
        description: 证书服务商(acme.sh支持的任意一种)
        required: true
        options:
          - zerossl
          - letsencrypt
          - buypass
          - ssl.com
          - google
        default: zerossl
  schedule:
    - cron: "7 1 25 * *"
  watch:
    types: [started]
env:
    INGITHUB: yes
    TZ: Asia/Shanghai
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - name: Checkout
      uses: actions/checkout@v2
      with:
        ref: main
        github_token: ${{ secrets.GITHUB_TOKEN }}
    - name: 安装环境
      run: |
         echo "安装go 环境 rclone 7z"
         sudo apt-get install -y golang rclone p7zip-full
         echo "安装go 依赖"
         sh /home/runner/work/acme.sh/acme.sh/go_get.sh
    - name: 初始化
      run: |
          go run *.go -r prepare 
    - name:  获取证书
      run: |
          go run *.go -r get -s ${{ github.event.inputs.SERV }} -f ${{ github.event.inputs.FORCE_UPDATE_CERT }}
    - name:  上传 到云厂商
      run: |
          go run *.go -r deploy
    - name:  删除过期或重复的证书
      run: |
          go run *.go -r clean

```

## 配置文件说明
`config.toml`
```ini
EMAIL="leiyanhui@gmail.com"
# TODO 定时任务的通知 
WebHookUrl="https://www.pushplus.plus/send?token=XXXX&title=标题&content=内容&template=html"
# TODO ssl证书打包加密发生到别处
ZIPPASSWORD=""
# TODO 推送到其他仓库
MYGITHUBKEY=""
# rclone 配置文件中的节点和路径 用于持久化数据  rclone节点:Bucket/目录
RcloneCopyNodeAndPath="oss-qd:disk-lyh/acme.sh" 
# 证书的别名 在腾讯云和阿里云的的接口 都需要这个参数
SSL_MANAGE_ALIAS="acme.sh"
# 证书的用途 腾讯云需要这个参数
CertificateUse="cdn"
# 域名配置列表 dns_dp 的定义和acme.sh的定义一致 cloud 表示云厂商 tencent|ali Account_tag是账号的标识 用于区分多个AK
DOMAIN_TO_DO = [
    {domianName = "domian1.com", is_enable = false,dns_type = "dns_dp", deploy_to =[ {cloud = "tencent", Account_tag="company2",cdn_domains = ["dev.domian1.com","www.domian1.com"]}]},
    {domianName = "domian2.com", is_enable = true,dns_type = "dns_dp", deploy_to = [{cloud = "ali",Account_tag="joyanhui",  cdn_domains = ["domian2.com","www.domian2.com"]}]}
]


# 阿里云账号 个人
[cloud_AK.ali.joyanhui]
id="XXXX"
secret="XXXX"
# 阿里云账号 公司
[cloud_AK.ali.company]
id="XXXX"
secret="XXXX"
# 腾讯云账号 公司
[cloud_AK.tencent.company2]
id="XXXX"
secret="XXXX"
[DNSAPI]
DP_Id="XXX"
DP_Key="XXX"
CF_Token="XXX"
CF_Account_ID="XXX"


```
`rclone.conf` 参考 
```ini
[oss-qd]
type = s3
provider = Alibaba
access_key_id = XXXX
secret_access_key = XXXX
endpoint = oss-cn-XXXX.aliyuncs.com
acl = private
storage_class = STANDARD
bucket_acl = private
```
## 目录结构
```sh
tree
.
├── .github/workflows/AutoACME.yml
├── cloud_ali.go
├── cloud_tencent.go
├── config.go
├── config.toml
├── go_get.sh
├── go.mod
├── go.sum
├── main.go
├── rclone.conf
├── README.md
└── utils.go
```
