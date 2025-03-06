package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tchttp "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/http"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
)

func upload_to_ssl_and_deploy_to_cdn_tencent(v Domains, d Deploy_To) (err error) {
	certId, err := upload_to_ssl_tencent(v, d)
	if err != nil {
		return err
	}
	err = deploy_to_cdn_tencent(d, certId)
	return err
}

func deploy_to_cdn_tencent(d Deploy_To, certId string) error {
	log.Println("开始部署证书到腾讯云", certId, d.CdnDomains)
	if len(d.CdnDomains) == 0 {
		log.Println("腾讯云域名为空,跳过")
		return nil
	}
	cdnDomainListString := ""
	for _, domain := range d.CdnDomains {
		//格式应该是 "www.shiyuxin.ltd|on","doc.shiyuxin.ltd|on"
		cdnDomainListString += "\"" + domain + "|on\","

	}
	// 如果最后一个字符是逗号，则去掉
	if cdnDomainListString[len(cdnDomainListString)-1] == ',' {
		cdnDomainListString = cdnDomainListString[:len(cdnDomainListString)-1]
	}
	credential := common.NewCredential(CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Id, CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Secret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ssl.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	client := common.NewCommonClient(credential, "", cpf).WithLogger(log.Default())

	request := tchttp.NewCommonRequest("ssl", "2019-12-05", "DeployCertificateInstance")
	//  params := "{\"CertificateId\":\"LgLV36Pe\",\"InstanceIdList\":[\"entrypoint.cf-cdn-ns.work|on\"],\"ResourceType\":\"cdn\"}"
	//	params := "{\"CertificateId\":\"" + CertificateId + "\",\"InstanceIdList\":[\"www.shiyuxin.ltd|on\",\"doc.shiyuxin.ltd|on\"],\"ResourceType\":\"cdn\",\"Status\":1}"
	//params := "{\"CertificateId\":\"" + CertificateId + "\",\"InstanceIdList\":[\"" + domainListString + "\"],\"ResourceType\":\"cdn\",\"Status\":1}"
	params := "{\"CertificateId\":\"" + certId + "\",\"InstanceIdList\":[" + cdnDomainListString + "],\"ResourceType\":\"cdn\",\"Status\":1}"
	log.Println("部署证书的参数:", params)

	err := request.SetActionParameters(params)
	if err != nil {
		return err
	}

	response := tchttp.NewCommonResponse()
	err = client.Send(request, response)
	if err != nil {
		log.Println("腾讯云部署证书失败:", err.Error())
	}

	log.Println("腾讯云部署证书的返回:", string(response.GetBody()))
	return nil
}

func upload_to_ssl_tencent(v Domains, d Deploy_To) (certId string, err error) {
	//从文件获取CertificatePublicKey
	CertificatePublicKey, err := os.ReadFile(PATCH_SSL_DIR + "/" + v.DomianName + ".cer")
	if err != nil {
		log.Println("腾讯云读取证书公钥文件失败,跳过:", err)
		return
	}
	//处理CertificatePublicKey 把换行符号替换 字符串\n
	CertificatePublicKey = []byte(strings.ReplaceAll(string(CertificatePublicKey), "\n", "\\n"))

	//从文件获取CertificatePrivateKey
	CertificatePrivateKey, err := os.ReadFile(PATCH_SSL_DIR + "/" + v.DomianName + ".key")
	if err != nil {
		log.Println("读取证书私钥文件失败:", err)
		return
	}
	//处理CertificatePrivateKey 把换行符号替换 字符串\n
	CertificatePrivateKey = []byte(strings.ReplaceAll(string(CertificatePrivateKey), "\n", "\\n"))

	credential := common.NewCredential(CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Id, CONFIG_TOML.CloudAK[d.Cloud][d.Account_tag].Secret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "ssl.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	client := common.NewCommonClient(credential, "", cpf).WithLogger(log.Default())

	request := tchttp.NewCommonRequest("ssl", "2019-12-05", "UploadCertificate")
	params := "{\"CertificatePublicKey\":\"" + string(CertificatePublicKey) + "\",\"CertificatePrivateKey\":\"" + string(CertificatePrivateKey) + "\",\"Alias\":\"" + CONFIG_TOML.SSL_MANAGE_ALIAS + "\",\"CertificateUse\":\"" + CONFIG_TOML.CertificateUse + "\"}"
	//log.Println("上传证书的参数:", params)
	err = request.SetActionParameters(params)
	if err != nil {
		return
	}

	response := tchttp.NewCommonResponse()
	err = client.Send(request, response)
	if err != nil {
		log.Println("上传证书失败:", err.Error())
	}

	log.Println("上传证书的返回:", string(response.GetBody()))

	//从返回的消息的json中获取返回 cert的id CertificateId
	var result struct {
		Response struct {
			CertificateId string
		}
	}
	err = json.Unmarshal(response.GetBody(), &result)
	if err != nil {
		return
	}
	certId = result.Response.CertificateId

	return certId, nil
}

// 结构体
type Tencent_Cert_list_one struct {
	Domain        string
	InsertTime    string
	CertificateId string
	IsExpiring    bool
}

func tencent_delete_expired_or_repeated(ak ak_cloud) {
	// 查询域名列表 =========================================== start
	credential := common.NewCredential(ak.Id, ak.Secret)
	cpf := profile.NewClientProfile()

	cpf.HttpProfile.Endpoint = "ssl.tencentcloudapi.com"
	cpf.HttpProfile.ReqMethod = "POST"
	client := common.NewCommonClient(credential, "", cpf).WithLogger(log.Default())

	request := tchttp.NewCommonRequest("ssl", "2019-12-05", "DescribeCertificates")
	//params := "{}"
	params := "{\"Limit\":1000}"

	err := request.SetActionParameters(params)
	if err != nil {
		return
	}

	response := tchttp.NewCommonResponse()
	err = client.Send(request, response)
	if err != nil {
		fmt.Println("fail to invoke api:", err.Error())
	}

	//fmt.Println(string(response.GetBody()))

	var result map[string]interface{}
	err = json.Unmarshal(response.GetBody(), &result)
	if err != nil {
		log.Println("解析json失败:", err)
	}
	var certList []Tencent_Cert_list_one

	//从json中获取Certificates
	for _, certificate := range result["Response"].(map[string]interface{})["Certificates"].([]interface{}) {
		certificateMap := certificate.(map[string]interface{})
		certSANs := make([]string, 0)
		for _, san := range certificateMap["CertSANs"].([]interface{}) {
			certSANs = append(certSANs, san.(string))
		}
		cert := Tencent_Cert_list_one{
			Domain:        certificateMap["Domain"].(string),
			InsertTime:    certificateMap["InsertTime"].(string),
			CertificateId: certificateMap["CertificateId"].(string),
			IsExpiring:    certificateMap["IsExpiring"].(bool),
		}
		certList = append(certList, cert)
	}
	// 查询证书列表 =========================================== end
	// 查询要过期的证书 =========================================== start
	expiredCertIds := []string{}
	for _, cert := range certList {
		if cert.IsExpiring {
			expiredCertIds = append(expiredCertIds, cert.CertificateId)
		}
	}
	log.Println("准备删除的 过期的 证书id列表:", expiredCertIds)
	// 查询要删除的证书 =========================================== end
	// 查询域名重复的列表 old_certListCertIds 去掉最 InsertTime最新的一个 =========================================== start
	domainLatestTime := make(map[string]int64) // 存储每个域名的最新 InsertTime
	duplicateIDs := make(map[string][]string)  // 存储重复域名的证书 ID
	// 第一次遍历，填充 domainMap 和 duplicateIDs
	for _, cert := range certList {
		insertTime, err := tencent_parseInsertTime(cert.InsertTime)
		if err != nil {
			return
		}

		if latestTime, exists := domainLatestTime[cert.Domain]; exists {
			if insertTime > latestTime {
				domainLatestTime[cert.Domain] = insertTime
			}
			duplicateIDs[cert.Domain] = append(duplicateIDs[cert.Domain], cert.CertificateId)
		} else {
			domainLatestTime[cert.Domain] = insertTime
			duplicateIDs[cert.Domain] = []string{cert.CertificateId}
		}
	}
	// 第二次遍历，找出不是最新的证书 ID
	var old_certListCertIds []string
	for domain, ids := range duplicateIDs {
		if len(ids) > 1 { // 只有在域名重复时才处理
			latestTime := domainLatestTime[domain]
			for _, id := range ids {
				// 在这里需要根据 ID 找到对应的 InsertTime
				insertTime, err := tencent_getInsertTimeByID(id, certList)
				if err != nil {
					return
				}
				if insertTime < latestTime {
					old_certListCertIds = append(old_certListCertIds, id)
				}
			}
		}
	}
	// 查询域名重复的列表 old_certListCertIds 去掉最 InsertTime最新的一个 =========================================== end

	// 合并 expiredCertIds 和 old_certListCertIds 去掉重复的
	uniqueCertIds := make(map[string]struct{})
	for _, id := range expiredCertIds {
		uniqueCertIds[id] = struct{}{}
	}
	for _, id := range old_certListCertIds {
		uniqueCertIds[id] = struct{}{}
	}

	// 将唯一的证书 ID 添加到切片中
	var finalCertIds []string
	for id := range uniqueCertIds {
		finalCertIds = append(finalCertIds, id)
	}

	log.Println("准备删除的 唯一证书id列表:", finalCertIds)

	// 开始删除 finalCertIds

	for _, id := range finalCertIds {
		credential := common.NewCredential(ak.Id, ak.Secret)
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Endpoint = "ssl.tencentcloudapi.com"
		cpf.HttpProfile.ReqMethod = "POST"
		client := common.NewCommonClient(credential, "", cpf).WithLogger(log.Default())
		request := tchttp.NewCommonRequest("ssl", "2019-12-05", "DeleteCertificate")
		params := "{\"CertificateId\":\"" + id + "\"}"
		err := request.SetActionParameters(params)
		if err != nil {
			log.Println("设置请求参数失败:", err)
			return
		}

		response := tchttp.NewCommonResponse()
		err = client.Send(request, response)
		if err != nil {
			log.Println("调用api失败:", err.Error())
		}

		log.Println("删除证书成功:", id, string(response.GetBody()))

	}

}

func tencent_getInsertTimeByID(id string, certList []Tencent_Cert_list_one) (int64, error) {
	for _, cert := range certList {
		if cert.CertificateId == id {
			insertTime, err := tencent_parseInsertTime(cert.InsertTime)
			if err != nil {
				return 0, err
			}
			return insertTime, nil
		}
	}

	return 0, nil
}

// 将时间字符串转换为时间戳
func tencent_parseInsertTime(insertTimeStr string) (int64, error) {
	layout := "2006-01-02 15:04:05" // 时间格式
	t, err := time.Parse(layout, insertTimeStr)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}
