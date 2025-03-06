package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/openapi-util/service"
	util "github.com/alibabacloud-go/tea-utils/v2/service"

	"github.com/alibabacloud-go/tea/tea"
)

func upload_to_ssl_and_deploy_to_cdn_ali(v Domains, d Deploy_To) error {
	certId, certName, err := upload_to_ssl_ali(v, d)
	if err != nil {
		return err
	}
	err = deploy_to_cdn_ali(d, certId, certName)
	return err
}

func upload_to_ssl_ali(v Domains, d Deploy_To) (certId int64, certName string, err error) {
	// 从文件获取CertificatePublicKey
	CertificatePublicKey, err := os.ReadFile(PATCH_SSL_DIR + "/" + v.DomianName + ".cer")
	if err != nil {
		log.Println("腾讯云读取证书公钥文件失败,跳过:", err)
		return
	}
	// 处理CertificatePublicKey 把换行符号替换 字符串\n
	//CertificatePublicKey = []byte(strings.ReplaceAll(string(CertificatePublicKey), "\n", "\\n"))

	// 从文件获取CertificatePrivateKey
	CertificatePrivateKey, err := os.ReadFile(PATCH_SSL_DIR + "/" + v.DomianName + ".key")
	if err != nil {
		log.Println("读取证书私钥文件失败:", err)
		return
	}
	// 处理CertificatePrivateKey 把换行符号替换 字符串\n
	//CertificatePrivateKey = []byte(strings.ReplaceAll(string(CertificatePrivateKey), "\n", "\\n"))
	// 创建客户端
	config := &openapi.Config{
		AccessKeyId:     tea.String(CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Id),
		AccessKeySecret: tea.String(CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Secret),
	}
	config.Endpoint = tea.String("cas.aliyuncs.com")
	var client = &openapi.Client{}
	client, err = openapi.NewClient(config)
	if err != nil {
		return 0, "", err
	}

	params := &openapi.Params{
		Action:      tea.String("UploadUserCertificate"),
		Version:     tea.String("2020-04-07"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}
	//certName
	certName = CONFIG_TOML.SSL_MANAGE_ALIAS + "-" + v.DomianName + "-" + time.Now().Format("20060102150405") //NAME用Alias+当前时间组成
	queries := map[string]interface{}{
		"Name": tea.String(certName),
		"Cert": tea.String(string(CertificatePublicKey)),
		"Key":  tea.String(string(CertificatePrivateKey)),
	}
	log.Println("上传证书参数:", queries)
	// runtime options
	runtime := &util.RuntimeOptions{}
	request := &openapi.OpenApiRequest{
		Query: openapiutil.Query(queries),
	}

	// 发送请求
	resp, err := client.CallApi(params, request, runtime)
	if err != nil {
		log.Println("阿里云上传证书失败:", err)
		return
	}
	log.Println("阿里云上传证书的响应:", resp)

	respMap := resp["body"].(map[string]interface{})
	certIdNum := respMap["CertId"].(json.Number)
	// 将 json.Number 转换为 int64
	certId, err = certIdNum.Int64()
	if err != nil {
		log.Fatalf("转换 CertId 时出错: %v", err)
	}
	ResourceId := respMap["ResourceId"].(string)
	log.Printf("阿里云上传证书成功, 证书ID(ResourceId): %s, CertId: %v,Name: %s\n", ResourceId, respMap["CertId"], certName)

	return certId, certName, nil
}
func deploy_to_cdn_ali(d Deploy_To, certId int64, certName string) error {
	var err error
	//阿里云每次只能部署一个域名 所以需要循环
	for _, cdn_domain := range d.CdnDomains {
		log.Println("=== 阿里云部署 start ====================", cdn_domain)
		// 构建请求参数
		params := &openapi.Params{
			Action:      tea.String("SetCdnDomainSSLCertificate"),
			Version:     tea.String("2018-05-10"),
			Protocol:    tea.String("HTTPS"),
			Method:      tea.String("POST"),
			AuthType:    tea.String("AK"),
			Style:       tea.String("RPC"),
			Pathname:    tea.String("/"),
			ReqBodyType: tea.String("json"),
			BodyType:    tea.String("json"),
		}
		queries := map[string]interface{}{}
		queries["DomainName"] = tea.String(cdn_domain)
		queries["CertName"] = tea.String(certName)
		queries["CertId"] = tea.Int(int(certId))
		queries["CertType"] = tea.String("cas")
		queries["SSLProtocol"] = tea.String("on")
		log.Println("阿里云部署参数: DomainName", cdn_domain, "CertName", certName, "CertId", certId)
		// runtime options
		runtime := &util.RuntimeOptions{}
		request := &openapi.OpenApiRequest{
			Query: openapiutil.Query(queries),
		}
		// 创建客户端
		config := &openapi.Config{
			AccessKeyId:     tea.String(CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Id),
			AccessKeySecret: tea.String(CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Secret),
		}
		config.Endpoint = tea.String("cdn.aliyuncs.com")
		var client = &openapi.Client{}
		client, err = openapi.NewClient(config)
		if err != nil {
			return err
		}

		// 发送请求
		resp, err := client.CallApi(params, request, runtime)
		if err != nil {
			log.Printf("部署证书到域名 %s 失败: %v\n", cdn_domain, err)
			continue
		}

		log.Printf("阿里云成功部署证书到域名: %s\n", cdn_domain)
		respMap := resp["body"].(map[string]interface{})
		log.Printf("阿里云部署响应body: %v\n", respMap)

		log.Println("=== 阿里云部署 end ====================", cdn_domain)

	}
	return nil

}

func ali_delete_expired_or_repeated(ak ak_cloud) {

	// 查询证书列表
	params := &openapi.Params{
		// 接口名称
		Action: tea.String("ListUserCertificateOrder"),
		// 接口版本
		Version: tea.String("2020-04-07"),
		// 接口协议
		Protocol: tea.String("HTTPS"),
		// 接口 HTTP 方法
		Method:   tea.String("POST"),
		AuthType: tea.String("AK"),
		Style:    tea.String("RPC"),
		// 接口 PATH
		Pathname: tea.String("/"),
		// 接口请求体内容格式
		ReqBodyType: tea.String("json"),
		// 接口响应体内容格式
		BodyType: tea.String("json"),
	}
	config := &openapi.Config{
		AccessKeyId:     tea.String(ak.Id),
		AccessKeySecret: tea.String(ak.Secret),
	}
	config.Endpoint = tea.String("cas.aliyuncs.com")
	client := &openapi.Client{}
	var _err error
	client, _err = openapi.NewClient(config)
	if _err != nil {
		log.Printf("NewClient failed: %v\n", _err)
		return
	}
	queries := map[string]interface{}{}
	queries["OrderType"] = tea.String("UPLOAD")
	queries["ShowSize"] = tea.Int(1000)

	// runtime options
	runtime := &util.RuntimeOptions{}
	request := &openapi.OpenApiRequest{
		Query: openapiutil.Query(queries),
	}
	// 复制代码运行请自行打印 API 的返回值
	// 返回值实际为 Map 类型，可从 Map 中获得三类数据：响应体 body、响应头 headers、HTTP 返回的状态码 statusCode。
	resp, err := client.CallApi(params, request, runtime)
	if err != nil {
		log.Printf("CallApi failed: %v\n", err)
		return
	}
	respMap := resp["body"].(map[string]interface{})
	//log.Println("证书列表:", respMap)
	/*
		map[CertificateOrderList:[map[CertificateId:17510552 City: CommonName:shandongnuoya.com Country: EndDate:2025-06-01 Expired:false Fingerprint:0D7AC2FAC7164D782AA3C834CE7C7E9CDCD285F8 InstanceId:cas-upload-poq9k3 Issuer:ZeroSSL Name:joyanhui/acme.sh自动上传20250304035443 OrgName: Province: ResourceGroupId:rg-acfm4ldjch6wjbi Sans:shandongnuoya.com,*.shandongnuoya.com SerialNo:00a7bf34d810c4549f7405d4b6052dfb1e Sha2:C2131C179B290AA5CD17CE61B9EAC75465DE2E48E7F40F9EC1840188C895E287 StartDate:2025-03-03 Status:ISSUED Upload:true] map[CertificateId:17510728 City: CommonName:shandongnuoya.com Country: EndDate:2025-06-01 Expired:false Fingerprint:12073FC2D26FE2925D61A3655E24D51D8F481C44 InstanceId:cas-upload-vj4s5g Issuer:ZeroSSL Name:joyanhui/acme.sh自动上传20250304041417 OrgName: Province: ResourceGroupId:rg-acfm4ldjch6wjbi Sans:shandongnuoya.com,*.shandongnuoya.com SerialNo:1b2158d961e58f46f7ba63ec731dfc94 Sha2:7B07B294A0F778865CB37E24C86F90EB34DA072DBB4F7AF10C8091BC964E19B8 StartDate:2025-03-03 Status:ISSUED Upload:true] map[CertificateId:17510759 City: CommonName:shandongnuoya.com Country: EndDate:2025-06-01 Expired:false Fingerprint:07E83562AFFBE295E83C24CEEDB29A8D1873FC11 InstanceId:cas-upload-m8wpgu Issuer:ZeroSSL Name:joyanhui/acme.sh自动上传20250304044518 OrgName: Province: ResourceGroupId:rg-acfm4ldjch6wjbi Sans:shandongnuoya.com,*.shandongnuoya.com SerialNo:4dcc56aad10930b5422b4c44820b8c2a Sha2:CF35EA5E7483255216433EA71085019C0EF17ACC53B6501D5E3573876685A04B StartDate:2025-03-03 Status:ISSUED Upload:true]] CurrentPage:1 RequestId:C208FF18-AA54-5541-AD38-F820BD75E497 ShowSize:1000 TotalCount:3]
	*/
	//获取里面的 CertificateOrderList
	certificateOrderList := respMap["CertificateOrderList"].([]interface{})
	var certList []Ali_Cert_list_one
	for _, v := range certificateOrderList {
		/*
			2025/03/06 11:35:35 map[CertificateId:17510552 City: CommonName:shandongnuoya.com Country: EndDate:2025-06-01 Expired:false Fingerprint:0D7AC2FAC7164D782AA3C834CE7C7E9CDCD285F8 InstanceId:cas-upload-poq9k3 Issuer:ZeroSSL Name:joyanhui/acme.sh自动上传20250304035443 OrgName: Province: ResourceGroupId:rg-acfm4ldjch6wjbi Sans:shandongnuoya.com,*.shandongnuoya.com SerialNo:00a7bf34d810c4549f7405d4b6052dfb1e Sha2:C2131C179B290AA5CD17CE61B9EAC75465DE2E48E7F40F9EC1840188C895E287 StartDate:2025-03-03 Status:ISSUED Upload:true]
		*/
		//log.Println(v)
		//解析 v 并添加到 certList
		certMap := v.(map[string]interface{}) // 将 v 转换为 map
		var certId int64
		certId, err = certMap["CertificateId"].(json.Number).Int64()
		if err != nil {
			// Handle the error appropriately
			log.Printf("转换 CertificateId 时出错: %v\n", err)
			continue
		}
		cert := Ali_Cert_list_one{
			CommonName:    certMap["CommonName"].(string),
			StartDate:     certMap["StartDate"].(string),
			CertificateId: certId,
			Expired:       certMap["Expired"].(bool),
		}
		certList = append(certList, cert) // 添加到 certList
	}
	// 查询要过期的证书 =========================================== start
	expiredCertIds := []int64{}
	for _, cert := range certList {
		if cert.Expired {
			expiredCertIds = append(expiredCertIds, cert.CertificateId)
		}
	}
	log.Println("准备删除的 过期的 证书id列表:", expiredCertIds)
	// 查询要过期的证书 =========================================== end
	// 查询域名重复的列表 old_certListCertIds 去掉最 InsertTime最新的一个 =========================================== start
	domainLatestTime := make(map[string]int64) // 存储每个域名的最新 InsertTime
	duplicateIDs := make(map[string][]int64)   // 存储重复域名的证书 ID
	// 第一次遍历，填充 domainMap 和 duplicateIDs
	for _, cert := range certList {
		startDate, err := ali_parseInsertTime(cert.StartDate)
		if err != nil {
			return
		}

		if latestTime, exists := domainLatestTime[cert.CommonName]; exists {
			if startDate > latestTime {
				domainLatestTime[cert.CommonName] = startDate
			}
			duplicateIDs[cert.CommonName] = append(duplicateIDs[cert.CommonName], cert.CertificateId)
		} else {
			domainLatestTime[cert.CommonName] = startDate
			duplicateIDs[cert.CommonName] = []int64{cert.CertificateId}
		}
	}
	// 第二次遍历，找出不是最新的证书 ID
	old_certListCertIds := []int64{}

	for domain, ids := range duplicateIDs {
		if len(ids) > 1 { // 只有在域名重复时才处理
			latestTime := domainLatestTime[domain]
			for _, id := range ids {
				// 在这里需要根据 ID 找到对应的 InsertTime
				startDate, err := ali_getStartDateByID(id, certList)
				if err != nil {
					return
				}
				if startDate < latestTime {
					log.Println("准备删除的 域名重复的 证书id列表 时间小:", id)
					old_certListCertIds = append(old_certListCertIds, id)
				}
				if startDate == latestTime { //过期时间相同 那么 id 最大的那个 是最新的 不应该添加到 old_certListCertIds
					maxID := ids[0] //暂定 第一个是最大的
					for _, id_in_ids := range ids {
						if id_in_ids > maxID {
							maxID = id_in_ids //有更大的
						}
					}
					// 看看 是否比 maxID 小
					for _, id := range ids {
						if id < maxID {
							old_certListCertIds = append(old_certListCertIds, id)
						}
					}

				}
			}
		}
	}
	// 查询域名重复的列表 old_certListCertIds 去掉最 InsertTime最新的一个 =========================================== end
	// 合并 expiredCertIds 和 old_certListCertIds 去掉重复的
	uniqueCertIds := make(map[int64]struct{})
	for _, id := range expiredCertIds {
		uniqueCertIds[id] = struct{}{}
	}
	for _, id := range old_certListCertIds {
		uniqueCertIds[id] = struct{}{}
	}
	// 将唯一的证书 ID 添加到切片中
	var finalCertIds []int64
	for id := range uniqueCertIds {
		finalCertIds = append(finalCertIds, id)
	}

	log.Println("准备删除的 唯一证书id列表:", finalCertIds)
	// 开始删除 finalCertIds
	for _, id := range finalCertIds {

		// 查询证书列表
		params := &openapi.Params{
			// 接口名称
			Action: tea.String("DeleteUserCertificate"),
			// 接口版本
			Version: tea.String("2020-04-07"),
			// 接口协议
			Protocol: tea.String("HTTPS"),
			// 接口 HTTP 方法
			Method:   tea.String("POST"),
			AuthType: tea.String("AK"),
			Style:    tea.String("RPC"),
			// 接口 PATH
			Pathname: tea.String("/"),
			// 接口请求体内容格式
			ReqBodyType: tea.String("json"),
			// 接口响应体内容格式
			BodyType: tea.String("json"),
		}
		config := &openapi.Config{
			AccessKeyId:     tea.String(ak.Id),
			AccessKeySecret: tea.String(ak.Secret),
		}
		config.Endpoint = tea.String("cas.aliyuncs.com")
		client := &openapi.Client{}
		var _err error
		client, _err = openapi.NewClient(config)
		if _err != nil {
			log.Printf("NewClient failed: %v\n", _err)
			return
		}
		queries := map[string]interface{}{}
		queries["CertId"] = tea.Int(int(id))
		runtime := &util.RuntimeOptions{}
		request := &openapi.OpenApiRequest{
			Query: openapiutil.Query(queries),
		}
		// 复制代码运行请自行打印 API 的返回值
		// 返回值实际为 Map 类型，可从 Map 中获得三类数据：响应体 body、响应头 headers、HTTP 返回的状态码 statusCode。
		resp, _err = client.CallApi(params, request, runtime)
		if _err != nil {
			return
		}
		respMap := resp["body"].(map[string]interface{})

		log.Printf("阿里云成功删除证书: %d\n%+v\n", id, respMap)
	}
}

// 结构体
type Ali_Cert_list_one struct {
	CommonName    string
	StartDate     string
	CertificateId int64
	Expired       bool
}

// 将时间字符串转换为时间戳
func ali_parseInsertTime(insertTimeStr string) (int64, error) {
	layout := "2006-01-02" // 时间格式
	t, err := time.Parse(layout, insertTimeStr)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

func ali_getStartDateByID(id int64, certList []Ali_Cert_list_one) (int64, error) {
	for _, cert := range certList {
		if cert.CertificateId == id {
			insertTime, err := ali_parseInsertTime(cert.StartDate)
			if err != nil {
				return 0, err
			}
			return insertTime, nil
		}
	}
	return 0, nil
}
