package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"f5-proxy-master/getConfig"
	"f5-proxy-master/logger"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Body struct {
	Body string `json:"body"`
}

type IpPort struct {
	DstIP   string `json:"dstIP"`
	DstPort int    `json:"dstPort"`
	SrcIP   string `json:"srcIP"`
	SrcPort int    `json:"srcPort"`
}

type potEvent struct {
	Ver       string  `json:"ver"`
	HostIp    string  `json:"host_ip"`
	Payload   Payload `json:"payload"`
	DataType  string  `json:"data_type"`
	ActName   string  `json:"act_name"`
	ActType   string  `json:"act_type"`
	HostMac   string  `json:"host_mac"`
	HostIp6   string  `json:"host_ip6"`
	Timestamp string  `json:"timestamp"`
	HostName  string  `json:"host_name"`
	PotId     string  `json:"pot_id"`
	Mid       string  `json:"mid"`
	HostOs    string  `json:"host_os"`
}

type Payload struct {
	Input         string `json:"input"`
	Pid1          int    `json:"pid1"`
	Output        string `json:"output"`
	Srcport       int    `json:"srcport"`
	Pname1        string `json:"pname1"`
	Dstip         string `json:"dstip"`
	Srcip         string `json:"srcip"`
	Pname0        string `json:"pname0"`
	Dstport       int    `json:"dstport"`
	User          string `json:"user"`
	Ugroup        string `json:"ugroup"`
	Pid0          int    `json:"pid0"`
	Timestamp     string `json:"timestamp"`
	Writelen      int    `json:"writelen"`
	Lastwritetime int64  `json:"lastwritetime"`
	Containerid   string `json:"containerid"`
	Env           string `json:"env"`
}

// 发送蜜罐心跳和事件
func send_post(json_data []byte, url string) []byte {
	//跳过证书认证
	tr := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS12},
	}
	client := &http.Client{Transport: tr}

	defer func() {
		if err := recover(); err != nil {
			logger.Error(err)
			return
		}
	}()

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(json_data))
	if err != nil {
		logger.Error(err.Error())
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Connection", "close")

	//res, err := client.Post(url,
	//	"application/json;charset=utf-8", bytes.NewBuffer(json_data))
	res, err := client.Do(req)

	if err != nil {
		logger.Error("failed connecting:", url, err)
		//FaildeData = append(FaildeData, Body{Body: string(json_data)})
		panic(err.Error())
	}

	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error("read io failed", err.Error())
	}
	//_ = (*string)(unsafe.Pointer(&content)) //转化为string,优化内存
	client.CloseIdleConnections()
	return content

}

// 判断是否过滤数据
func isFilter(res *http.Response) bool {
	IP_port := getIpPort(res)
	//监控动态修改区内容
	latest_netfilter_len := len(POTFILTER.Latest.NetFilter)
	for i := 0; i < latest_netfilter_len; i++ {

		latest_netfilter_sip_len := len(POTFILTER.Latest.NetFilter[i].SIp)
		latest_netfilter_dip_len := len(POTFILTER.Latest.NetFilter[i].DIp)

		for j := 0; j < latest_netfilter_sip_len; j++ {
			if IP_port.SrcIP == POTFILTER.Latest.NetFilter[i].SIp[j] {
				return true
			}
		}

		for j := 0; j < latest_netfilter_dip_len; j++ {
			if IP_port.DstIP == POTFILTER.Latest.NetFilter[i].DIp[j] {
				return true
			}
		}

	}
	return false
}

// 更新过滤规则
func updateFilter(newFilter []byte) {
	isUpdate := false
	_potconfig := getConfig.ResPacket{}
	json.Unmarshal(newFilter, &_potconfig)
	if _potconfig.StrategyInfo.CmdFilter != nil {
		POTFILTER.Latest.CmdFilter = _potconfig.StrategyInfo.CmdFilter
		isUpdate = true
	}
	if _potconfig.StrategyInfo.FeatureCode != nil {
		POTFILTER.Latest.FeatureCode = _potconfig.StrategyInfo.FeatureCode
		isUpdate = true
	}
	if _potconfig.StrategyInfo.FileFilter != nil {
		POTFILTER.Latest.FileFilter = _potconfig.StrategyInfo.FileFilter
		isUpdate = true
	}
	if _potconfig.StrategyInfo.MonitorStrategy.OpenStatus != "" {
		POTFILTER.Latest.MonitorStrategy.OpenStatus = _potconfig.StrategyInfo.MonitorStrategy.OpenStatus
		isUpdate = true
	}
	if _potconfig.StrategyInfo.NetFilter != nil {
		POTFILTER.Latest.NetFilter = _potconfig.StrategyInfo.NetFilter
		isUpdate = true
	}
	if _potconfig.StrategyInfo.ProcFilter != nil {
		POTFILTER.Latest.ProcFilter = _potconfig.StrategyInfo.ProcFilter
		isUpdate = true
	}
	if isUpdate == true {
		update_data, err := json.Marshal(POTFILTER)
		if err != nil {
			logger.Infof("write to strategy.json err!", err)
		} else {
			ioutil.WriteFile("/etc/strategy.json", update_data, 0777)
		}
	}
}

