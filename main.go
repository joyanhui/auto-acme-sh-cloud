package main

import (
	"flag"
	"log"
)

var (
	FLAG_IS_FORCE_UPDATE_CERT = flag.Bool("f", false, "是否强制acme.sh更新证书")
	FLAG_ACMESH_SSL_SERVER    = flag.String("s", "zerossl", "证书服务器 默认zerossl 可以指定其他 acme.sh支持的证书服务器")
	FLAG_RUN_MODE             = flag.String("r", "prepare", "运行模式: `prepare`|`p`安装基本工具(acme.sh 7zip rclone),`get`|`g` 获取证书,`deploy`|`d` 推送到cdn,`clean`|`c` 删除过期或重复的证书")
	FLAG_PATH_ACMESH_EXEC     = flag.String("a", "", "acme.sh执行文件路径 如果不指定 默认 为 acme.sh(本地) 或 /home/runner/.acme.sh/acme.sh(github actions) ")
	FLAG_PATCH_CONFIG         = flag.String("c", "", "配置文件 路径  如果不指定 默认 为 /home/yh/myworkspace/acme.sh/config.toml(本地) 或 /home/runner/work/acme.sh/acme.sh/config.toml(github actions) ")
	FLAG_PATCH_DOT_ACMESH     = flag.String("dotacme", "", ".acme.sh路径 也是工作路径 如果不指定 默认 为 /home/yh/.acme.sh(本地) 或 /home/runner/.acme.sh(github actions) ")

	FLAG_PATCH_SSL_DIR     = flag.String("ssl", "", "证书路径 如果不指定 默认 为 /home/yh/.acme.sh/ssl(本地) 或 /home/runner/work/acme.sh/acme.sh/ssl(github actions) ")
	FLAG_PATCH_RCLONE_CONF = flag.String("rclone_conf", "", "RCLONE 配置文件路径 如果不指定 默认 为 /home/yh/.rclone.conf(本地) 或 /home/runner/.rclone.conf(github actions) ")
)

func main() {
	flag.Parse() //解析启动参数
	init_path()  // 初始一些路径相关的
	CONFIG_TOML = getConfig()
	switch *FLAG_RUN_MODE {
	case "p":
		fallthrough
	case "prepare":
		log.Println("=== 准备环境 start ==================== ")
		log.Println("=== 从rclone copy dot_acmesh 到本地 ")
		prepare_rclone()
		rclone_copyto(CONFIG_TOML.RcloneAC_NodeAndPath+"/dot_acmesh", PATCH_DOT_ACMESH)
		log.Println("=== 安装基本工具 ")
		prepare_acmesh_7z_dnsapi()
		log.Println("=== 准备环境 end ==================== ")
	case "g":
		fallthrough
	case "get":
		get_cert()
		log.Println("=== 备份ssl start ==================== ")
		rclone_copyto(PATCH_SSL_DIR, CONFIG_TOML.RcloneAC_NodeAndPath+"/cert_file")
		log.Println("=== 用rclone copy dot_acmesh 到远程 ")
		rclone_copyto(PATCH_DOT_ACMESH, CONFIG_TOML.RcloneAC_NodeAndPath+"/dot_acmesh")
		log.Println("=== 备份ssl end ==================== ")
	case "d":
		fallthrough
	case "deploy":
		upload_and_deploy_to_cloud()
	case "c":
	case "clean":
		cleanCloudCert()
	default:
	}

}
func cleanCloudCert() {
	// 遍历配置文件中的ak
	for cloudname, v := range CONFIG_TOML.CloudAK {
		log.Println("=== 清理", cloudname, "的证书 start ============================================================ ")
		//遍历cloud对应的证书
		for Account_tag, ak := range v {
			log.Println("=== 清理", cloudname, "的证书 Account_tag", Account_tag, " start ==================== ")
			switch cloudname {
			case "tencent":
				tencent_delete_expired_or_repeated(ak)
			case "ali":
				ali_delete_expired_or_repeated(ak)
			}
			log.Println("=== 清理", cloudname, "的证书 Account_tag", Account_tag, " end ==================== ")
		}
		log.Println("=== 清理", cloudname, "的证书 end ============================================================ ")

	}

}
