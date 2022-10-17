# httpsProxy
https代理  支持

# 编译成可执行文件

- linu

```shell

set GOOS = liunx
go build
```

- Windows

```shell

set GOOS = Windows
go build

```

# 配置yaml文件

```yaml

Location:
 IP: 192.168.72.2 #需要被代理的IP
 Prot: 80 #需要被代理的端口
Proxy:
 IP: 127.0.0.1 #代理的IP
 Proxy: 8080 #代理的端口
Protocol: https #协议http/https

```

# 参数

``` shell
chmod +x ./httpsProxy

./httpsProxy
Usage of ./httpsProxy
-i install the server
-r run the server
-u uninstall the server


```

其中-i注册为服务后会自动启动服务

