package getConfig

import (
	"encoding/json"
	"f5-proxy-master/logger"
	"io/ioutil"
	"os"
)

var (
	LinuxPotCfg    = "/etc/cfg.json"
	LinuxFilterCfg = "/etc/strategy.json"
)

type ResPacket struct {
	Code           string        `json:"code"`
	Flag           string        `json:"flag"`
	MonitorProcess []interface{} `json:"monitor_process"`
	StrategyInfo   struct {
		CmdFilter   []interface{} `json:"cmdFilter"`
		FeatureCode []struct {
			SIp         []interface{} `json:"sIp"`
			Black       string        `json:"black"`
			DIp         []interface{} `json:"dIp"`
			FeatureName []string      `json:"featureName"`
			SPort       []interface{} `json:"sPort"`
			DPort       []interface{} `json:"dPort"`
		} `json:"featureCode"`
		FileFilter      []interface{} `json:"fileFilter"`
		MonitorStrategy struct {
			OpenStatus string `json:"openStatus"`
		} `json:"monitorStrategy"`
		NetFilter []struct {
			SIp     []string      `json:"sIp"`
			EvtType []string      `json:"evtType"`
			DIp     []string      `json:"dIp"`
			DPort   []interface{} `json:"dPort"`
			Black   string        `json:"black"`
			SPort   []interface{} `json:"sPort"`
		} `json:"netFilter"`
		ProcFilter []interface{} `json:"procFilter"`
	} `json:"strategy_info"`
	UpdateInfo struct {
	} `json:"update_info"`
}

type PotFilter struct {
	Latest struct {
		NetFilter []struct {
			SIp     []string      `json:"sIp"`
			EvtType []string      `json:"evtType"`
			DIp     []string      `json:"dIp"`
			DPort   []interface{} `json:"dPort"`
			Black   string        `json:"black"`
			SPort   []interface{} `json:"sPort"`
		} `json:"netFilter"`
		FeatureCode []struct {
			SIp         []interface{} `json:"sIp"`
			Black       string        `json:"black"`
			DIp         []interface{} `json:"dIp"`
			FeatureName []string      `json:"featureName"`
			SPort       []interface{} `json:"sPort"`
			DPort       []interface{} `json:"dPort"`
		} `json:"featureCode"`
		ProcFilter      []interface{} `json:"procFilter"`
		MonitorStrategy struct {
			OpenStatus string `json:"openStatus"`
		} `json:"monitorStrategy"`
		CmdFilter  []interface{} `json:"cmdFilter"`
		FileFilter []interface{} `json:"fileFilter"`
	} `json:"latest"`
	Dynamic struct {
		TopProc struct {
			Black    string        `json:"black"`
			ProcPath []interface{} `json:"procPath"`
		} `json:"TopProc"`
		PidTable struct {
			Field1 int `json:"1631"`
			Field2 int `json:"1605"`
			Field3 int `json:"1801"`
			Field4 int `json:"1805"`
		} `json:"pidTable"`
	} `json:"dynamic"`
	Backup struct {
		CmdFilter []struct {
			ProcPath []string `json:"procPath"`
		} `json:"cmdFilter"`
		FileFilter []struct {
			EvtType  interface{} `json:"evtType,omitempty"`
			FileName interface{} `json:"fileName,omitempty"`
			ProcPath interface{} `json:"procPath,omitempty"`
		} `json:"fileFilter"`
		NetFilter []struct {
			Field1   string   `json:"2,omitempty"`
			Field2   string   `json:"1,omitempty"`
			SPort    string   `json:"sPort,omitempty"`
			DPort    string   `json:"dPort,omitempty"`
			SIp      []string `json:"sIp,omitempty"`
			DIp      []string `json:"dIp,omitempty"`
			ProcPath string   `json:"procPath,omitempty"`
		} `json:"netFilter"`
	} `json:"backup"`
}

type PotConfig struct {
	Remoteport int    `json:"remoteport"`
	Mid        string `json:"mid"`
	LuaType    string `json:"lua_type"`
	Remoteip   string `json:"remoteip"`
	DataType   string `json:"data_type"`
	PotId      string `json:"pot_id"`
	HostName   string `json:"host_name"`
	Ssl        bool   `json:"ssl"`
	Ver        string `json:"ver"`
	HostOs     string `json:"host_os"`
}

func GetPotConfig() PotConfig {
	jsonFile, err := os.Open(LinuxPotCfg)
	if err != nil {
		logger.Error(err.Error())

	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var potconfig PotConfig

	json.Unmarshal(byteValue, &potconfig)

	return potconfig

}

func GetFilterConfig() PotFilter {
	jsonFile, err := os.Open(LinuxFilterCfg)
	if err != nil {
		logger.Error(err.Error())
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var potfilter PotFilter

	json.Unmarshal(byteValue, &potfilter)

	return potfilter

}
