// This code is transcompiled by LuneScript.
package lns
import . "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
import LnsFront "github.com/ifritJP/LuneScript/src/lune/base"
import LnsOpt "github.com/ifritJP/LuneScript/src/lune/base"
import LnsUtil "github.com/ifritJP/LuneScript/src/lune/base"
import lnsLog "github.com/ifritJP/LuneScript/src/lune/base"
var init_handle bool
var handle__mod__ string
var handle_handler Types_HandleIF
var handle_canAcceptRuntime *handle_Runtime
// for 23: ExpCast
func conv2Form0_327( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 51: ExpCast
func conv2Form0_408( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 73: ExpCast
func conv2Form0_544( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 169: ExpCast
func conv2Form0_924( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 23
func handle_convExp0_364(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 167
func handle_convExp0_958(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 217
func handle_convExp0_1151(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 238
func handle_convExp0_1211(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 256
func handle_convExp0_1260(arg1 []LnsAny) (LnsAny, string) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 ).(string)
}
// for 51
func handle_convExp0_424(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 58
func handle_convExp0_472(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 141
func handle_convExp0_779(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 165
func handle_convExp0_900(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// 188: decl @lns.@handle.setupHandle
func Handle_setupHandle(_env *LnsEnv, path string) bool {
    var list *LnsList2_[string]
    list = LnsUtil.Util_splitStr(_env, path, "[^,]+")
    if list.Len() != 2{
        Lns_print(Lns_2DDD(_env.GetVM().String_format("illegal param -- %s", Lns_2DDD(path))))
        return false
    }
    var handlePath string
    handlePath = list.GetAt(1)
    var canAccessPath string
    canAccessPath = list.GetAt(2)
    {
        _runtime := handle_Runtime_convScript_2_(_env, canAccessPath)
        if !Lns_IsNil( _runtime ) {
            runtime := _runtime.(*handle_Runtime)
            {
                __func := runtime.FP.loadScript(_env, "createHandler")
                if !Lns_IsNil( __func ) {
                    _func := __func.(*Lns_luaValue)
                        _env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))
                } else {
                    return false
                }
            }
            Lns_LockEnvSync( _env, 205, func () {
                handle_canAcceptRuntime = runtime
            })
        } else {
            return false
        }
    }
    var result bool
    result = false
    Lns_LockEnvSync( _env, 213, func () {
        {
            _handlerRuntime := handle_Runtime_convScript_2_(_env, handlePath)
            if !Lns_IsNil( _handlerRuntime ) {
                handlerRuntime := _handlerRuntime.(*handle_Runtime)
                {
                    __func := handlerRuntime.FP.loadScript(_env, "createHandler")
                    if !Lns_IsNil( __func ) {
                        _func := __func.(*Lns_luaValue)
                            {
                                _obj := handle_convExp0_1151(Lns_2DDD(_env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))))
                                if !Lns_IsNil( _obj ) {
                                    obj := _obj
                                    handle_handler = Newhandle_UserHandlerWrapper(_env, obj).FP
                                    result = true
                                } else {
                                    Lns_print(Lns_2DDD("illegal return value -- ", handlePath))
                                }
                            }
                    }
                }
            }
        }
    })
    return result
}

// 230: decl @lns.@handle.canAccept
func Handle_canAccept(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsInt, string) {
    var asyncHandler LnsAny
    asyncHandler = nil
    {
        __func := handle_canAcceptRuntime.FP.loadScript(_env, "createHandler")
        if !Lns_IsNil( __func ) {
            _func := __func.(*Lns_luaValue)
                {
                    _obj := handle_convExp0_1211(Lns_2DDD(_env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))))
                    if !Lns_IsNil( _obj ) {
                        obj := _obj
                        asyncHandler = Newhandle_UserAsyncHandlerWrapper(_env, obj)
                    } else {
                        Lns_print(Lns_2DDD("illegal return value -- "))
                    }
                }
        }
    }
    if asyncHandler != nil{
        asyncHandler_189 := asyncHandler.(*handle_UserAsyncHandlerWrapper)
        return asyncHandler_189.FP.CanAccept(_env, uri, headerMap)
    }
    return 200, ""
}

// 251: decl @lns.@handle.getTunnelInfo
func Handle_getTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    var info LnsAny
    var mess string
    Lns_LockEnvSync( _env, 255, func () {
        info, mess = handle_handler.GetTunnelInfo(_env, uri, headerMap)
    })
    return info, mess
}

// 262: decl @lns.@handle.onEndTunnel
func Handle_onEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
    Lns_LockEnvSync( _env, 263, func () {
        handle_handler.OnEndTunnel(_env, tunnelInfo)
    })
}

// 18: decl @lns.@handle.UserAsyncHandlerWrapper.canAccept
func (self *handle_UserAsyncHandlerWrapper) CanAccept(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsInt, string) {
    var statusCode LnsInt
    statusCode = 500
    var mess string
    mess = "internal error"
        {
            __func := self.obj.(*Lns_luaValue).GetAt("canAccept")
            if !Lns_IsNil( __func ) {
                _func := __func
                {
                    _work1, _work2 := handle_convExp0_364(Lns_2DDD(_env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD(Lns_2DDD(self.obj, uri, headerMap)))))
                    if !Lns_IsNil( _work1 ) && !Lns_IsNil( _work2 ) {
                        work1 := _work1
                        work2 := _work2
                        statusCode = Lns_forceCastInt(work1)
                        mess = work2.(string)
                    }
                }
            }
        }
    return statusCode, mess
}
// 43: decl @lns.@handle.UserHandlerWrapper.getTunnelInfo
func (self *handle_UserHandlerWrapper) GetTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    var luaval LnsAny
    luaval = nil
    var lua_mess LnsAny
    lua_mess = ""
    var val LnsAny
    val = nil
        {
            __func := self.obj.(*Lns_luaValue).GetAt("getTunnelInfo")
            if !Lns_IsNil( __func ) {
                _func := __func
                var work LnsAny
                var mess LnsAny
                work,mess = handle_convExp0_424(Lns_2DDD(_env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD(Lns_2DDD(self.obj, uri, headerMap)))))
                luaval = work
                lua_mess = _env.GetVM().ExpandLuavalMap(mess)
                val = _env.GetVM().ExpandLuavalMap(luaval)
            }
        }
    if val != nil{
        val_108 := val
        var info LnsAny
        var mess LnsAny
        info,mess = Types_ReqTunnelInfo__fromStem(_env, val_108,nil)
        if info != nil && luaval != nil{
            info_112 := info.(*Types_ReqTunnelInfo)
            luaval_113 := luaval
            self.reqTunnelInfoMap.Set(info_112,luaval_113)
            return info_112, ""
        }
        panic(_env.GetVM().String_format("failed to fromStem -- %s", Lns_2DDD(mess)))
    }
    if lua_mess != nil{
        lua_mess_115 := lua_mess
        return nil, lua_mess_115.(string)
    }
    return nil, "failed to getTunnelInfo"
}
// 70: decl @lns.@handle.UserHandlerWrapper.onEndTunnel
func (self *handle_UserHandlerWrapper) OnEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
        {
            __func := self.obj.(*Lns_luaValue).GetAt("onEndTunnel")
            if !Lns_IsNil( __func ) {
                _func := __func
                _env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD(Lns_2DDD(self.obj, self.reqTunnelInfoMap.Get(tunnelInfo))))
            }
        }
}
// 85: decl @lns.@handle.DefaultHandler.getTunnelInfo
func (self *handle_DefaultHandler) GetTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    __func__ := "@lns.@handle.DefaultHandler.getTunnelInfo"
    self.id = self.id + 1
    Lns_print(Lns_2DDD(__func__, "url", uri))
    for _key, _valList := range( headerMap.Items ) {
        key := _key.(string)
        valList := _valList.(*LnsList)
        for _, _val := range( valList.Items ) {
            val := _val.(string)
            Lns_print(Lns_2DDD(__func__, "header", _env.GetVM().String_format("%s: %s", Lns_2DDD(key, val))))
        }
    }
    var connectMode string
    connectMode = Types_ConnectMode__Client
    var mode string
    mode = "wsclient"
    return NewTypes_ReqTunnelInfo(_env, _env.GetVM().String_format("%d", Lns_2DDD(self.id)), "localhost", self.id, connectMode, mode, NewLnsList(Lns_2DDD("../kptunnel", mode, ":10000", "-omit", "-intlog", "-int", "5")), NewLnsMap( map[LnsAny]LnsAny{"GOGC":"50",})), ""
}
// 117: decl @lns.@handle.DefaultHandler.onEndTunnel
func (self *handle_DefaultHandler) OnEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
    __func__ := "@lns.@handle.DefaultHandler.onEndTunnel"
    Lns_print(Lns_2DDD(__func__, tunnelInfo.FP.Get_host(_env), tunnelInfo.FP.Get_port(_env)))
}
// 134: decl @lns.@handle.Runtime.createDummy
func handle_Runtime_createDummy_1_(_env *LnsEnv) *handle_Runtime {
    var path string
    path = "dummy"
    var option *LnsOpt.Option_Option
    option = LnsOpt.Option_analyze(_env, NewLnsList2_[string](Lns_2DDDGen[string](path, "exe")))
    return Newhandle_Runtime(_env, LnsFront.NewFront_Front(_env, option, nil), path, "")
}
// 140: decl @lns.@handle.Runtime.convScript
func handle_Runtime_convScript_2_(_env *LnsEnv, path string) LnsAny {
    var fileObj Lns_luaStream
    
    {
        _fileObj := handle_convExp0_779(Lns_2DDD(Lns_io_open(path, nil)))
        if _fileObj == nil{
            Lns_print(Lns_2DDD(_env.GetVM().String_format("failed to open -- %s", Lns_2DDD(path))))
            return nil
        } else {
            fileObj = _fileObj.(Lns_luaStream)
        }
    }
    var lnsCode string
    
    {
        _lnsCode := fileObj.Read(_env, "*a")
        if _lnsCode == nil{
            Lns_print(Lns_2DDD(_env.GetVM().String_format("failed to read -- %s", Lns_2DDD(path))))
            return nil
        } else {
            lnsCode = _lnsCode.(string)
        }
    }
    var front *LnsFront.Front_Front
    Lns_LockEnvSync( _env, 151, func () {
        var option *LnsOpt.Option_Option
        option = LnsOpt.Option_analyze(_env, NewLnsList2_[string](Lns_2DDDGen[string](path, "exe")))
        front = LnsFront.NewFront_Front(_env, option, nil)
    })
    var luaCode string
    luaCode = front.FP.ConvertLnsCode2LuaCodeWithOpt(_env, lnsCode, path, nil)
    return Newhandle_Runtime(_env, front, path, luaCode)
}
// 159: decl @lns.@handle.Runtime.loadScript
func (self *handle_Runtime) loadScript(_env *LnsEnv, funcName string) LnsAny {
    self.front.FP.SetupPreloadWithImportedModules(_env, true)
    var _func LnsAny
    _func = nil
        var loaded LnsAny
        var mess LnsAny
        loaded,mess = _env.GetVM().Load(self.luaCode, nil)
        if loaded != nil{
            loaded_149 := loaded.(*Lns_luaValue)
            {
                _mod := handle_convExp0_958(Lns_2DDD(_env.GetVM().RunLoadedfunc(loaded_149,Lns_2DDD([]LnsAny{}))))
                if !Lns_IsNil( _mod ) {
                    mod := _mod
                    {
                        _work := mod.(*Lns_luaValue).GetAt(funcName)
                        if !Lns_IsNil( _work ) {
                            work := _work
                            _func = work.(*Lns_luaValue)
                        } else {
                            Lns_print(Lns_2DDD("not found func -- ", funcName))
                        }
                    }
                } else {
                    Lns_print(Lns_2DDD("failed to exec the load module -- ", self.path))
                }
            }
        } else {
            Lns_print(Lns_2DDD("failed to load -- ", self.path, mess))
        }
    return _func
}
// declaration Class -- UserAsyncHandlerWrapper
type handle_UserAsyncHandlerWrapperMtd interface {
    CanAccept(_env *LnsEnv, arg1 string, arg2 *LnsMap)(LnsInt, string)
}
type handle_UserAsyncHandlerWrapper struct {
    obj LnsAny
    FP handle_UserAsyncHandlerWrapperMtd
}
func handle_UserAsyncHandlerWrapper2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*handle_UserAsyncHandlerWrapper).FP
}
func handle_UserAsyncHandlerWrapper_toSlice(slice []LnsAny) []*handle_UserAsyncHandlerWrapper {
    ret := make([]*handle_UserAsyncHandlerWrapper, len(slice))
    for index, val := range slice {
        ret[index] = val.(handle_UserAsyncHandlerWrapperDownCast).Tohandle_UserAsyncHandlerWrapper()
    }
    return ret
}
type handle_UserAsyncHandlerWrapperDownCast interface {
    Tohandle_UserAsyncHandlerWrapper() *handle_UserAsyncHandlerWrapper
}
func handle_UserAsyncHandlerWrapperDownCastF( multi ...LnsAny ) LnsAny {
    if len( multi ) == 0 { return nil }
    obj := multi[ 0 ]
    if ddd, ok := multi[ 0 ].([]LnsAny); ok { obj = ddd[0] }
    work, ok := obj.(handle_UserAsyncHandlerWrapperDownCast)
    if ok { return work.Tohandle_UserAsyncHandlerWrapper() }
    return nil
}
func (obj *handle_UserAsyncHandlerWrapper) Tohandle_UserAsyncHandlerWrapper() *handle_UserAsyncHandlerWrapper {
    return obj
}
func Newhandle_UserAsyncHandlerWrapper(_env *LnsEnv, arg1 LnsAny) *handle_UserAsyncHandlerWrapper {
    obj := &handle_UserAsyncHandlerWrapper{}
    obj.FP = obj
    obj.Inithandle_UserAsyncHandlerWrapper(_env, arg1)
    return obj
}
// 14: DeclConstr
func (self *handle_UserAsyncHandlerWrapper) Inithandle_UserAsyncHandlerWrapper(_env *LnsEnv, obj LnsAny) {
    self.obj = obj
}


