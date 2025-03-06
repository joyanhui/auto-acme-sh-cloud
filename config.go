package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

var (
	PATH_ACMESH_EXEC  = "" //替代 FLAG_PATH_ACMESH_EXEC 在代码中使用
	PATCH_CONFIG      = "" //替代 FLAG_PATCH_CONFIG 在代码中使用
	PATCH_DOT_ACMESH  = "" //替代 FLAG_PATCH_DOT_ACMESH 在代码中使用
	PATCH_SSL_DIR     = "" //替代 FLAG_SSL_FILE 在代码中使用
	PATCH_RCLONE_CONF = "" //替代 FLAG_RCLONE_CONF 在代码中使用

)

var CONFIG_TOML configToml

type (
	configToml struct {
		EMAIL                string                         `toml:"EMAIL"`
		WebHookUrl           string                         `toml:"WebHookUrl"`
		ZIPPASSWORD          string                         `toml:"ZIPPASSWORD"`
		RcloneAC_NodeAndPath string                         `toml:"RcloneCopyNodeAndPath"`
		CloudAK              map[string]map[string]ak_cloud `toml:"cloud_AK"`
		SSL_MANAGE_ALIAS     string                         `toml:"SSL_MANAGE_ALIAS"`
		CertificateUse       string                         `toml:"CertificateUse"`
		DNSAPI               map[string]string              `toml:"DNSAPI"`
		DOMAIN_TO_DO         []Domains                      `toml:"DOMAIN_TO_DO"`
	}
	ak_cloud struct {
		Id     string `toml:"id"`
		Secret string `toml:"secret"`
	}
	Domains struct {
		DomianName string      `toml:"domianName"`
		ISENABLE   bool        `toml:"is_enable"`
		DnsType    string      `toml:"dns_type"`
		DeployTo   []Deploy_To `toml:"deploy_to"`
	}
	Deploy_To struct {
		Account_tag string   `toml:"Account_tag"`
		Cloud       string   `toml:"cloud"`
		CdnDomains  []string `toml:"cdn_domains"`
	}
)

// 从toml中获取配置
func getConfig() configToml {
	var config configToml
	_, err := toml.DecodeFile(PATCH_CONFIG, &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return config
}

// 初始化一些路径
func init_path() {

	if check_local_or_githubactions() {
		if *FLAG_PATCH_CONFIG == "" {
			PATCH_CONFIG = "/home/runner/work/acme.sh/acme.sh/config.toml"
		} else {
			PATCH_CONFIG = *FLAG_PATCH_CONFIG
		}

		if *FLAG_PATCH_DOT_ACMESH == "" {
			PATCH_DOT_ACMESH = "/home/runner/.acme.sh"
		} else {
			PATCH_DOT_ACMESH = *FLAG_PATCH_DOT_ACMESH
		}
		if *FLAG_PATCH_SSL_DIR == "" {
			PATCH_SSL_DIR = "/home/runner/.acme.sh/ssl"
		} else {
			PATCH_SSL_DIR = *FLAG_PATCH_SSL_DIR
		}

		if *FLAG_PATCH_RCLONE_CONF == "" {
			PATCH_RCLONE_CONF = "/home/runner/work/acme.sh/acme.sh/rclone.conf"
		} else {
			PATCH_RCLONE_CONF = *FLAG_PATCH_RCLONE_CONF
		}
		if *FLAG_PATH_ACMESH_EXEC == "" {
			PATH_ACMESH_EXEC = "/home/runner/.acme.sh/acme.sh"
		} else {
			PATH_ACMESH_EXEC = *FLAG_PATH_ACMESH_EXEC
		}

		fmt.Println("在github actions")
	} else {
		if *FLAG_PATCH_CONFIG == "" {
			PATCH_CONFIG = "/home/yh/myworkspace/acme.sh/config.toml"
		} else {
			PATCH_CONFIG = *FLAG_PATCH_CONFIG
		}
		if *FLAG_PATCH_DOT_ACMESH == "" {
			PATCH_DOT_ACMESH = "/home/yh/.acme.sh"
		} else {
			PATCH_DOT_ACMESH = *FLAG_PATCH_DOT_ACMESH
		}
		if *FLAG_PATCH_SSL_DIR == "" {
			PATCH_SSL_DIR = "/home/yh/.acme.sh/ssl"
		} else {
			PATCH_SSL_DIR = *FLAG_PATCH_SSL_DIR
		}
		if *FLAG_PATCH_RCLONE_CONF == "" {
			PATCH_RCLONE_CONF = "/home/yh/.config/rclone/rclone.conf"
		} else {
			PATCH_RCLONE_CONF = *FLAG_PATCH_RCLONE_CONF
		}

		if *FLAG_PATH_ACMESH_EXEC == "" {
			PATH_ACMESH_EXEC = "acme.sh" // 默认认为已经Alis过了
		} else {
			PATH_ACMESH_EXEC = *FLAG_PATH_ACMESH_EXEC
		}
		fmt.Println("在本地")
	}
}

func check_local_or_githubactions() bool {
	//检查是否在本地或者在github actions
	//检查过环境变量 INGITHUB 是否存在 如果存在 那么就是在 github actions
	if _, exists := os.LookupEnv("INGITHUB"); exists {
		return true
	} else {
		return false
	}
}
