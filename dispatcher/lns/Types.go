// This code is transcompiled by LuneScript.
package lns
import . "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
var init_Types bool
var Types__mod__ string
// decl enum -- ConnectMode 
type Types_ConnectMode = string
const Types_ConnectMode__CanReconnect = "CanReconnect"
const Types_ConnectMode__Client = "Client"
const Types_ConnectMode__OneShot = "OneShot"
var Types_ConnectModeList_ = NewLnsList( []LnsAny {
  Types_ConnectMode__OneShot,
  Types_ConnectMode__CanReconnect,
  Types_ConnectMode__Client,
})
func Types_ConnectMode_get__allList(_env *LnsEnv) *LnsList{
    return Types_ConnectModeList_
}
var Types_ConnectModeMap_ = map[string]string {
  Types_ConnectMode__CanReconnect: "ConnectMode.CanReconnect",
  Types_ConnectMode__Client: "ConnectMode.Client",
  Types_ConnectMode__OneShot: "ConnectMode.OneShot",
}
func Types_ConnectMode__from(_env *LnsEnv, arg1 string) LnsAny{
    if _, ok := Types_ConnectModeMap_[arg1]; ok { return arg1 }
    return nil
}

func Types_ConnectMode_getTxt(arg1 string) string {
    return Types_ConnectModeMap_[arg1];
}
type Types_CreateHandlerFunc func (_env *LnsEnv) Types_HandleIF
// declaration Class -- ReqTunnelInfo
type Types_ReqTunnelInfoMtd interface {
    ToMap() *LnsMap
    Get_connectMode(_env *LnsEnv) string
    Get_envMap(_env *LnsEnv) *LnsMap
    Get_host(_env *LnsEnv) string
    Get_id(_env *LnsEnv) string
    Get_mode(_env *LnsEnv) string
    Get_port(_env *LnsEnv) LnsInt
    Get_tunnelArgList(_env *LnsEnv) *LnsList
}
type Types_ReqTunnelInfo struct {
    id string
    host string
    port LnsInt
    connectMode string
    mode string
    tunnelArgList *LnsList
    envMap *LnsMap
    FP Types_ReqTunnelInfoMtd
}
func Types_ReqTunnelInfo2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*Types_ReqTunnelInfo).FP
}
func Types_ReqTunnelInfo_toSlice(slice []LnsAny) []*Types_ReqTunnelInfo {
    ret := make([]*Types_ReqTunnelInfo, len(slice))
    for index, val := range slice {
        ret[index] = val.(Types_ReqTunnelInfoDownCast).ToTypes_ReqTunnelInfo()
    }
    return ret
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
func NewTypes_ReqTunnelInfo(_env *LnsEnv, arg1 string, arg2 string, arg3 LnsInt, arg4 string, arg5 string, arg6 *LnsList, arg7 *LnsMap) *Types_ReqTunnelInfo {
    obj := &Types_ReqTunnelInfo{}
    obj.FP = obj
    obj.InitTypes_ReqTunnelInfo(_env, arg1, arg2, arg3, arg4, arg5, arg6, arg7)
    return obj
}
func (self *Types_ReqTunnelInfo) InitTypes_ReqTunnelInfo(_env *LnsEnv, arg1 string, arg2 string, arg3 LnsInt, arg4 string, arg5 string, arg6 *LnsList, arg7 *LnsMap) {
    self.id = arg1
    self.host = arg2
    self.port = arg3
    self.connectMode = arg4
    self.mode = arg5
    self.tunnelArgList = arg6
    self.envMap = arg7
}
func (self *Types_ReqTunnelInfo) Get_id(_env *LnsEnv) string{ return self.id }
func (self *Types_ReqTunnelInfo) Get_host(_env *LnsEnv) string{ return self.host }
func (self *Types_ReqTunnelInfo) Get_port(_env *LnsEnv) LnsInt{ return self.port }
func (self *Types_ReqTunnelInfo) Get_connectMode(_env *LnsEnv) string{ return self.connectMode }
func (self *Types_ReqTunnelInfo) Get_mode(_env *LnsEnv) string{ return self.mode }
func (self *Types_ReqTunnelInfo) Get_tunnelArgList(_env *LnsEnv) *LnsList{ return self.tunnelArgList }
func (self *Types_ReqTunnelInfo) Get_envMap(_env *LnsEnv) *LnsMap{ return self.envMap }
func (self *Types_ReqTunnelInfo) ToMapSetup( obj *LnsMap ) *LnsMap {
    obj.Items["id"] = Lns_ToCollection( self.id )
    obj.Items["host"] = Lns_ToCollection( self.host )
    obj.Items["port"] = Lns_ToCollection( self.port )
    obj.Items["connectMode"] = Lns_ToCollection( self.connectMode )
    obj.Items["mode"] = Lns_ToCollection( self.mode )
    obj.Items["tunnelArgList"] = Lns_ToCollection( self.tunnelArgList )
    obj.Items["envMap"] = Lns_ToCollection( self.envMap )
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
    if ok,conv,mess := Lns_ToStrSub( objMap.Items["id"], false, nil); !ok {
       return false,nil,"id:" + mess.(string)
    } else {
       newObj.id = conv.(string)
    }
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
    if ok,conv,mess := Lns_ToStrSub( objMap.Items["connectMode"], false, nil); !ok {
       return false,nil,"connectMode:" + mess.(string)
    } else {
       newObj.connectMode = conv.(string)
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
    if ok,conv,mess := Lns_ToLnsMapSub( objMap.Items["envMap"], false, []Lns_ToObjParam{Lns_ToObjParam{
            Lns_ToStrSub, false,nil},Lns_ToObjParam{
            Lns_ToStrSub, false,nil}}); !ok {
       return false,nil,"envMap:" + mess.(string)
    } else {
       newObj.envMap = conv.(*LnsMap)
    }
    return true, newObj, nil
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

type Types_HandleIF interface {
        GetTunnelInfo(_env *LnsEnv, arg1 string, arg2 *LnsMap)(LnsAny, string)
        OnEndTunnel(_env *LnsEnv, arg1 *Types_ReqTunnelInfo)
}
func Lns_cast2Types_HandleIF( obj LnsAny ) LnsAny {
    if _, ok := obj.(Types_HandleIF); ok { 
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