// declaration Class -- UserHandlerWrapper
type handle_UserHandlerWrapperMtd interface {
    GetTunnelInfo(_env *LnsEnv, arg1 string, arg2 *LnsMap)(LnsAny, string)
    OnEndTunnel(_env *LnsEnv, arg1 *Types_ReqTunnelInfo)
}
type handle_UserHandlerWrapper struct {
    obj LnsAny
    reqTunnelInfoMap *LnsMap
    FP handle_UserHandlerWrapperMtd
}
func handle_UserHandlerWrapper2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*handle_UserHandlerWrapper).FP
}
func handle_UserHandlerWrapper_toSlice(slice []LnsAny) []*handle_UserHandlerWrapper {
    ret := make([]*handle_UserHandlerWrapper, len(slice))
    for index, val := range slice {
        ret[index] = val.(handle_UserHandlerWrapperDownCast).Tohandle_UserHandlerWrapper()
    }
    return ret
}
type handle_UserHandlerWrapperDownCast interface {
    Tohandle_UserHandlerWrapper() *handle_UserHandlerWrapper
}
func handle_UserHandlerWrapperDownCastF( multi ...LnsAny ) LnsAny {
    if len( multi ) == 0 { return nil }
    obj := multi[ 0 ]
    if ddd, ok := multi[ 0 ].([]LnsAny); ok { obj = ddd[0] }
    work, ok := obj.(handle_UserHandlerWrapperDownCast)
    if ok { return work.Tohandle_UserHandlerWrapper() }
    return nil
}
func (obj *handle_UserHandlerWrapper) Tohandle_UserHandlerWrapper() *handle_UserHandlerWrapper {
    return obj
}
func Newhandle_UserHandlerWrapper(_env *LnsEnv, arg1 LnsAny) *handle_UserHandlerWrapper {
    obj := &handle_UserHandlerWrapper{}
    obj.FP = obj
    obj.Inithandle_UserHandlerWrapper(_env, arg1)
    return obj
}
// 38: DeclConstr
func (self *handle_UserHandlerWrapper) Inithandle_UserHandlerWrapper(_env *LnsEnv, obj LnsAny) {
    self.obj = obj
    self.reqTunnelInfoMap = NewLnsMap( map[LnsAny]LnsAny{})
}


