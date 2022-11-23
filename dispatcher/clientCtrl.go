//-*- coding: utf-8 -*-
package main

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var controlMutex = new(sync.Mutex)

const MAX_SESSION_PER_CLIENT = 2

var pattern = regexp.MustCompile(":[ 0-9]+$")

type MaskIP struct {
	ip   net.IP
	mask net.IPMask
}

func (maskIP *MaskIP) inRange(ip net.IP) bool {
	return maskIP.ip.Equal(ip.Mask(maskIP.mask))
}

func remoteAddr2ip(remoteAddr string) net.IP {
	if loc := pattern.FindStringIndex(remoteAddr); loc != nil {
		remoteAddr = remoteAddr[:loc[0]]
	}
	return net.ParseIP(remoteAddr)
}

func ippattern2MaskIP(ipPattern string) (*MaskIP, error) {
	dIndex := strings.Index(ipPattern, "/")

	var maskLen = 0
	ipTxt := ipPattern

	if dIndex != -1 {
		ipTxt = ipPattern[:dIndex]
		var err error
		maskLen, err = strconv.Atoi(ipPattern[dIndex+1:])
		if err != nil {
			return nil, err
		}
	}
	ip := net.ParseIP(ipTxt)
	maxBit := 4 * 8
	if strings.Index(ipTxt, ":") != -1 {
		maxBit = 16 * 8
	}
	if maskLen == 0 {
		maskLen = maxBit
	}
	mask := net.CIDRMask(maskLen, maxBit)
	work := ip.Mask(mask)
	fmt.Printf("maskedIP %d %s %s\n",
		maxBit, work.String(), ipPattern)

	return &MaskIP{work, mask}, nil
}

func AcceptClient(req *http.Request, param *TunnelParam) error {
	if req.URL.Path != param.serverInfo.Path {
		return fmt.Errorf("unmatch path -- %s, %s", req.URL.Path, param.serverInfo.Path)
	}

	remoteAddr := req.RemoteAddr

	controlMutex.Lock()
	defer controlMutex.Unlock()

	remoteIP := remoteAddr2ip(remoteAddr)
	ipTxt := remoteIP.String()

	if param.maskedIP != nil {
		// 接続元のアドレスをチェックする
		if !param.maskedIP.inRange(remoteIP) {
			return fmt.Errorf("unmatch ip -- %s", ipTxt)
		}
	}

	return nil
}
