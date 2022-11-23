// -*- coding: utf-8 -*-
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"unsafe"

	//"net"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"

	"crypto/aes"
	"crypto/cipher"
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
	// query
	Query string
}

// 接続先の文字列表現
func (info *HostInfo) toStr() string {
	work := fmt.Sprintf("%s%s:%d%s", info.Scheme, info.Name, info.Port, info.Path)
	if info.Query != "" {
		work = fmt.Sprintf("%s?%s", work, info.Query)
	}
	return work
}

// パスワードからキーを生成する
func getKey(pass []byte) []byte {
	sum := sha256.Sum256(pass)
	return sum[:]
}

// 暗号化モード
type CryptMode struct {
	// 暗号化を行なう最大回数。
	// -1: 無制限
	//  0: 暗号化なし
	//  N: 最大暗号化回数 N 回
	countMax int
	// 現在の暗号化回数
	count int
	// 作業用バッファ
	work []byte
	// 暗号化処理
	stream cipher.Stream
}
type CryptCtrl struct {
	enc CryptMode
	dec CryptMode
}

// 暗号用のオブジェクトを生成する
//
// @param pass パスワード
// @param count 暗回化回数
func CreateCryptCtrl(pass *string, count int) *CryptCtrl {
	if pass == nil || count == 0 {
		return nil
	}

	bufSize := BUFSIZE
	key := getKey([]byte(*pass))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	iv := make([]byte, aes.BlockSize)
	for index := 0; index < len(iv); index++ {
		iv[index] = byte(index)
	}

	encrypter := cipher.NewCFBEncrypter(block, iv)
	decrypter := cipher.NewCFBDecrypter(block, iv)

	ctrl := CryptCtrl{
		CryptMode{count, 0, make([]byte, bufSize), encrypter},
		CryptMode{count, 0, make([]byte, bufSize), decrypter}}

	return &ctrl
}

func (mode *CryptMode) IsValid() bool {
	if mode == nil || mode.countMax == 0 {
		return false
	}
	return true
}

// 暗号・複合処理
//
// @param inbuf 処理対象のデータを保持するバッファ
// @param outbuf 処理後のデータを格納するバッファ。
//    nil を指定した場合 CryptMode の work に結果を格納する。
// @return 処理後のデータを格納するバッファ。
//   outbuf に nil 以外を指定した場合、 outbuf の slice を返す。
//   outbuf に nil を指定した場合、CryptMode の work の slice を返す。
func (mode *CryptMode) Process(inbuf []byte, outbuf []byte) []byte {
	work := outbuf
	if outbuf == nil {
		work = mode.work
	}
	if len(inbuf) > len(work) {
		panic(fmt.Errorf("over length"))
	}
	if mode.countMax == 0 {
		return inbuf
	}
	if mode.countMax > 0 {
		if mode.countMax > mode.count {
			mode.count++
		} else if mode.countMax <= mode.count {
			mode.countMax = 0
			log.Print("crypto is disabled")
		}
	}
	// buf := work[:len(inbuf)]
	// mode.stream.XORKeyStream( buf, inbuf )
	// return buf

	mode.stream.XORKeyStream(work, inbuf)

	return work[:len(inbuf)]
}

// 暗号化
func (ctrl *CryptCtrl) Encrypt(bytes []byte) []byte {
	return ctrl.enc.Process(bytes, nil)
}

// 複合化
func (ctrl *CryptCtrl) Decrypt(bytes []byte) []byte {
	return ctrl.dec.Process(bytes, nil)
}

// 通常パッケット
const PACKET_KIND_NORMAL = 0

// 無通信を避けるためのダミーパケット
const PACKET_KIND_DUMMY = 1

// packetWriter() の処理終了を通知するためのパケット
const PACKET_KIND_EOS = 2

// Tunnel の通信を同期するためのパケット
const PACKET_KIND_SYNC = 3
const PACKET_KIND_NORMAL_DIRECT = 4

var dummyKindBuf = []byte{PACKET_KIND_DUMMY}
var normalKindBuf = []byte{PACKET_KIND_NORMAL}
var syncKindBuf = []byte{PACKET_KIND_SYNC}

var PACKET_LEN_HEADER int = 0

func init() {
	var citiId uint32
	PACKET_LEN_HEADER = len(normalKindBuf) + int(unsafe.Sizeof(citiId))
}

