package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func rclone_copyto(src string, dst string) {
	cmd := exec.Command("rclone", "copyto", src, dst, "--config", PATCH_RCLONE_CONF)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("combined run rclon copyto out:\n%s\n", string(out))
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	fmt.Printf("combined run rclon copyto  \n%s\n", string(out))

}

/*
prepare_run_env 通过判断 命令是否存在 然后 安装   如果是本地测试 最好手动安装 尤其是nixos
安装命令 默认只支持 ubuntu on github action
*/

func prepare_acmesh_7z_dnsapi() {

	//检查acme.sh 是否安装 也就是 acme.sh 命令是否存在

	cmd := exec.Command(PATH_ACMESH_EXEC, "-v")
	out, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.Command("sh", "-c", "curl https://get.acme.sh | sh")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("combined get acme.sh out:\n%s\n", string(out))
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}
		fmt.Printf("combined get acme.sh out:\n%s\n", string(out))
	} else {
		fmt.Printf("acme.sh 已安装,跳过安装...out:\n%s\n", string(out))
	}

	//检查 7z 命令

	cmd = exec.Command("7z")
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Println("7z 未安装,正在安装... sudo apt-get install -y p7zip-full")
		cmd := exec.Command("sudo", "apt-get", "install", "-y", "p7zip-full")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("combined get 7z out:\n%s\n", string(out))
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}
		fmt.Printf("combined get 7z out:\n%s\n", string(out))
	} else {
		// 只打印前 N 行
		outStr := string(out)
		lines := []string{}
		for _, line := range strings.Split(outStr, "\n") {
			if len(lines) < 3 {
				lines = append(lines, line)
			}
		}
		fmt.Printf("7z 已安装,跳过安装...out:\n%s\n", strings.Join(lines, "\n"))
	}

	//把 DNSAPI 写入 acme.sh/account.conf          echo "$DNSAPI" >> /home/runner/.acme.sh/account.conf
	//循环 TOML_CONFIG.DNSAPI 然后打印key和value
	exec.Command("mkdir", "-p", PATCH_DOT_ACMESH).Run() // 创建 .acme.sh 目录
	exec.Command("mkdir", "-p", PATCH_SSL_DIR).Run()    // 创建 ssl 目录
	log.Println("=== 删除account.conf ")
	os.RemoveAll(PATCH_DOT_ACMESH + "/account.conf") //需要删除这个文件 因为后面会重新写

	//acme.sh --register-account -m my@example.com
	log.Println("注册证书")
	cmd = exec.Command(PATH_ACMESH_EXEC, "--register-account", "-m", CONFIG_TOML.EMAIL)
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("combined run acme.sh register out:\n%s\n", string(out))
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	fmt.Printf("combined run acme.sh register out:\n%s\n", string(out))

	log.Println("写入 .acme.sh/account.conf")
	file, err := os.OpenFile(PATCH_DOT_ACMESH+"/account.conf", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("无法打开文件: %v", err)
	}
	defer file.Close()
	for key, value := range CONFIG_TOML.DNSAPI {
		// 把key 和 value 写入 acme.sh/account.conf
		line := "export " + key + "=\"" + value + "\"\n"
		//log.Println(line)
		if _, err := file.WriteString(line); err != nil {
			log.Fatalf("写入文件时出错: %v", err)
		}
	}
	//打印 PATCH_DOT_ACMESH+"/account.conf" 的内容
	cmd = exec.Command("cat", PATCH_DOT_ACMESH+"/account.conf")
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("combined run cat out:\n%s\n", string(out))
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	fmt.Printf("combined run cat out:\n%s\n", string(out))
}

func prepare_rclone() {
	//检查 rclone 命令
	log.Println("检查 or 安装 rclone")
	cmd := exec.Command("rclone", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("rclone 未安装,正在安装... sudo apt-get install -y rclone")
		cmd := exec.Command("sudo", "apt-get", "install", "-y", "rclone")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("combined get rclone out:\n%s\n", string(out))
			log.Fatalf("cmd.Run() failed with %s\n", err)
		}
		fmt.Printf("combined get rclone out:\n%s\n", string(out))
	} else {
		fmt.Printf("rclone 已安装,跳过安装...out:\n%s\n", string(out))
	}
}