func getIpPort(res *http.Response) IpPort {
	var IPPort IpPort

	dst_ip_port := res.Request.Host
	src_ip_port := res.Request.RemoteAddr

	dst_ip, dst_port, err_dst := net.SplitHostPort(dst_ip_port)

	if err_dst != nil {
		logger.Error(err_dst)
	} else {
		IPPort.DstIP = dst_ip
		dstport, _ := strconv.Atoi(dst_port)
		IPPort.DstPort = dstport
	}

	src_ip, src_port, err_src := net.SplitHostPort(src_ip_port)

	if err_src != nil {
		logger.Error(err_src)
	} else {
		IPPort.SrcIP = src_ip
		srcport, _ := strconv.Atoi(src_port)
		IPPort.SrcPort = srcport
	}

	return IPPort

}

func stdSendEvent(res *http.Response) []byte {

	IP_Port := getIpPort(res)

	_req_data := reqString(res)
	_res_data := resString(res)

	_lastwritetime := time.Now().Unix()
	_timestamp := time.Unix(_lastwritetime, 0).Format("2006-01-02 15:04:05")
	PotEventData := potEvent{
		Ver:    POTCONFIG.Ver,
		HostIp: IP_Port.DstIP,
		Payload: Payload{
			Input: base64.StdEncoding.EncodeToString([]byte(_req_data)),
			//Pid1:          52901,
			Output:  base64.StdEncoding.EncodeToString([]byte(_res_data)),
			Srcport: IP_Port.SrcPort,
			//Pname1:        "nginx",
			Dstip: IP_Port.DstIP,
			Srcip: IP_Port.SrcIP,
			//Pname0:        "nginx",
			Dstport: IP_Port.DstPort,
			//User:          "nobody",
			//Ugroup:        "nobody",
			//Pid0:          21666,
			Timestamp:     _timestamp,
			Writelen:      len(_res_data),
			Lastwritetime: _lastwritetime,
		},
		DataType: POTCONFIG.DataType,
		//ActName:   "net_inout",
		//ActType:   "net_net",
		HostMac:   getLocalAddr().Mac,
		HostIp6:   getLocalAddr().Ipv6,
		Timestamp: _timestamp,
		HostName:  POTCONFIG.HostName,
		PotId:     POTCONFIG.PotId,
		Mid:       POTCONFIG.Mid,
		HostOs:    POTCONFIG.HostOs,
	}

	json_byte, _ := json.Marshal(PotEventData)
	_PotEventData := string(json_byte)
	logger.Infof(_PotEventData)
	//fmt.Println(_PotEventData)
	bodyData := []Body{
		{Body: _PotEventData},
	}

	_bodyData, _ := json.Marshal(bodyData)
	return _bodyData
}

func scanCapdataHandler(v interface{}) {
	switch send_data := v.(type) {
	case []byte:
		send_post(send_data, eventurl)
	}
}

// 拼接请求体
func reqString(res *http.Response) string {
	req_data := res.Request.Method + " /" + res.Request.URL.Host + res.Request.URL.Path + "\r\n" + "Host:" + res.Request.Host + "\r\n"
	for k, v := range res.Request.Header {
		req_data += k + ":"
		for i := 0; i < len(v); i++ {
			if i != len(v)-1 {
				req_data += v[i] + ","
			} else {
				req_data += v[i] + "\r\n"
			}

		}

	}
	req_data += "\r\n"
	return req_data

}

// 拼接响应体
func resString(res *http.Response) string {
	res_data := res.Proto + " " + res.Status
	for k, v := range res.Header {
		res_data += k + ":"
		for i := 0; i < len(v); i++ {
			if i != len(v)-1 {
				res_data += v[i] + ","
			} else {
				res_data += v[i] + "\r\n"
			}

		}

	}

	res_body, _ := ioutil.ReadAll(res.Body)
	res.Body = ioutil.NopCloser(bytes.NewBuffer(res_body))
	res_data += "\r\n" + string(res_body)
	return res_data
}
