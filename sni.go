package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
)

//SNI config
type SNI struct {
	Concurrency      int `json:"concurrency"`
	Timeout          int `json:"timeout"`
	HandshakeTimeout int
	Delay            int      `json:"delay"`
	ServerName       []string `json:"server_name"`
	SortByDelay      bool     `json:"sort_by_delay"`
}

const (
	sniIPFileName     string = "sniip.txt"
	sniResultFileName string = "sniip_output.txt"
	sniJSONFileName   string = "ip.txt"
	configFileName    string = "sni.json"
	certFileName      string = "cacert.pem"
)

//custom log level
const (
	Info = iota
	Warning
	Debug
	Error
)

var config SNI
var certPool *x509.CertPool
var tlsConfig *tls.Config
var dialer net.Dialer

func init() {
	fmt.Println("initial...")
	parseConfig()
	config.HandshakeTimeout = config.Timeout
	loadCertPem()
	createFile()
}
func main() {
	flag.Set("logtostderr", "true")
	flag.Parse()

	ips := getSNIIP()
	var lastOKIP []string
	for _, ip := range getLastOkIP() {
		lastOKIP = append(lastOKIP, ip.Address)
	}
	ips = append(lastOKIP, ips...)
	err := os.Truncate(sniResultFileName, 0)
	checkErr(fmt.Sprintf("truncate file %s error: ", sniResultFileName), err, Error)
	jobs := make(chan string, config.Concurrency)
	done := make(chan bool, config.Concurrency)

	//check all sni ip begin
	t0 := time.Now()
	go func() {
		for _, ip := range ips {
			jobs <- ip
		}
		close(jobs)
	}()
	for ip := range jobs {
		done <- true
		go checkIP(ip, done)
	}
	for i := 0; i < cap(done); i++ {
		done <- true
	}
	t1 := time.Now()
	cost := int(t1.Sub(t0).Seconds())
	var rawiplist, jsoniplist []string
	okIPs := getLastOkIP()
	if config.SortByDelay {
		sort.Sort(ByDelay{IPs(okIPs)})
	}
	for _, ip := range okIPs {
		rawiplist = append(rawiplist, fmt.Sprintf("%s %dms", ip.Address, ip.Delay))
		if ip.Delay <= config.Delay {
			jsoniplist = append(jsoniplist, ip.Address)
		}
	}

	ipstr := strings.Join(rawiplist, "\n")
	writeIP2File(ipstr, sniResultFileName)
	jsonip := strings.Join(jsoniplist, "|")
	jsonip += "\n\n\n"
	jsonip += `"`
	jsonip += strings.Join(jsoniplist, `","`)
	jsonip += `"`
	writeIP2File(jsonip, sniJSONFileName)

	fmt.Printf("\ntime: %ds, ok ip count: %d, matched ip with delay(%dms) count: %d\n\n", cost, len(rawiplist), config.Delay, len(jsoniplist))
	fmt.Scanln()
}

//Load cacert.pem
func loadCertPem() {
	certpem, err := ioutil.ReadFile(certFileName)
	checkErr(fmt.Sprintf("read pem file %s error: ", certFileName), err, Error)
	certPool = x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certpem) {
		checkErr(fmt.Sprintf("load pem file %s error: ", certFileName), errors.New("load pem file error"), Error)
	}
}
func checkIP(ip string, done chan bool) {
	defer func() {
		<-done
	}()
	delays := make([]int, len(config.ServerName))
	dialer = net.Dialer{
		Timeout:   time.Millisecond * time.Duration(config.Timeout),
		KeepAlive: 0,
		DualStack: false,
	}
Next:
	for i, server := range config.ServerName {
		conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, "443"))
		if err != nil {
			checkErr(fmt.Sprintf("%s dial error: ", ip), err, Debug)
			return
		}
		defer conn.Close()

		tlsConfig = &tls.Config{
			RootCAs:            certPool,
			InsecureSkipVerify: false,
			ServerName:         server,
		}

		t0 := time.Now()
		tlsClient := tls.Client(conn, tlsConfig)
		tlsClient.SetDeadline(time.Now().Add(time.Millisecond * time.Duration(config.HandshakeTimeout)))
		err = tlsClient.Handshake()

		if err != nil {
			checkErr(fmt.Sprintf("%s handshake error: ", ip), err, Debug)
			return
		}
		defer tlsClient.Close()
		t1 := time.Now()
		delays[i] = int(t1.Sub(t0).Seconds() * 1000)
		if tlsClient.ConnectionState().PeerCertificates == nil {
			checkErr(fmt.Sprintf("%s peer certificates error: ", ip), errors.New("peer certificates is nil"), Debug)
			return
		}

		//peerCertSubject := tlsClient.ConnectionState().PeerCertificates[0].Subject
		DNSNames := tlsClient.ConnectionState().PeerCertificates[0].DNSNames
		//commonName := peerCertSubject.CommonName
		for _, DNSName := range DNSNames {
			if DNSName == server || DNSName == strings.Replace(server, "www", "*", -1) {
				continue Next
			}
		}
		return
	}
	sum := 0
	for _, d := range delays {
		sum += d
	}
	delay := sum / len(delays)
	checkErr(fmt.Sprintf("%s %dms, sni ip, recorded.", ip, delay), errors.New(""), Info)

	appendIP2File(IP{Address: ip, Delay: delay}, sniResultFileName)
}

//Parse config file
func parseConfig() {
	conf, err := ioutil.ReadFile(configFileName)
	checkErr("read config file error: ", err, Error)
	err = json.Unmarshal(conf, &config)
	checkErr("parse config file error: ", err, Error)
}

//Create files if they donnot exist, or truncate them.
func createFile() {
	if !isFileExist(sniResultFileName) {
		_, err := os.Create(sniResultFileName)
		checkErr(fmt.Sprintf("create file %s error: ", sniResultFileName), err, Error)
	}
	if !isFileExist(sniJSONFileName) {
		_, err := os.Create(sniJSONFileName)
		checkErr(fmt.Sprintf("create file %s error: ", sniJSONFileName), err, Error)
	}
}

//CheckErr checks given error
func checkErr(messge string, err error, level int) {
	if err != nil {
		switch level {
		case Info, Warning, Debug:
			glog.Infoln(messge, err)
		case Error:
			glog.Fatalln(messge, err)
		}
	}
}

//Whether file exists.
func isFileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

//append ip to related file
func appendIP2File(ip IP, filename string) {
	f, err := os.OpenFile(filename, os.O_APPEND, os.ModeAppend)
	checkErr(fmt.Sprintf("open file %s error: ", filename), err, Error)
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("%s %dms\n", ip.Address, ip.Delay))
	checkErr(fmt.Sprintf("append ip to file %s error: ", filename), err, Error)
	f.Close()
}

//write ip to related file
func writeIP2File(ips string, filename string) {
	err := os.Truncate(filename, 0)
	checkErr(fmt.Sprintf("truncate file %s error: ", filename), err, Error)
	err = ioutil.WriteFile(filename, []byte(ips), 0755)
	checkErr(fmt.Sprintf("write ip to file %s error: ", filename), err, Error)
}
