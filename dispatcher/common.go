// -*- coding: utf-8 -*-
package main

import (
	"bytes"
	"container/ring"
	"crypto/sha256"
	"fmt"
	"io"
	"sync"
	//"net"
)

// 接続先情報
type HostInfo struct {
	// スキーム。 http:// など
	Scheme string
	// ホスト名
	Name string
	// ポート番号
	Port int
	// パス
	Path string
}

// 接続先の文字列表現
func (info *HostInfo) toStr() string {
	return fmt.Sprintf("%s%s:%d%s", info.Scheme, info.Name, info.Port, info.Path)
}

// パスワードからキーを生成する
func getKey(pass []byte) []byte {
	sum := sha256.Sum256(pass)
	return sum[:]
}

func init() {
}

type Lock struct {
	mutex sync.Mutex
	owner string
}

func (lock *Lock) get(name string) {
	lock.mutex.Lock()
	lock.owner = name
}

func (lock *Lock) rel() {
	lock.owner = ""
	lock.mutex.Unlock()
}

// tunnel の制御パラメータ
type TunnelParam struct {
	// セッションのモード
	Mode string
	// 接続可能な IP パターン。
	// nil の場合、 IP 制限しない。
	maskedIP *MaskIP
	// サーバ情報
	serverInfo HostInfo
}

type RingBuf struct {
	ring *ring.Ring
}

func NewRingBuf(num, bufsize int) *RingBuf {
	ring := ring.New(num)
	for index := 0; index < num; index++ {
		ring.Value = make([]byte, bufsize)
		ring = ring.Next()
	}
	return &RingBuf{ring}
}

func (ringBuf *RingBuf) getNext() []byte {
	buf := ringBuf.ring.Value.([]byte)
	ringBuf.ring = ringBuf.ring.Next()
	return buf
}

func (ringBuf *RingBuf) getCur() []byte {
	return ringBuf.ring.Value.([]byte)
}

// コネクション情報
type ConnInfo struct {
	// コネクション
	Conn        io.ReadWriteCloser
	writeBuffer bytes.Buffer
}

// ConnInfo の生成
//
// @param conn コネクション
// @param pass 暗号化パスワード
// @param count 暗号化回数
// @param sessionInfo セッション情報
// @return ConnInfo
func CreateConnInfo(conn io.ReadWriteCloser) *ConnInfo {
	return &ConnInfo{conn, bytes.Buffer{}}
}