func upload_and_deploy_to_cloud() {
	for _, v := range CONFIG_TOML.DOMAIN_TO_DO {
		if v.ISENABLE {
			for _, d := range v.DeployTo {
				switch d.Cloud {
				case "tencent":
					log.Println("=== 腾讯云上传并部署 start ====================", d)
					err := upload_to_ssl_and_deploy_to_cdn_tencent(v, d)
					if err != nil {
						log.Println("=== 腾讯云上传并部署 X 失败 ====================", d, err)
					}
					log.Println("=== 腾讯云上传并部署 end ====================", d)
				case "ali":
					log.Println("=== 阿里云上传并部署 start ====================", d)
					err := upload_to_ssl_and_deploy_to_cdn_ali(v, d)
					if err != nil {
						log.Println("=== 阿里云上传并部署 X 失败 ====================", d, err)
					}
					log.Println("=== 阿里云上传并部署 end ====================", d)

				}

			}

		}
	}

}

/*
获取证书
*/
func get_cert() {
	var wg sync.WaitGroup
	var successful_list []string
	var failed_list []string
	//循环 DOMAIN_TO_DO
	for _, v := range CONFIG_TOML.DOMAIN_TO_DO {
		if v.ISENABLE {
			wg.Add(1)
			go func() {
				log.Println("=== 获取证书 正在后台申请", v.DomianName)

				var cmd *exec.Cmd
				var out []byte
				var err error
				cmd = exec.Command(PATH_ACMESH_EXEC, "--issue", "--dns", v.DnsType, "-d", v.DomianName, "-d", "*."+v.DomianName, "--server", *FLAG_ACMESH_SSL_SERVER)
				if *FLAG_IS_FORCE_UPDATE_CERT {
					cmd = exec.Command(PATH_ACMESH_EXEC, "--issue", "--dns", v.DnsType, "-d", v.DomianName, "-d", "*."+v.DomianName, "--server", *FLAG_ACMESH_SSL_SERVER, "--force")
				}
				out, err = cmd.CombinedOutput()
				log.Println("=== 获取证书 start ====================", v.DomianName)
				if err != nil {
					fmt.Printf("=== 获取证书失败:%s \n%s\n", v.DomianName, string(out))
				} else {
					fmt.Printf("=== 获取证书成功:%s \n%s\n", v.DomianName, string(out))
				}
				log.Println("=== 获取证书 end ====================", v.DomianName)
				// 导出证书
				log.Println("=== 导出证书 start ====================", v.DomianName)
				cmd = exec.Command(PATH_ACMESH_EXEC, "--installcert", "-d", v.DomianName, "--key-file", PATCH_SSL_DIR+"/"+v.DomianName+".key", "--fullchain-file", PATCH_SSL_DIR+"/"+v.DomianName+".cer")
				if *FLAG_IS_FORCE_UPDATE_CERT {
					cmd = exec.Command(PATH_ACMESH_EXEC, "--installcert", "-d", v.DomianName, "--key-file", PATCH_SSL_DIR+"/"+v.DomianName+".key", "--fullchain-file", PATCH_SSL_DIR+"/"+v.DomianName+".cer", "--force")
				}
				out, err = cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("=== 导出域名失败 :%s \n%s\n", v.DomianName, string(out))
					failed_list = append(failed_list, v.DomianName)
				} else {
					successful_list = append(successful_list, v.DomianName)
					fmt.Printf("=== 导出证书成功:%s \n%s\n", v.DomianName, string(out))
				}
				log.Println("=== 导出证书 end ====================", v.DomianName)
				wg.Done()
			}()
		}
	}
	wg.Wait()
	log.Println("=== 申请和导出证书成功的域名 list  start ====================")
	log.Println("===========================================")
	log.Println("===========================================")
	for _, v := range successful_list {
		log.Println(v)
	}
	log.Println("===========================================")
	log.Println("===========================================")
	log.Println("=== 申请和导出证书成功的域名 list end ====================")

}
