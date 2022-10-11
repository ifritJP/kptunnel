// This code is transcompiled by LuneScript.
package lns
import . "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
var init_handle bool
var handle__mod__ string
var handle_port LnsInt
// 11: decl @lns.@handle.canAcceptRequest
func Handle_canAcceptRequest(_env *LnsEnv, uri string,headerMap *LnsMap) *Handle_ReqTunnelInfo {
    handle_port = handle_port + 1
    return NewHandle_ReqTunnelInfo(_env, 200, "", "localhost", handle_port, NewLnsList([]LnsAny{"../kptunnel", "r-server", _env.GetVM().String_format(":%d", []LnsAny{handle_port}), ":20000,:22"}))
}

// 19: decl @lns.@handle.onEndTunnel
func Handle_onEndTunnel(_env *LnsEnv, tunnelInfo *Handle_ReqTunnelInfo) {
}

// declaration Class -- ReqTunnelInfo
type Handle_ReqTunnelInfoMtd interface {
    Get_host(_env *LnsEnv) string
    Get_message(_env *LnsEnv) string
    Get_port(_env *LnsEnv) LnsInt
    Get_statusCode(_env *LnsEnv) LnsInt
    Get_tunnelArgList(_env *LnsEnv) *LnsList
}
type Handle_ReqTunnelInfo struct {
    statusCode LnsInt
    message string
    host string
    port LnsInt
    tunnelArgList *LnsList
    FP Handle_ReqTunnelInfoMtd
}
func Handle_ReqTunnelInfo2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*Handle_ReqTunnelInfo).FP
}
type Handle_ReqTunnelInfoDownCast interface {
    ToHandle_ReqTunnelInfo() *Handle_ReqTunnelInfo
}
func Handle_ReqTunnelInfoDownCastF( multi ...LnsAny ) LnsAny {
    if len( multi ) == 0 { return nil }
    obj := multi[ 0 ]
    if ddd, ok := multi[ 0 ].([]LnsAny); ok { obj = ddd[0] }
    work, ok := obj.(Handle_ReqTunnelInfoDownCast)
    if ok { return work.ToHandle_ReqTunnelInfo() }
    return nil
}
func (obj *Handle_ReqTunnelInfo) ToHandle_ReqTunnelInfo() *Handle_ReqTunnelInfo {
    return obj
}
func NewHandle_ReqTunnelInfo(_env *LnsEnv, arg1 LnsInt, arg2 string, arg3 string, arg4 LnsInt, arg5 *LnsList) *Handle_ReqTunnelInfo {
    obj := &Handle_ReqTunnelInfo{}
    obj.FP = obj
    obj.InitHandle_ReqTunnelInfo(_env, arg1, arg2, arg3, arg4, arg5)
    return obj
}
func (self *Handle_ReqTunnelInfo) InitHandle_ReqTunnelInfo(_env *LnsEnv, arg1 LnsInt, arg2 string, arg3 string, arg4 LnsInt, arg5 *LnsList) {
    self.statusCode = arg1
    self.message = arg2
    self.host = arg3
    self.port = arg4
    self.tunnelArgList = arg5
}
func (self *Handle_ReqTunnelInfo) Get_statusCode(_env *LnsEnv) LnsInt{ return self.statusCode }
func (self *Handle_ReqTunnelInfo) Get_message(_env *LnsEnv) string{ return self.message }
func (self *Handle_ReqTunnelInfo) Get_host(_env *LnsEnv) string{ return self.host }
func (self *Handle_ReqTunnelInfo) Get_port(_env *LnsEnv) LnsInt{ return self.port }
func (self *Handle_ReqTunnelInfo) Get_tunnelArgList(_env *LnsEnv) *LnsList{ return self.tunnelArgList }

func Lns_handle_init(_env *LnsEnv) {
    if init_handle { return }
    init_handle = true
    handle__mod__ = "@lns.@handle"
    Lns_InitMod()
    handle_port = 10000
}
func init() {
    init_handle = false
}
