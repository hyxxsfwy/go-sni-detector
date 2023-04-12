# go-sni-detector

============

## 说明

用于扫描SNI服务器，sniip_ok.txt中的延迟值为配置中指定的各server_name的延迟的平均值。

由于在初始化时读取了所有ip以便执行去重操作，所以会消耗大量的内存，对于需要扫描大量ip且机器性能不够强大的用户，请将`soft_mode`置为`true`。

请将待测试的ip段放到sniip.txt文件，支持以下ip格式：

1. 127.0.0.1
2. 127.0.0.0/24
3. 127.0.0.0-127.0.0.255
4. 127.0.0.0-127.0.0.
5. 127.0.0.-127.0.1.

## 快速使用
```
wget https://www.ipdeny.com/ipblocks/data/aggregated/cn-aggregated.zone
screen ./go-sni-detector_linux_amd64.bin --softmode --snifile cn-aggregated.zone --outputfile scan-result-gapi.txt --concurrency 50 --timeout 2000 --servername www.googleapis.com
```

## 高级用法

支持命令，优先级高于配置文件，通过指定`-r`命令可以直接将指定的参数写入到配置文件。

```
Usage: go-sni-detector [COMMAND] [VARS]

SUPPORT COMMANDS:
	-h, --help                   help message
	-a, --allhostname            lookup all hostname of ip, or lookup the first one by default
	-r, --override               override settings
	-m, --softmode               reduce memory usage

SUPPORT VARS:
	-i, --snifile<=path>                put your ip ranges into this file
	-o, --outputfile<=path>             output sni ip to this file
	-j, --jsonfile<=path>               output sni ip as json format to this file
	-c, --concurrency<=number>          concurrency
	-t, --timeout<=number>              timeout
	-ht, --handshaketimeout<=number>    handshake timeout
	-d, --delay<=number>                delay
	-s, --servername<=string>           comma-separated server names
```

## 配置说明

`"concurrency":1000` 并发线程数，可根据自己的硬件配置调整

`"delay":1200` 扫描完成后，提取所有小于等于该延迟的ip

`"server_name"` 用于测试SNI服务器的域名，以逗号分隔

`"soft_mode"` 边读取ip边扫描，适合需要扫描大量ip且内存较小的用户

## Windows 平台

针对Windows平台出了浏览器模式，目前功能正在完善中。项目使用了websocket，参见[Web Sockets浏览器兼容一览表](http://caniuse.mojijs.com/Home/Html/item/key/websockets/index.html)判断浏览器是否兼容websocket。

## Wiki

[Wiki](https://plumwine.me/go-sni-detector-usage-wiki/)

## 其它工具

扫描google ip工具：[go-checkiptools](https://github.com/johnsonz/go-checkiptools)
