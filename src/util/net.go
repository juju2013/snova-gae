package util

import (
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
)

func Long2IPv4(i int64) string {
	return fmt.Sprintf("%d.%d.%d.%d", (i>>24)&0xFF, (i>>16)&0xFF, (i>>8)&0xFF, i&0xFF)
}

func IPv42Int(ip string) (int64, error) {
	addrArray := strings.Split(ip, ".")
	var num int64
	num = 0
	for i := 0; i < len(addrArray); i++ {
		power := 3 - i
		if v, err := strconv.Atoi(addrArray[i]); nil != err {
			return -1, err
		} else {
			num += (int64(v) % 256 * int64(math.Pow(float64(256), float64(power))))
		}
	}
	return num, nil
}

func IsPrivateIP(ip string) bool {
	if strings.EqualFold(ip, "localhost") {
		return true
	}
	value, err := IPv42Int(ip)
	if nil != err {
		return false
	}
	if strings.HasPrefix(ip, "127.0") {
		return true
	}
	if (value >= 0x0A000000 && value <= 0x0AFFFFFF) || (value >= 0xAC100000 && value <= 0xAC1FFFFF) || (value >= 0xC0A80000 && value <= 0xC0A8FFFF) {
		return true
	}
	return false
}

func GetLocalIP() string {
	hostname, err := os.Hostname()
	if nil != err {
		return "127.0.0.1"
	}
	ipp, err := net.LookupHost(hostname)
	if nil != err {
		return "127.0.0.1"
	}
	return ipp[0]
}

func ParseRangeHeaderValue(value string) (startPos, endPos int) {
	vs := strings.Split(value, "=")
	vs = strings.Split(vs[1], "-")
	startPos, _ = strconv.Atoi(vs[0])
	if tmp, err := strconv.Atoi(vs[1]); nil != err {
		endPos = -1
	} else {
		endPos = tmp
	}
	return
}

//func qhandler(m, r *dns.Msg, e error, data interface{}) {
//	ips := make([]string, 0)
//	if r != nil && r.Rcode == dns.RcodeSuccess {
//		for _, aa := range r.Answer {
//			switch aa.(type) {
//			case *dns.RR_A:
//				ips = append(ips, aa.(*dns.RR_A).A.String()+"")
//			case *dns.RR_AAAA:
//				ips = append(ips, "["+aa.(*dns.RR_AAAA).AAAA.String()+"]")
//			}
//		}
//		data.(chan []string) <- ips
//		return
//	}
//	data.(chan []string) <- nil
//}
//
//func addresses(conf *dns.ClientConfig, c *dns.Client, name string, dnsserver string, onlyIPv4 bool) []string {
//	m4 := new(dns.Msg)
//	m4.SetQuestion(dns.Fqdn(name), dns.TypeA)
//	m6 := new(dns.Msg)
//	m6.SetQuestion(dns.Fqdn(name), dns.TypeAAAA)
//
//	addr := make(chan []string)
//	defer close(addr)
//
//	var ips []string
//	i := 1 // two outstanding queries
//	c.Do(m4, dnsserver+":"+conf.Port, addr, qhandler)
//	if !onlyIPv4 {
//		c.Do(m6, dnsserver+":"+conf.Port, addr, qhandler)
//		i = 2
//	}
//
//forever:
//	for {
//		select {
//		case ip := <-addr:
//			ips = append(ips, ip...)
//			i--
//			if i == 0 {
//				break forever
//			}
//		}
//	}
//	return ips
//}
//
//func DnsTCPLookup(dnsserver []string, domain string, onlyIPv4 bool) ([]string, error) {
//	conf := new(dns.ClientConfig)
//	conf.Servers = dnsserver // small, but the standard limit
//	conf.Search = make([]string, 0)
//	conf.Port = "53"
//	conf.Ndots = 1
//	conf.Timeout = 5
//	conf.Attempts = 2
//	m := new(dns.Msg)
//	m.Question = make([]dns.Question, 1)
//	c := new(dns.Client)
//	c.Net = "tcp"
//
//	for _, server := range dnsserver {
//		addr := addresses(conf, c, domain, server, onlyIPv4)
//		if len(addr) > 0 {
//			return addr, nil
//		}
//	}
//	return nil, errors.New("No DNS result found")
//}