func WriteDummy(ostream io.Writer) error {
	if _, err := ostream.Write(dummyKindBuf); err != nil {
		return err
	}
	return nil
}

func WriteSimpleKind(ostream io.Writer, kind int8, citiId uint32, buf []byte) error {

	var kindbuf []byte
	switch kind {
	case PACKET_KIND_SYNC:
		kindbuf = syncKindBuf
	default:
		log.Fatal("illegal kind -- ", kind)
	}

	var buffer bytes.Buffer
	buffer.Grow(PACKET_LEN_HEADER + len(buf))

	if _, err := buffer.Write(kindbuf); err != nil {
		return err
	}
	if err := binary.Write(&buffer, binary.BigEndian, citiId); err != nil {
		return err
	}
	if _, err := buffer.Write(buf); err != nil {
		return err
	}

	_, err := buffer.WriteTo(ostream)
	return err
}

// データを出力する
//
// ostream 出力先
// buf データ
// ctrl 暗号化情報
func WriteItem(
	ostream io.Writer, citiId uint32,
	buf []byte, ctrl *CryptCtrl, workBuf *bytes.Buffer) error {
	// write のコール数が多いと通信効率が悪いので
	// 一旦バッファに書き込んでから ostream に出力する。
	var buffer *bytes.Buffer = workBuf
	if buffer == nil {
		buffer = &bytes.Buffer{}
	} else {
		buffer.Reset()
	}
	size := uint16(len(buf))
	buffer.Grow(
		len(normalKindBuf) + int(unsafe.Sizeof(citiId)) +
			int(unsafe.Sizeof(size)) + len(buf))

	if err := WriteItemDirect(buffer, citiId, buf, ctrl); err != nil {
		return err
	}

	_, err := buffer.WriteTo(ostream)
	return err
}

// データを出力する
//
// ostream 出力先
// buf データ
// ctrl 暗号化情報
func WriteItemDirect(ostream io.Writer, citiId uint32, buf []byte, ctrl *CryptCtrl) error {
	if _, err := ostream.Write(normalKindBuf); err != nil {
		return err
	}
	if err := binary.Write(ostream, binary.BigEndian, citiId); err != nil {
		return err
	}
	if ctrl != nil {
		buf = ctrl.enc.Process(buf, nil)
	}
	if err := binary.Write(ostream, binary.BigEndian, uint16(len(buf))); err != nil {
		return err
	}
	_, err := ostream.Write(buf)
	return err
}

type PackItem struct {
	citiId uint32
	buf    []byte
	kind   int8
}

func ReadCitiId(istream io.Reader) (uint32, error) {
	buf := make([]byte, 4)
	_, error := io.ReadFull(istream, buf)
	if error != nil {
		return 0, error
	}
	return binary.BigEndian.Uint32(buf), nil
}

func ReadPackNo(istream io.Reader, kind int8) (*PackItem, error) {
	var item PackItem
	item.kind = kind
	var error error
	if item.citiId, error = ReadCitiId(istream); error != nil {
		return nil, error
	}
	var packNo int64
	item.buf = make([]byte, unsafe.Sizeof(packNo))
	_, err := io.ReadFull(istream, item.buf)
	if err != nil {
		return &item, err
	}
	return &item, nil
}

type CitiBuf interface {
	// citiId 向けのバッファを取得する
	GetPacketBuf(citiId uint32, packSize uint16) []byte
}

type HeapCitiBuf struct {
}

var heapCitiBuf *HeapCitiBuf = &HeapCitiBuf{}

func (citiBuf *HeapCitiBuf) GetPacketBuf(citiId uint32, packSize uint16) []byte {
	return make([]byte, packSize)
}

