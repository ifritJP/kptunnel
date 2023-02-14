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
func conv2Form0_329( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 50: ExpCast
func conv2Form0_402( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 68: ExpCast
func conv2Form0_513( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 166: ExpCast
func conv2Form0_897( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 23
func handle_convExp0_366(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 164
func handle_convExp0_931(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 214
func handle_convExp0_1124(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 235
func handle_convExp0_1184(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 253
func handle_convExp0_1233(arg1 []LnsAny) (LnsAny, string) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 ).(string)
}
// for 50
func handle_convExp0_418(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 56
func handle_convExp0_455(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 136
func handle_convExp0_742(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 162
func handle_convExp0_873(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// 185: decl @lns.@handle.setupHandle
func Handle_setupHandle(_env *LnsEnv, path string) bool {
    var list *LnsList
    list = LnsUtil.Util_splitStr(_env, path, "[^,]+")
    if list.Len() != 2{
        Lns_print([]LnsAny{_env.GetVM().String_format("illegal param -- %s", []LnsAny{path})})
        return false
    }
    var handlePath string
    handlePath = list.GetAt(1).(string)
    var canAccessPath string
    canAccessPath = list.GetAt(2).(string)
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
            Lns_LockEnvSync( _env, 202, func () {
                handle_canAcceptRuntime = runtime
            })
        } else {
            return false
        }
    }
    var result bool
    result = false
    Lns_LockEnvSync( _env, 210, func () {
        {
            _handlerRuntime := handle_Runtime_convScript_2_(_env, handlePath)
            if !Lns_IsNil( _handlerRuntime ) {
                handlerRuntime := _handlerRuntime.(*handle_Runtime)
                {
                    __func := handlerRuntime.FP.loadScript(_env, "createHandler")
                    if !Lns_IsNil( __func ) {
                        _func := __func.(*Lns_luaValue)
                            {
                                _obj := handle_convExp0_1124(Lns_2DDD(_env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))))
                                if !Lns_IsNil( _obj ) {
                                    obj := _obj
                                    handle_handler = Newhandle_UserHandlerWrapper(_env, obj).FP
                                    result = true
                                } else {
                                    Lns_print([]LnsAny{"illegal return value -- ", handlePath})
                                }
                            }
                    }
                }
            }
        }
    })
    return result
}

// 227: decl @lns.@handle.canAccept
func Handle_canAccept(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsInt, string) {
    var asyncHandler LnsAny
    asyncHandler = nil
    {
        __func := handle_canAcceptRuntime.FP.loadScript(_env, "createHandler")
        if !Lns_IsNil( __func ) {
            _func := __func.(*Lns_luaValue)
                {
                    _obj := handle_convExp0_1184(Lns_2DDD(_env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))))
                    if !Lns_IsNil( _obj ) {
                        obj := _obj
                        asyncHandler = Newhandle_UserAsyncHandlerWrapper(_env, obj)
                    } else {
                        Lns_print([]LnsAny{"illegal return value -- "})
                    }
                }
        }
    }
    if asyncHandler != nil{
        asyncHandler_185 := asyncHandler.(*handle_UserAsyncHandlerWrapper)
        return asyncHandler_185.FP.CanAccept(_env, uri, headerMap)
    }
    return 200, ""
}

// 248: decl @lns.@handle.getTunnelInfo
func Handle_getTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    var info LnsAny
    var mess string
    Lns_LockEnvSync( _env, 252, func () {
        info, mess = handle_handler.GetTunnelInfo(_env, uri, headerMap)
    })
    return info, mess
}

// 259: decl @lns.@handle.onEndTunnel
func Handle_onEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
    Lns_LockEnvSync( _env, 260, func () {
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
                    _work1, _work2 := handle_convExp0_366(Lns_2DDD(_env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD([]LnsAny{self.obj, uri, headerMap}))))
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
    var val LnsAny
    val = nil
        {
            __func := self.obj.(*Lns_luaValue).GetAt("getTunnelInfo")
            if !Lns_IsNil( __func ) {
                _func := __func
                var work LnsAny
                work = handle_convExp0_418(Lns_2DDD(_env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD([]LnsAny{self.obj, uri, headerMap}))))
                luaval = work
                val = _env.GetVM().ExpandLuavalMap(luaval)
            }
        }
    if val != nil{
        val_106 := val
        var info LnsAny
        var mess LnsAny
        info,mess = Types_ReqTunnelInfo__fromStem(_env, val_106,nil)
        if info != nil && luaval != nil{
            info_110 := info.(*Types_ReqTunnelInfo)
            luaval_111 := luaval
            self.reqTunnelInfoMap.Set(info_110,luaval_111)
            return info_110, ""
        }
        panic(_env.GetVM().String_format("failed to fromStem -- %s", []LnsAny{mess}))
    }
    return nil, "failed to getTunnelInfo"
}
// 65: decl @lns.@handle.UserHandlerWrapper.onEndTunnel
func (self *handle_UserHandlerWrapper) OnEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
        {
            __func := self.obj.(*Lns_luaValue).GetAt("onEndTunnel")
            if !Lns_IsNil( __func ) {
                _func := __func
                _env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD([]LnsAny{self.obj, self.reqTunnelInfoMap.Get(tunnelInfo)}))
            }
        }
}
// 80: decl @lns.@handle.DefaultHandler.getTunnelInfo
func (self *handle_DefaultHandler) GetTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    __func__ := "@lns.@handle.DefaultHandler.getTunnelInfo"
    self.id = self.id + 1
    Lns_print([]LnsAny{__func__, "url", uri})
    for _key, _valList := range( headerMap.Items ) {
        key := _key.(string)
        valList := _valList.(*LnsList)
        for _, _val := range( valList.Items ) {
            val := _val.(string)
            Lns_print([]LnsAny{__func__, "header", _env.GetVM().String_format("%s: %s", []LnsAny{key, val})})
        }
    }
    var connectMode string
    connectMode = Types_ConnectMode__Client
    var mode string
    mode = "wsclient"
    return NewTypes_ReqTunnelInfo(_env, _env.GetVM().String_format("%d", []LnsAny{self.id}), "localhost", self.id, connectMode, mode, NewLnsList([]LnsAny{"../kptunnel", mode, ":10000", "-omit"}), NewLnsMap( map[LnsAny]LnsAny{"GOGC":"50",})), ""
}
// 112: decl @lns.@handle.DefaultHandler.onEndTunnel
func (self *handle_DefaultHandler) OnEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
    __func__ := "@lns.@handle.DefaultHandler.onEndTunnel"
    Lns_print([]LnsAny{__func__, tunnelInfo.FP.Get_host(_env), tunnelInfo.FP.Get_port(_env)})
}
// 129: decl @lns.@handle.Runtime.createDummy
func handle_Runtime_createDummy_1_(_env *LnsEnv) *handle_Runtime {
    var path string
    path = "dummy"
    var option *LnsOpt.Option_Option
    option = LnsOpt.Option_analyze(_env, NewLnsList([]LnsAny{path, "exe"}))
    return Newhandle_Runtime(_env, LnsFront.NewFront_Front(_env, option, nil), path, "")
}
// 135: decl @lns.@handle.Runtime.convScript
func handle_Runtime_convScript_2_(_env *LnsEnv, path string) LnsAny {
    var fileObj Lns_luaStream
    
    {
        _fileObj := handle_convExp0_742(Lns_2DDD(Lns_io_open(path, nil)))
        if _fileObj == nil{
            Lns_print([]LnsAny{_env.GetVM().String_format("failed to open -- %s", []LnsAny{path})})
            return nil
        } else {
            fileObj = _fileObj.(Lns_luaStream)
        }
    }
    var lnsCode string
    
    {
        _lnsCode := fileObj.Read(_env, "*a")
        if _lnsCode == nil{
            Lns_print([]LnsAny{_env.GetVM().String_format("failed to read -- %s", []LnsAny{path})})
            return nil
        } else {
            lnsCode = _lnsCode.(string)
        }
    }
    var luaCode string
    var front *LnsFront.Front_Front
    var option *LnsOpt.Option_Option
    Lns_LockEnvSync( _env, 148, func () {
        option = LnsOpt.Option_analyze(_env, NewLnsList([]LnsAny{path, "exe"}))
        front = LnsFront.NewFront_Front(_env, option, nil)
    })
    luaCode = front.FP.ConvertLnsCode2LuaCodeWithOpt(_env, option, lnsCode, path, nil)
    return Newhandle_Runtime(_env, front, path, luaCode)
}
// 156: decl @lns.@handle.Runtime.loadScript
func (self *handle_Runtime) loadScript(_env *LnsEnv, funcName string) LnsAny {
    self.front.FP.SetupPreloadWithImportedModules(_env, true)
    var _func LnsAny
    _func = nil
        var loaded LnsAny
        var mess LnsAny
        loaded,mess = _env.GetVM().Load(self.luaCode, nil)
        if loaded != nil{
            loaded_145 := loaded.(*Lns_luaValue)
            {
                _mod := handle_convExp0_931(Lns_2DDD(_env.GetVM().RunLoadedfunc(loaded_145,Lns_2DDD([]LnsAny{}))))
                if !Lns_IsNil( _mod ) {
                    mod := _mod
                    {
                        _work := mod.(*Lns_luaValue).GetAt(funcName)
                        if !Lns_IsNil( _work ) {
                            work := _work
                            _func = work.(*Lns_luaValue)
                        } else {
                            Lns_print([]LnsAny{"not found func -- ", funcName})
                        }
                    }
                } else {
                    Lns_print([]LnsAny{"failed to exec the load module -- ", self.path})
                }
            }
        } else {
            Lns_print([]LnsAny{"failed to load -- ", self.path, mess})
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
// 76: DeclConstr
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
// 123: DeclConstr
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
