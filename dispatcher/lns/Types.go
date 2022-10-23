// This code is transcompiled by LuneScript.
package lns
import . "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
var init_Types bool
var Types__mod__ string
type Types_CreateHandlerFunc func (_env *LnsEnv) Types_HandleIF
// declaration Class -- ReqTunnelInfo
type Types_ReqTunnelInfoMtd interface {
    ToMap() *LnsMap
    Get_host(_env *LnsEnv) string
    Get_mode(_env *LnsEnv) string
    Get_port(_env *LnsEnv) LnsInt
    Get_tunnelArgList(_env *LnsEnv) *LnsList
}
type Types_ReqTunnelInfo struct {
    host string
    port LnsInt
    mode string
    tunnelArgList *LnsList
    FP Types_ReqTunnelInfoMtd
}
func Types_ReqTunnelInfo2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*Types_ReqTunnelInfo).FP
}
type Types_ReqTunnelInfoDownCast interface {
    ToTypes_ReqTunnelInfo() *Types_ReqTunnelInfo
}
func Types_ReqTunnelInfoDownCastF( multi ...LnsAny ) LnsAny {
    if len( multi ) == 0 { return nil }
    obj := multi[ 0 ]
    if ddd, ok := multi[ 0 ].([]LnsAny); ok { obj = ddd[0] }
    work, ok := obj.(Types_ReqTunnelInfoDownCast)
    if ok { return work.ToTypes_ReqTunnelInfo() }
    return nil
}
func (obj *Types_ReqTunnelInfo) ToTypes_ReqTunnelInfo() *Types_ReqTunnelInfo {
    return obj
}
func NewTypes_ReqTunnelInfo(_env *LnsEnv, arg1 string, arg2 LnsInt, arg3 string, arg4 *LnsList) *Types_ReqTunnelInfo {
    obj := &Types_ReqTunnelInfo{}
    obj.FP = obj
    obj.InitTypes_ReqTunnelInfo(_env, arg1, arg2, arg3, arg4)
    return obj
}
func (self *Types_ReqTunnelInfo) InitTypes_ReqTunnelInfo(_env *LnsEnv, arg1 string, arg2 LnsInt, arg3 string, arg4 *LnsList) {
    self.host = arg1
    self.port = arg2
    self.mode = arg3
    self.tunnelArgList = arg4
}
func (self *Types_ReqTunnelInfo) Get_host(_env *LnsEnv) string{ return self.host }
func (self *Types_ReqTunnelInfo) Get_port(_env *LnsEnv) LnsInt{ return self.port }
func (self *Types_ReqTunnelInfo) Get_mode(_env *LnsEnv) string{ return self.mode }
func (self *Types_ReqTunnelInfo) Get_tunnelArgList(_env *LnsEnv) *LnsList{ return self.tunnelArgList }
func (self *Types_ReqTunnelInfo) ToMapSetup( obj *LnsMap ) *LnsMap {
    obj.Items["host"] = Lns_ToCollection( self.host )
    obj.Items["port"] = Lns_ToCollection( self.port )
    obj.Items["mode"] = Lns_ToCollection( self.mode )
    obj.Items["tunnelArgList"] = Lns_ToCollection( self.tunnelArgList )
    return obj
}
func (self *Types_ReqTunnelInfo) ToMap() *LnsMap {
    return self.ToMapSetup( NewLnsMap( map[LnsAny]LnsAny{} ) )
}
func Types_ReqTunnelInfo__fromMap(_env,  arg1 LnsAny, paramList []Lns_ToObjParam)(LnsAny, LnsAny){
   return Types_ReqTunnelInfo_FromMap( arg1, paramList )
}
func Types_ReqTunnelInfo__fromStem(_env,  arg1 LnsAny, paramList []Lns_ToObjParam)(LnsAny, LnsAny){
   return Types_ReqTunnelInfo_FromMap( arg1, paramList )
}
func Types_ReqTunnelInfo_FromMap( obj LnsAny, paramList []Lns_ToObjParam ) (LnsAny, LnsAny) {
    _,conv,mess := Types_ReqTunnelInfo_FromMapSub(obj,false, paramList);
    return conv,mess
}
func Types_ReqTunnelInfo_FromMapSub( obj LnsAny, nilable bool, paramList []Lns_ToObjParam ) (bool, LnsAny, LnsAny) {
    var objMap *LnsMap
    if work, ok := obj.(*LnsMap); !ok {
       return false, nil, "no map -- " + Lns_ToString(obj)
    } else {
       objMap = work
    }
    newObj := &Types_ReqTunnelInfo{}
    newObj.FP = newObj
    return Types_ReqTunnelInfo_FromMapMain( newObj, objMap, paramList )
}
func Types_ReqTunnelInfo_FromMapMain( newObj *Types_ReqTunnelInfo, objMap *LnsMap, paramList []Lns_ToObjParam ) (bool, LnsAny, LnsAny) {
    if ok,conv,mess := Lns_ToStrSub( objMap.Items["host"], false, nil); !ok {
       return false,nil,"host:" + mess.(string)
    } else {
       newObj.host = conv.(string)
    }
    if ok,conv,mess := Lns_ToIntSub( objMap.Items["port"], false, nil); !ok {
       return false,nil,"port:" + mess.(string)
    } else {
       newObj.port = conv.(LnsInt)
    }
    if ok,conv,mess := Lns_ToStrSub( objMap.Items["mode"], false, nil); !ok {
       return false,nil,"mode:" + mess.(string)
    } else {
       newObj.mode = conv.(string)
    }
    if ok,conv,mess := Lns_ToListSub( objMap.Items["tunnelArgList"], false, []Lns_ToObjParam{Lns_ToObjParam{
            Lns_ToStrSub, false,nil}}); !ok {
       return false,nil,"tunnelArgList:" + mess.(string)
    } else {
       newObj.tunnelArgList = conv.(*LnsList)
    }
    return true, newObj, nil
}

type Types_HandleIF interface {
        GetTunnelInfo(_env *LnsEnv, arg1 string, arg2 *LnsMap) *Types_ReqTunnelInfo
        OnEndTunnel(_env *LnsEnv, arg1 *Types_ReqTunnelInfo)
}
func Lns_cast2Types_HandleIF( obj LnsAny ) LnsAny {
    if _, ok := obj.(Types_HandleIF); ok { 
        return obj
    }
    return nil
}

type Types_AsyncHandleIF interface {
        CanAccept(_env *LnsEnv, arg1 string, arg2 *LnsMap)(LnsInt, string)
}
func Lns_cast2Types_AsyncHandleIF( obj LnsAny ) LnsAny {
    if _, ok := obj.(Types_AsyncHandleIF); ok { 
        return obj
    }
    return nil
}

func Lns_Types_init(_env *LnsEnv) {
    if init_Types { return }
    init_Types = true
    Types__mod__ = "@lns.@Types"
    Lns_InitMod()
}
func init() {
    init_Types = false
}