// データを読み込む
//
// @param istream 読み込み元ストリーム
// @param ctrl 暗号化制御
// @param workBuf
func ReadItem(
	istream io.Reader, ctrl *CryptCtrl,
	workBuf []byte, citiBuf CitiBuf) (*PackItem, error) {

	var item PackItem

	var kindbuf []byte
	if workBuf != nil {
		kindbuf = workBuf[:1]
	} else {
		kindbuf = make([]byte, 1)
	}
	_, error := io.ReadFull(istream, kindbuf)
	if error != nil {
		return nil, error
	}
	switch item.kind = int8(kindbuf[0]); item.kind {
	case PACKET_KIND_DUMMY:
		return &item, nil
	case PACKET_KIND_SYNC:
		return ReadPackNo(istream, item.kind)
	case PACKET_KIND_NORMAL:
		if item.citiId, error = ReadCitiId(istream); error != nil {
			return nil, error
		}
		var buf []byte
		if workBuf != nil {
			buf = workBuf[:2]
		} else {
			buf = make([]byte, 2)
		}

		//buf := make([]byte,2)
		_, error := io.ReadFull(istream, buf)
		if error != nil {
			return nil, error
		}
		packSize := binary.BigEndian.Uint16(buf)
		var packBuf []byte
		var citiPackBuf []byte = nil
		if workBuf == nil {
			packBuf = make([]byte, packSize)
		} else {
			if len(workBuf) < int(packSize) {
				log.Fatal("workbuf size is short -- ", len(workBuf))
			}
			citiPackBuf = citiBuf.GetPacketBuf(item.citiId, packSize)
			if ctrl == nil || !ctrl.dec.IsValid() {
				// 暗号化無しなら packBuf に citiPackBuf を直接入れる
				packBuf = citiPackBuf
			} else {
				// 暗号化ありなら packBuf に workBuf を設定して、
				// 暗号化後のバッファを citiPackBuf に設定する
				packBuf = workBuf[:packSize]
			}
		}
		_, error = io.ReadFull(istream, packBuf)
		if error != nil {
			return nil, error
		}
		if ctrl != nil {
			packBuf = ctrl.dec.Process(packBuf, citiPackBuf)
		}
		item.buf = packBuf
		return &item, nil
	default:
		return nil, fmt.Errorf("ReadItem illegal kind -- %d", item.kind)
	}
}

// データを読み込む
func readItemForNormal(istream io.Reader, ctrl *CryptCtrl) (*PackItem, error) {
	item, err := ReadItem(istream, ctrl, nil, heapCitiBuf)
	if err != nil {
		return nil, err
	}
	if item.kind != PACKET_KIND_NORMAL {
		return nil, fmt.Errorf("readItemForNormal illegal kind -- %d", item.kind)
	}
	return item, nil
}

// データを読み込む
func readItemWithReader(istream io.Reader, ctrl *CryptCtrl) (io.Reader, error) {
	item, err := readItemForNormal(istream, ctrl)
	if err != nil {
		return nil, err
	}
	if item.citiId != CITIID_CTRL {
		return nil, fmt.Errorf("citiid != 0 -- %d", item.citiId)
	}
	return bytes.NewReader(item.buf), nil
}

// server -> client
type AuthChallenge struct {
	Ver       string
	Challenge string
	Mode      string
}

const BENCH_LOOP_COUNT = 200

const CTRL_NONE = 0
const CTRL_BENCH = 1
const CTRL_STOP = 2

// client -> server
type AuthResponse struct {
	//
	Response     string
	Hint         string
	SessionToken string
	WriteNo      int64
	ReadNo       int64
	Ctrl         int
	ForwardList  []ForwardInfo
}

// server -> client
type AuthResult struct {
	Result       string
	SessionId    int
	SessionToken string
	WriteNo      int64
	ReadNo       int64
	ForwardList  []ForwardInfo
}