// declaration Class -- DefaultHandler
type handle_DefaultHandlerMtd interface {
    GetTunnelInfo(_env *LnsEnv, arg1 string, arg2 *LnsMap)(LnsAny, string)
    OnEndTunnel(_env *LnsEnv, arg1 *Types_ReqTunnelInfo)
}
type handle_DefaultHandler struct {
    id LnsInt
    FP handle_DefaultHandlerMtd
}
func handle_DefaultHandler2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*handle_DefaultHandler).FP
}
func handle_DefaultHandler_toSlice(slice []LnsAny) []*handle_DefaultHandler {
    ret := make([]*handle_DefaultHandler, len(slice))
    for index, val := range slice {
        ret[index] = val.(handle_DefaultHandlerDownCast).Tohandle_DefaultHandler()
    }
    return ret
}
type handle_DefaultHandlerDownCast interface {
    Tohandle_DefaultHandler() *handle_DefaultHandler
}
func handle_DefaultHandlerDownCastF( multi ...LnsAny ) LnsAny {
    if len( multi ) == 0 { return nil }
    obj := multi[ 0 ]
    if ddd, ok := multi[ 0 ].([]LnsAny); ok { obj = ddd[0] }
    work, ok := obj.(handle_DefaultHandlerDownCast)
    if ok { return work.Tohandle_DefaultHandler() }
    return nil
}
func (obj *handle_DefaultHandler) Tohandle_DefaultHandler() *handle_DefaultHandler {
    return obj
}
func Newhandle_DefaultHandler(_env *LnsEnv) *handle_DefaultHandler {
    obj := &handle_DefaultHandler{}
    obj.FP = obj
    obj.Inithandle_DefaultHandler(_env)
    return obj
}
// 81: DeclConstr
func (self *handle_DefaultHandler) Inithandle_DefaultHandler(_env *LnsEnv) {
    self.id = 10000
}


