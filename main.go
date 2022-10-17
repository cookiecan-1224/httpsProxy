package main

import (
	"context"
	"encoding/json"
	"f5-proxy-master/getConfig"
	"f5-proxy-master/jroutinepool"
	"f5-proxy-master/logger"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	CONFIG     = new(Config)
	POTCONFIG  = getConfig.PotConfig{}
	POTFILTER  = getConfig.PotFilter{}
	srv        http.Server
	prg        program
	vclearSvr  service.Service
	bInstall   = flag.Bool("i", false, "install the server")
	bUnInstall = flag.Bool("u", false, "uninstall the server")
	bRun       = flag.Bool("r", false, "run the server")
	rbcp       *jroutinepool.MultipleRoutineConsumePool
	localAddr  string
	c          *cache.Cache
	TimeTicker *time.Ticker
	FaildeData []Body
	eventurl   = "https://192.168.75.200:30443/api1/send_data/"
	hearturl   = "https://192.168.75.200:30443/api1/beat/"
)

type heartpkt struct {
	Host_ip      string  `json:"host_ip"`
	Host_ip6     string  `json:"host_ip6"`
	Host_mac     string  `json:"host_mac"`
	Pot_id       string  `json:"pot_id"`
	Mid          string  `json:"mid"`
	Data_type    string  `json:"data_type"`
	Host_os      string  `json:"host_os"`
	Host_name    string  `json:"host_name"`
	Prog_ver     string  `json:"prog_ver"`
	Localip      string  `json:"localip"`
	Cpu_total    int     `json:"cpu_total"`
	Cpu_used     float64 `json:"cpu_used"`
	Memory_total uint64  `json:"memory_total"`
	Memory_used  float64 `json:"memory_used"`
}

// var cmd Cmd
//var srv http.Server

type Config struct {
	Local    *Location `yaml:"Location"`
	Proxy    *Proxy    `yaml:"Proxy"`
	Protocol string    `yaml:"Protocol"`
}

type Location struct {
	Port string `yaml:"Port"`
	IP   string `yaml:"IP"`
}

type Proxy struct {
	Port string `yaml:"Port"`
	IP   string `yaml:"IP"`
}

type program struct {
	ctx     context.Context
	canfunc context.CancelFunc
}

func init() {

	TimeTicker = time.NewTicker(180 * time.Second) //定时器

	serviceArgs := make([]string, 0)

	serviceArgs = append(serviceArgs, "-r")

	svcConfig := &service.Config{
		Name:        "proxy",
		DisplayName: "proxy",
		Description: "proxy",
		Arguments:   serviceArgs,
		Executable:  os.Args[0],
	}

	prg.ctx, prg.canfunc = context.WithCancel(context.Background())
	vclearSvr, _ = service.New(&prg, svcConfig)

	var loghook = logger.LoggerConfig{
		Filename:   "/var/log/" + os.Args[0] + ".log", // 日志文件路径
		MaxSize:    100,                               // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 10,                                // 日志文件最多保存多少个备份
		MaxAge:     30,                                // 文件最多保存多少天
		Level:      zap.InfoLevel,
	}

	loghook.MessageKey = "prox"
	loghook.Encoder = zapcore.NewJSONEncoder
	logger.Initlog(loghook, zap.AddCaller(), zap.Development())

	yamlFile, err := ioutil.ReadFile("/etc/config.yml") //read config to global
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, CONFIG)
	if err != nil {
		logger.Error(err.Error())
	}

	//判断ip是否为空
	if CONFIG.Proxy.IP == "" {
		CONFIG.Proxy.IP = getLocalAddr().Ipv4
	}
	if CONFIG.Local.IP == "" {
		CONFIG.Local.IP = "127.0.0.1"
	}
}

func stopListen() {
	err := srv.Shutdown(nil)
	if err != nil {
		logger.Error(err)
	}
}

func startHttpListen() {
	remote := fmt.Sprintf("%s://%s:%s", CONFIG.Protocol, CONFIG.Proxy.IP, CONFIG.Proxy.Port)

	h := &handle{reverseProxy: remote}
	srv.Addr = ":" + CONFIG.Local.Port
	srv.Handler = h

	logger.Infof("Listening on %s, forwarding to %s", srv.Addr, remote)
	//go func() {
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("ListenAndServe: ", err)
	}

}

