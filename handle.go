package main

import (
	"context"
	"crypto/tls"
	"f5-proxy-master/jbufpool"
	"f5-proxy-master/jroutinepool"
	"f5-proxy-master/logger"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

//var rbcp *jroutinepool.MultipleRoutineConsumePool

func init() {

	rbcp = jroutinepool.NewMultipleRoutineConsumePool(64, func() io.Writer {
		return jbufpool.NewBytebufPoolConsumeRoutine(512, scanCapdataHandler)
	})
}

//var localAddr string

type IPaddr struct {
	Ipv6 string `json:"ipv6"`
	Ipv4 string `json:"ipv4"`
	Mac  string `json:"mac"`
}

func getLocalAddr() IPaddr {
	var ipaddr IPaddr

	infs, err := net.Interfaces()
	if err != nil {
		ipaddr.Ipv4 = ""
		ipaddr.Ipv6 = ""
		ipaddr.Mac = ""
		return ipaddr
	}

	//没有up状态则过滤

	//var ncrds []NetCard
	// index := 1

	for _, inf := range infs {

		if inf.Flags&net.FlagLoopback == net.FlagLoopback {
			continue
		}

		if inf.Flags&net.FlagPointToPoint == net.FlagPointToPoint {
			continue
		}

		if inf.Flags&net.FlagUp != net.FlagUp {
			continue
		}

		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}

		ipaddr.Mac = inf.HardwareAddr.String()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipNet.IP.To4() == nil {
				ipaddr.Ipv6 = ipNet.IP.String()
				continue
			}

			if !ipNet.IP.IsGlobalUnicast() {
				continue
			}

			ipaddr.Ipv4 = ipNet.IP.String()
			//return localAddr
		}
		return ipaddr
	}

	return ipaddr
}

type handle struct {
	reverseProxy string
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func (this *handle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Println(r.RemoteAddr + " " + r.Method + " " + r.URL.String() + " " + r.Proto + " " + r.UserAgent())
	remote, err := url.Parse(this.reverseProxy)
	if err != nil {
		log.Fatalln(err)
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}
	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		remote := strings.Split(addr, ":")
		// if cmd.ip == "" {
		// 	resolver := dns_resolver.New([]string{"114.114.114.114", "114.114.115.115", "119.29.29.29", "223.5.5.5", "8.8.8.8", "208.67.222.222", "208.67.220.220"})
		// 	resolver.RetryTimes = 5
		// 	ip, err := resolver.LookupHost(remote[0])
		// 	if err != nil {
		// 		log.Println(err)
		// 		cmd.ip = remote[0]

		// 	} else {
		// 		cmd.ip = ip[0].String()
		// 	}

		// }
		// cmd.ip = remote[0]
		addr = remote[0] + ":" + remote[1]
		return dialer.DialContext(ctx, network, addr)
	}
	//	proxy := httputil.NewSingleHostReverseProxy(remote)
	targetQuery := remote.RawQuery

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			//_config := InitConfig()
			req.URL.Scheme = remote.Scheme
			req.URL.Host = remote.Host
			req.URL.Path, req.URL.RawPath = joinURLPath(remote, req.URL)
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
			//req.
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
				//req.Response
			}

			if _, ok := req.Header["Origin"]; ok {

				req.Header.Set("Origin", fmt.Sprintf("%s://%s:%s", CONFIG.Protocol, CONFIG.Proxy.IP, CONFIG.Proxy.Port))
				//req.Header.Set("Origin", fmt.Sprintf("https://%s:%s", CONFIG.Proxy.IP, CONFIG.Proxy.Port))

			}

			if _, ok := req.Header["Referer"]; ok {

				req.Header.Set("Referer", fmt.Sprintf("%s://%s:%s", CONFIG.Protocol, CONFIG.Proxy.IP, CONFIG.Proxy.Port)) // "https://192.168.128.242:8443/tmui/login.jsp?msgcode=1&")
				//req.Header.Set("Referer", fmt.Sprintf("https://%s:%s", CONFIG.Proxy.IP, CONFIG.Proxy.Port)) // "https://192.168.128.242:8443/tmui/login.jsp?msgcode=1&")

			}

		},
		ModifyResponse: func(res *http.Response) error {
			//fmt.Println("request", res.Request)
			//BIGIPAuthCookie=vwxqvsJS8BDEjcqxXIKRv0D2ilRSITidKZkXMP1u; path=/;HttpOnly
			cok := "Set-Cookie"

			if _, ok := res.Header[cok]; ok {

				for k, v := range res.Header[cok] {
					res.Header[cok][k] = strings.ReplaceAll(v, "Secure;", "")
				}

			}

			_, err = rbcp.Write(stdSendEvent(res))
			if err != nil {
				logger.Error(err)
			}
			return nil

		},
	}
	//proxy.Director.
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS12},
	}

	r.Host = remote.Host
	proxy.ServeHTTP(w, r)
}
