// This code is transcompiled by LuneScript.
package lns
import . "github.com/ifritJP/LuneScript/src/lune/base/runtime_go"
import LnsFront "github.com/ifritJP/LuneScript/src/lune/base"
import LnsOpt "github.com/ifritJP/LuneScript/src/lune/base"
import LnsUtil "github.com/ifritJP/LuneScript/src/lune/base"
var init_handle bool
var handle__mod__ string
var handle_handler Types_HandleIF
var handle_canAcceptLuaCode string
// for 20: ExpCast
func conv2Form0_279( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 47: ExpCast
func conv2Form0_352( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 65: ExpCast
func conv2Form0_464( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 131: ExpCast
func conv2Form0_764( src func (_env *LnsEnv)) LnsForm {
    return func (_env *LnsEnv,  argList []LnsAny) []LnsAny {
        src(_env)
        return []LnsAny{}
    }
}
// for 20
func handle_convExp0_316(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 129
func handle_convExp0_797(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 175
func handle_convExp0_1000(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 196
func handle_convExp0_1065(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 215
func handle_convExp0_1117(arg1 []LnsAny) (LnsAny, string) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 ).(string)
}
// for 47
func handle_convExp0_368(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 53
func handle_convExp0_405(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// for 107
func handle_convExp0_649(arg1 []LnsAny) LnsAny {
    return Lns_getFromMulti( arg1, 0 )
}
// for 127
func handle_convExp0_740(arg1 []LnsAny) (LnsAny, LnsAny) {
    return Lns_getFromMulti( arg1, 0 ), Lns_getFromMulti( arg1, 1 )
}
// 106: decl @lns.@handle.convScript
func handle_convScript_3_(_env *LnsEnv, path string) LnsAny {
    var fileObj Lns_luaStream
    
    {
        _fileObj := handle_convExp0_649(Lns_2DDD(Lns_io_open(path, nil)))
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
    Lns_LockEnvSync( _env, 117, func () {
        var option *LnsOpt.Option_Option
        option = LnsOpt.Option_analyze(_env, NewLnsList([]LnsAny{path, "lua"}))
        luaCode = LnsFront.Front_convertLnsCode2LuaCodeWithOpt(_env, option, lnsCode, path, nil)
    })
    return luaCode
}

// 124: decl @lns.@handle.loadScript
func handle_loadScript_4_(_env *LnsEnv, luaCode string,path string,funcName string) LnsAny {
    var _func LnsAny
    _func = nil
        var loaded LnsAny
        var mess LnsAny
        loaded,mess = _env.GetVM().Load(luaCode, nil)
        if loaded != nil{
            loaded_128 := loaded.(*Lns_luaValue)
            {
                _mod := handle_convExp0_797(Lns_2DDD(_env.GetVM().RunLoadedfunc(loaded_128,Lns_2DDD([]LnsAny{}))))
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
                    Lns_print([]LnsAny{"failed to exec the load module -- ", path})
                }
            }
        } else {
            Lns_print([]LnsAny{"failed to load -- ", path, mess})
        }
    return _func
}

// 145: decl @lns.@handle.setupHandle
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
        __exp := handle_convScript_3_(_env, canAccessPath)
        if !Lns_IsNil( __exp ) {
            _exp := __exp.(string)
            handle_canAcceptLuaCode = _exp
            var check LnsAny
                check = handle_loadScript_4_(_env, handle_canAcceptLuaCode, "asyncHandle", "createHandler")
            if Lns_op_not(check){
                return false
            }
        } else {
            return false
        }
    }
    var luaCode string
    
    {
        _luaCode := handle_convScript_3_(_env, handlePath)
        if _luaCode == nil{
            return false
        } else {
            luaCode = _luaCode.(string)
        }
    }
    var result bool
    result = false
    Lns_LockEnvSync( _env, 173, func () {
        {
            __func := handle_loadScript_4_(_env, luaCode, handlePath, "createHandler")
            if !Lns_IsNil( __func ) {
                _func := __func.(*Lns_luaValue)
                {
                    _obj := handle_convExp0_1000(Lns_2DDD(_env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))))
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
    })
    return result
}

// 187: decl @lns.@handle.canAccept
func Handle_canAccept(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsInt, string) {
    var asyncHandler LnsAny
    asyncHandler = nil
    if handle_canAcceptLuaCode != ""{
            {
                __func := handle_loadScript_4_(_env, handle_canAcceptLuaCode, "createHandler", "createHandler")
                if !Lns_IsNil( __func ) {
                    _func := __func.(*Lns_luaValue)
                    {
                        _obj := handle_convExp0_1065(Lns_2DDD(_env.GetVM().RunLoadedfunc(_func,Lns_2DDD([]LnsAny{}))))
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
            asyncHandler_167 := asyncHandler.(*handle_UserAsyncHandlerWrapper)
            return asyncHandler_167.FP.CanAccept(_env, uri, headerMap)
        }
    }
    return 200, ""
}

// 210: decl @lns.@handle.getTunnelInfo
func Handle_getTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    var info LnsAny
    var mess string
    Lns_LockEnvSync( _env, 214, func () {
        info, mess = handle_handler.GetTunnelInfo(_env, uri, headerMap)
    })
    return info, mess
}

// 221: decl @lns.@handle.onEndTunnel
func Handle_onEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
    Lns_LockEnvSync( _env, 222, func () {
        handle_handler.OnEndTunnel(_env, tunnelInfo)
    })
}

// 15: decl @lns.@handle.UserAsyncHandlerWrapper.canAccept
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
                    _work1, _work2 := handle_convExp0_316(Lns_2DDD(_env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD([]LnsAny{self.obj, uri, headerMap}))))
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
// 40: decl @lns.@handle.UserHandlerWrapper.getTunnelInfo
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
                work = handle_convExp0_368(Lns_2DDD(_env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD([]LnsAny{self.obj, uri, headerMap}))))
                luaval = work
                val = _env.GetVM().ExpandLuavalMap(luaval)
            }
        }
    if val != nil{
        val_93 := val
        var info LnsAny
        var mess LnsAny
        info,mess = Types_ReqTunnelInfo__fromStem(_env, val_93,nil)
        if info != nil && luaval != nil{
            info_97 := info.(*Types_ReqTunnelInfo)
            luaval_98 := luaval
            self.reqTunnelInfoMap.Set(info_97,luaval_98)
            return info_97, ""
        }
        panic(_env.GetVM().String_format("failed to fromStem -- %s", []LnsAny{mess}))
    }
    panic("failed to expandLuavalMap")
// insert a dummy
    return nil,""
}
// 62: decl @lns.@handle.UserHandlerWrapper.onEndTunnel
func (self *handle_UserHandlerWrapper) OnEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
        {
            __func := self.obj.(*Lns_luaValue).GetAt("onEndTunnel")
            if !Lns_IsNil( __func ) {
                _func := __func
                _env.GetVM().RunLoadedfunc((_func.(*Lns_luaValue)),Lns_2DDD([]LnsAny{self.obj, self.reqTunnelInfoMap.Get(tunnelInfo)}))
            }
        }
}
// 77: decl @lns.@handle.DefaultHandler.getTunnelInfo
func (self *handle_DefaultHandler) GetTunnelInfo(_env *LnsEnv, uri string,headerMap *LnsMap)(LnsAny, string) {
    __func__ := "@lns.@handle.DefaultHandler.getTunnelInfo"
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
    connectMode = Types_ConnectMode__CanReconnect
    var mode string
    mode = "server"
    return NewTypes_ReqTunnelInfo(_env, "localhost", self.port, connectMode, mode, NewLnsList([]LnsAny{"../kptunnel", mode, _env.GetVM().String_format(":%d", []LnsAny{self.port}), _env.GetVM().String_format(":%d,192.168.0.101:22", []LnsAny{10000 + self.port})}), NewLnsMap( map[LnsAny]LnsAny{"GOGC":"50",})), ""
}
// 98: decl @lns.@handle.DefaultHandler.onEndTunnel
func (self *handle_DefaultHandler) OnEndTunnel(_env *LnsEnv, tunnelInfo *Types_ReqTunnelInfo) {
    __func__ := "@lns.@handle.DefaultHandler.onEndTunnel"
    Lns_print([]LnsAny{__func__, tunnelInfo.FP.Get_host(_env), tunnelInfo.FP.Get_port(_env)})
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
// 11: DeclConstr
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
// 35: DeclConstr
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
    port LnsInt
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
// 73: DeclConstr
func (self *handle_DefaultHandler) Inithandle_DefaultHandler(_env *LnsEnv) {
    self.port = 10000
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
    handle_handler = Newhandle_DefaultHandler(_env).FP
    handle_canAcceptLuaCode = ""
}
func init() {
    init_handle = false
}