func generateChallengeResponse(challenge string, pass *string, hint string) string {
	sum := sha512.Sum512([]byte(challenge + *pass + hint))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// サーバ側のネゴシエーション処理
//
// 接続しに来たクライアントの認証を行なう。
//
// @param connInfo 接続コネクション情報
// @param param Tunnel情報
// @param remoteAddr 接続元のアドレス
// @return bool 新しい session の場合 true
// @return []ForwardInfo 接続する ForwardInfo リスト
// @return error
func ProcessServerAuth(
	connInfo *ConnInfo, param *TunnelParam,
	remoteAddr string, forwardList []ForwardInfo) (bool, []ForwardInfo, error) {

	stream := connInfo.Conn
	log.Print("start auth")

	if err := CorrectLackOffsetWrite(stream); err != nil {
		return false, nil, err
	}
	if err := CorrectLackOffsetRead(stream); err != nil {
		return false, nil, err
	}

	// 共通文字列を暗号化して送信することで、
	// 接続先の暗号パスワードが一致しているかチェック出来るようにデータ送信
	WriteItem(stream, CITIID_CTRL, []byte(param.magic), connInfo.CryptCtrlObj, nil)

	// challenge 文字列生成
	nano := time.Now().UnixNano()
	sum := sha256.Sum256([]byte(fmt.Sprint("%v", nano)))
	str := base64.StdEncoding.EncodeToString(sum[:])
	challenge := AuthChallenge{"1.00", str, param.Mode}

	bytes, _ := json.Marshal(challenge)
	if err := WriteItem(
		stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil); err != nil {
		return false, nil, err
	}
	log.Print("challenge ", challenge.Challenge)
	connInfo.SessionInfo.SetState(Session_state_authchallenge)

	// challenge-response 処理
	reader, err := readItemWithReader(stream, connInfo.CryptCtrlObj)
	if err != nil {
		return false, nil, err
	}
	var resp AuthResponse
	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		return false, nil, err
	}
	if resp.Response != generateChallengeResponse(
		challenge.Challenge, param.pass, resp.Hint) {
		// challenge-response が不一致なので、認証失敗
		bytes, _ := json.Marshal(AuthResult{"ng", 0, "", 0, 0, nil})
		if err := WriteItem(
			stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil); err != nil {
			return false, nil, err
		}
		log.Print("mismatch password")
		return false, nil, fmt.Errorf("mismatch password")
	}

	// ここまででクライアントの認証が成功したので、
	// これ以降はクライアントが通知してきた情報を受けいれて OK

	// クライアントが送ってきた sessionId を取り入れる
	sessionToken := resp.SessionToken
	newSession := false
	if sessionToken == "" {
		// sessionId が "" なら、新規セッション
		connInfo.SessionInfo = NewSessionInfo(true)
		newSession = true
	} else {
		if sessionInfo, has := GetSessionInfo(sessionToken); !has {
			mess := fmt.Sprintf("not found session -- %d", sessionToken)
			bytes, _ := json.Marshal(AuthResult{"ng: " + mess, 0, "", 0, 0, nil})
			if err := WriteItem(
				stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil); err != nil {
				return false, nil, err
			}
			return false, nil, fmt.Errorf(mess)
		} else {
			connInfo.SessionInfo = sessionInfo
			WaitPauseSession(connInfo.SessionInfo)
		}
	}
	log.Printf(
		"sessionId: %s, ReadNo: %d(%d), WriteNo: %d(%d)",
		sessionToken, connInfo.SessionInfo.ReadNo, resp.WriteNo,
		connInfo.SessionInfo.WriteNo, resp.ReadNo)

	// AuthResult を返す
	bytes, _ = json.Marshal(
		AuthResult{
			"ok", connInfo.SessionInfo.SessionId, connInfo.SessionInfo.SessionToken,
			connInfo.SessionInfo.WriteNo, connInfo.SessionInfo.ReadNo, forwardList})
	log.Printf("sent forwardList -- %v", forwardList)

	if len(forwardList) == 0 {
		forwardList = resp.ForwardList
		log.Printf("receive forwardList -- %v", resp.ForwardList)
	}

	if err := WriteItem(
		stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil); err != nil {
		return false, nil, err
	}
	log.Print("match password")
	connInfo.SessionInfo.SetState(Session_state_authresult)

	// データ再送のための設定
	connInfo.SessionInfo.SetReWrite(resp.ReadNo)

	if resp.Ctrl == CTRL_BENCH {
		// ベンチマーク
		benchBuf := make([]byte, 100)
		for count := 0; count < BENCH_LOOP_COUNT; count++ {
			if _, err := ReadItem(
				stream, connInfo.CryptCtrlObj, benchBuf, heapCitiBuf); err != nil {
				return false, nil, err
			}
			if err := WriteItem(
				stream, CITIID_CTRL, benchBuf, connInfo.CryptCtrlObj, nil); err != nil {
				return false, nil, err
			}
		}
		return false, nil, fmt.Errorf("benchmarck")
	}
	if resp.Ctrl == CTRL_STOP {
		log.Print("receive the stop request")
		os.Exit(0)
	}

	SetSessionConn(connInfo)
	// if !newSession {
	//     // 新規セッションでない場合、既にセッションが処理中なので、
	//     // そのセッションでコネクションが close されるのを待つ
	//     JoinUntilToCloseConn( stream )
	// }

	return newSession, forwardList, nil
}