func (p *program) run() {

	p.ctx, p.canfunc = context.WithCancel(context.Background())
	var binitEnvOnce sync.Once

	for {
		//	runSCanCap()
		if getLocalAddr().Ipv4 != "" {
			binitEnvOnce.Do(func() {
				go startHttpListen()
				go ConfigureHeartpkt() //开启发送心跳包协程
				//go Resend()
			})
		}

		select {
		case <-p.ctx.Done(): //程序退出
			stopListen()
			return
		case <-time.After(30 * time.Second):
		}

	}

}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.canfunc != nil {
		p.canfunc()
	}
	return nil
}

//func Resend() {
//	for {
//		select {
//		case <-TimeTicker.C:
//			if len(FaildeData) != 0 {
//				ResetData, _ := json.Marshal(FaildeData)
//
//				tr := &http.Transport{
//					DisableKeepAlives: true,
//					TLSClientConfig:   &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS12},
//				}
//				client := &http.Client{Transport: tr}
//				defer func() {
//					if err := recover(); err != nil {
//						logger.Infof("重发连接超时:", err)
//						return
//					}
//				}()
//				req, err := http.NewRequest("POST", eventurl, bytes.NewBuffer(ResetData))
//				if err != nil {
//					logger.Error(err.Error())
//				}
//				req.Header.Set("Content-Type", "application/json;charset=utf-8")
//				req.Header.Set("Connection", "close")
//
//				res, err := client.Do(req)
//
//				if err != nil {
//					fmt.Println("重发失败，等待下一次重发", err.Error())
//					panic(err.Error())
//				} else {
//					fmt.Println("重发成功", string(ResetData))
//					FaildeData = []Body{}
//				}
//
//				_, err = ioutil.ReadAll(res.Body)
//				if err != nil {
//					fmt.Println("Fatal error ", err.Error())
//				}
//			}
//		default:
//
//		}
//	}
//}

func ConfigureHeartpkt() {

	//sigs := make(chan os.Signal, 1)
	//signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var heartpkt heartpkt
	ipaddr := getLocalAddr()

	heartpkt.Host_ip = ipaddr.Ipv4
	heartpkt.Host_ip6 = ipaddr.Ipv6
	heartpkt.Host_mac = ipaddr.Mac
	heartpkt.Pot_id = POTCONFIG.PotId
	heartpkt.Mid = POTCONFIG.Mid
	heartpkt.Data_type = POTCONFIG.DataType
	heartpkt.Host_os = POTCONFIG.HostOs
	heartpkt.Host_name = POTCONFIG.HostName
	heartpkt.Prog_ver = POTCONFIG.Ver
	heartpkt.Localip = ipaddr.Ipv4
	heartpkt.Cpu_total = getConfig.GetCpuCount()
	heartpkt.Cpu_used = getConfig.GetCpuPercent()
	heartpkt.Memory_total = getConfig.GetMenTotal() / 1024 / 1024
	heartpkt.Memory_used = getConfig.GetMemPercent()
	heartpkt_data, _ := json.Marshal(heartpkt)

	fmt.Println(heartpkt)
	for {
		//select {
		//case <-sigs:
		//	return
		//default:
		//update_data := send_post(heartpkt_data, "https://192.168.75.200:30443/api1/beat/")
		//go updateFilter(update_data)
		send_post(heartpkt_data, hearturl)
		rand_time := rand.Intn(181)
		time.Sleep(time.Duration(rand_time) * time.Second)
		//}
	}
}

func main() {
	POTCONFIG = getConfig.GetPotConfig()
	POTFILTER = getConfig.GetFilterConfig()
	flag.Parse()
	if *bInstall {
		vclearSvr.Install()
		vclearSvr.Stop()
		vclearSvr.Start()
		return
	}

	if *bUnInstall {
		vclearSvr.Stop()
		vclearSvr.Uninstall()
		return
	}

	if *bRun {
		vclearSvr.Run()
		return
	}

	flag.Usage()

}