// declaration Class -- Runtime
type handle_RuntimeMtd interface {
    loadScript(_env *LnsEnv, arg1 string) LnsAny
}
type handle_Runtime struct {
    path string
    front *LnsFront.Front_Front
    luaCode string
    FP handle_RuntimeMtd
}
func handle_Runtime2Stem( obj LnsAny ) LnsAny {
    if obj == nil {
        return nil
    }
    return obj.(*handle_Runtime).FP
}
func handle_Runtime_toSlice(slice []LnsAny) []*handle_Runtime {
    ret := make([]*handle_Runtime, len(slice))
    for index, val := range slice {
        ret[index] = val.(handle_RuntimeDownCast).Tohandle_Runtime()
    }
    return ret
}
type handle_RuntimeDownCast interface {
    Tohandle_Runtime() *handle_Runtime
}
func handle_RuntimeDownCastF( multi ...LnsAny ) LnsAny {
    if len( multi ) == 0 { return nil }
    obj := multi[ 0 ]
    if ddd, ok := multi[ 0 ].([]LnsAny); ok { obj = ddd[0] }
    work, ok := obj.(handle_RuntimeDownCast)
    if ok { return work.Tohandle_Runtime() }
    return nil
}
func (obj *handle_Runtime) Tohandle_Runtime() *handle_Runtime {
    return obj
}
func Newhandle_Runtime(_env *LnsEnv, arg1 *LnsFront.Front_Front, arg2 string, arg3 string) *handle_Runtime {
    obj := &handle_Runtime{}
    obj.FP = obj
    obj.Inithandle_Runtime(_env, arg1, arg2, arg3)
    return obj
}
// 128: DeclConstr
func (self *handle_Runtime) Inithandle_Runtime(_env *LnsEnv, front *LnsFront.Front_Front,path string,luaCode string) {
    self.path = path
    self.front = front
    self.luaCode = luaCode
}


func Lns_handle_init(_env *LnsEnv) {
    if init_handle { return }
    init_handle = true
    handle__mod__ = "@lns.@handle"
    Lns_InitMod()
    Lns_Types_init(_env)
    LnsFront.Lns_front_init(_env)
    LnsOpt.Lns_Option_init(_env)
    LnsUtil.Lns_Util_init(_env)
    lnsLog.Lns_Log_init(_env)
    handle_handler = Newhandle_DefaultHandler(_env).FP
    handle_canAcceptRuntime = handle_Runtime_createDummy_1_(_env)
}
func init() {
    init_handle = false
}