func CorrectLackOffsetWrite(stream io.Writer) error {
	// proxy 経由の websocket だと、
	// 最初のデータが欠けることがある。
	// proxy サーバの影響か、 websocket の実装上の問題か？
	// proxy サーバの問題な気がするが。。
	// WriteItem() を使うと、データ長とデータがペアで送信されるが、
	// データが欠けることでデータ長とデータに不整合が発生し、
	// 存在しないデータ長を読みこもうとして、タイムアウトするまで戻ってこない。
	// そこで、最初のデータにどれだけズレがあるかを確認するための
	// バイト列を出力する。
	// 0x00 〜 0x09 を2回出力する。
	bytes := make([]byte, 1)
	for subIndex := 0; subIndex < 2; subIndex++ {
		for index := 0; index < 10; index++ {
			// stream の write ごとに欠けるようなので、1 バイトづつ出力する
			bytes[0] = byte(index)
			if _, err := stream.Write(bytes); err != nil {
				return err
			}
		}
	}
	return nil
}

func CorrectLackOffsetRead(stream io.Reader) error {
	// proxy 経由の websocket だと、
	// 最初のデータが正常に送信されないことがある。
	// ここで、最初のデータにどれだけズレがあるかを確認する。

	// 0x00 〜 0x09 までのバイト列が 2 回あるので、
	// 最初に 10 バイト読み込み、
	// 読み込めた値を見てズレを確認する
	buf := make([]byte, 10)
	if _, err := io.ReadFull(stream, buf); err != nil {
		return err
	}
	log.Printf("num: %x\n", buf)
	offset := int(buf[0])
	log.Printf("offset: %d\n", offset)
	if offset >= 10 {
		return fmt.Errorf("illegal num -- %d", offset)
	}

	// ズレ量に応じて残りのデータを読み込む
	if _, err := io.ReadFull(stream, buf[:10-offset]); err != nil {
		return err
	}
	log.Printf("num2: %x\n", buf)
	for index := 0; index < 10-offset; index++ {
		if int(buf[index]) != offset+index {
			return fmt.Errorf(
				"unmatch num -- offset %d: %d != %d", offset, index, buf[index])
		}
	}
	return nil
}

// サーバとのネゴシエーションを行なう
//
// クライアントの認証に必要や手続と、再接続時のセッション情報などをやり取りする
//
// @param connInfo コネクション。再接続時はセッション情報をセットしておく。
// @param param TunnelParam
// @return bool エラー時に、処理を継続するかどうか。true の場合継続する。
// @return error エラー
func ProcessClientAuth(
	connInfo *ConnInfo, param *TunnelParam,
	forwardList []ForwardInfo) ([]ForwardInfo, bool, error) {

	log.Print("start auth")

	stream := connInfo.Conn

	if err := CorrectLackOffsetRead(stream); err != nil {
		return nil, true, err
	}
	if err := CorrectLackOffsetWrite(stream); err != nil {
		return nil, true, err
	}

	magicItem, err := readItemForNormal(stream, connInfo.CryptCtrlObj)
	if err != nil {
		return nil, true, err
	}
	if !bytes.Equal(magicItem.buf, []byte(param.magic)) {
		return nil, true, fmt.Errorf("unmatch MAGIC %x", magicItem.buf)
	}

	// challenge を読み込み、認証用パスワードから response を生成する
	var reader io.Reader
	reader, err = readItemWithReader(stream, connInfo.CryptCtrlObj)
	log.Print("read challenge")
	if err != nil {
		return nil, true, err
	}
	var challenge AuthChallenge
	if err := json.NewDecoder(reader).Decode(&challenge); err != nil {
		return nil, true, err
	}
	log.Print("challenge ", challenge.Challenge)
	// サーバ側のモードを確認して、不整合がないかチェックする
	switch challenge.Mode {
	case "server":
		if param.Mode != "client" && param.Mode != "wsclient" {
			return nil, false, fmt.Errorf("unmatch mode -- %s", challenge.Mode)
		}
	case "r-server":
		if param.Mode != "r-client" && param.Mode != "r-wsclient" {
			return nil, false, fmt.Errorf("unmatch mode -- %s", challenge.Mode)
		}
	case "wsserver":
		if param.Mode != "client" && param.Mode != "wsclient" {
			return nil, false, fmt.Errorf("unmatch mode -- %s", challenge.Mode)
		}
	case "r-wsserver":
		if param.Mode != "r-client" && param.Mode != "r-wsclient" {
			return nil, false, fmt.Errorf("unmatch mode -- %s", challenge.Mode)
		}
	}

	// response を生成
	nano := time.Now().UnixNano()
	sum := sha256.Sum256([]byte(fmt.Sprint("%v", nano)))
	hint := base64.StdEncoding.EncodeToString(sum[:])
	resp := generateChallengeResponse(challenge.Challenge, param.pass, hint)
	bytes, _ := json.Marshal(
		AuthResponse{
			resp, hint, connInfo.SessionInfo.SessionToken,
			connInfo.SessionInfo.WriteNo,
			connInfo.SessionInfo.ReadNo, param.ctrl, forwardList})
	if err := WriteItem(
		stream, CITIID_CTRL, bytes, connInfo.CryptCtrlObj, nil); err != nil {
		return nil, true, err
	}
	connInfo.SessionInfo.SetState(Session_state_authresponse)

	var result AuthResult
	{
		// AuthResult を取得する
		log.Print("read auth result")
		reader, err := readItemWithReader(stream, connInfo.CryptCtrlObj)
		if err != nil {
			return nil, true, err
		}
		if err := json.NewDecoder(reader).Decode(&result); err != nil {
			return nil, true, err
		}
		if result.Result != "ok" {
			return nil, false, fmt.Errorf("failed to auth -- %s", result.Result)
		}

		log.Printf("received forwardList -- %v", result.ForwardList)
		if forwardList != nil &&
			result.ForwardList != nil && len(result.ForwardList) > 0 {
			// クライアントが指定している ForwardList と、
			// サーバ側が指定している ForwardList に違いがあるか調べて、
			// 違う場合は警告を出力する。
			orgMap := map[string]bool{}
			for _, forwardInfo := range forwardList {
				orgMap[forwardInfo.toStr()] = true
			}
			newMap := map[string]bool{}
			for _, forwardInfo := range result.ForwardList {
				newMap[forwardInfo.toStr()] = true
			}
			diff := false
			if len(orgMap) != len(newMap) {
				diff = true
			} else {
				for org, _ := range orgMap {
					if _, has := newMap[org]; !has {
						diff = true
						break
					}
				}
			}
			if diff {
				log.Printf("******* override forward *******")
				forwardList = result.ForwardList
			}
		}

		if param.ctrl == CTRL_BENCH {
			// ベンチマーク
			benchBuf := make([]byte, 100)
			prev := time.Now()
			for count := 0; count < BENCH_LOOP_COUNT; count++ {
				if err := WriteItem(
					stream, CITIID_CTRL, benchBuf, connInfo.CryptCtrlObj, nil); err != nil {
					return nil, false, err
				}
				if _, err := ReadItem(
					stream, connInfo.CryptCtrlObj, benchBuf, heapCitiBuf); err != nil {
					return nil, false, err
				}
			}
			duration := time.Now().Sub(prev)

			return nil, false, fmt.Errorf("benchmarck -- %s", duration)
		}
		if param.ctrl == CTRL_STOP {
			os.Exit(0)
		}

		if result.SessionId != connInfo.SessionInfo.SessionId {
			if connInfo.SessionInfo.SessionId == 0 {
				// 新規接続だった場合、セッション情報を更新する
				//connInfo.SessionInfo.SessionId = result.SessionId
				connInfo.SessionInfo.UpdateSessionId(
					result.SessionId, result.SessionToken)
			} else {
				return nil, false, fmt.Errorf(
					"illegal sessionId -- %d, %d",
					connInfo.SessionInfo.SessionId, result.SessionId)
			}
		}

		log.Printf(
			"sessionId: %d, ReadNo: %d(%d), WriteNo: %d(%d)",
			result.SessionId, connInfo.SessionInfo.ReadNo, result.WriteNo,
			connInfo.SessionInfo.WriteNo, result.ReadNo)
		connInfo.SessionInfo.SetReWrite(result.ReadNo)
	}

	return forwardList, true, nil
}
