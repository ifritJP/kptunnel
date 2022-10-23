--lns/Types.lns
local _moduleObj = {}
local __mod__ = '@lns.@Types'
local _lune = {}
if _lune7 then
   _lune = _lune7
end
function _lune._toStem( val )
   return val
end
function _lune._toInt( val )
   if type( val ) == "number" then
      return math.floor( val )
   end
   return nil
end
function _lune._toReal( val )
   if type( val ) == "number" then
      return val
   end
   return nil
end
function _lune._toBool( val )
   if type( val ) == "boolean" then
      return val
   end
   return nil
end
function _lune._toStr( val )
   if type( val ) == "string" then
      return val
   end
   return nil
end
function _lune._toList( val, toValInfoList )
   if type( val ) == "table" then
      local tbl = {}
      local toValInfo = toValInfoList[ 1 ]
      for index, mem in ipairs( val ) do
         local memval, mess = toValInfo.func( mem, toValInfo.child )
         if memval == nil and not toValInfo.nilable then
            if mess then
              return nil, string.format( "%d.%s", index, mess )
            end
            return nil, index
         end
         tbl[ index ] = memval
      end
      return tbl
   end
   return nil
end
function _lune._toMap( val, toValInfoList )
   if type( val ) == "table" then
      local tbl = {}
      local toKeyInfo = toValInfoList[ 1 ]
      local toValInfo = toValInfoList[ 2 ]
      for key, mem in pairs( val ) do
         local mapKey, keySub = toKeyInfo.func( key, toKeyInfo.child )
         local mapVal, valSub = toValInfo.func( mem, toValInfo.child )
         if mapKey == nil or mapVal == nil then
            if mapKey == nil then
               return nil
            end
            if keySub == nil then
               return nil, mapKey
            end
            return nil, string.format( "%s.%s", mapKey, keySub)
         end
         tbl[ mapKey ] = mapVal
      end
      return tbl
   end
   return nil
end
function _lune._fromMap( obj, map, memInfoList )
   if type( map ) ~= "table" then
      return false
   end
   for index, memInfo in ipairs( memInfoList ) do
      local val, key = memInfo.func( map[ memInfo.name ], memInfo.child )
      if val == nil and not memInfo.nilable then
         return false, key and string.format( "%s.%s", memInfo.name, key) or memInfo.name
      end
      obj[ memInfo.name ] = val
   end
   return true
end

if not _lune7 then
   _lune7 = _lune
end
local ReqTunnelInfo = {}
setmetatable( ReqTunnelInfo, { ifList = {Mapping,} } )
_moduleObj.ReqTunnelInfo = ReqTunnelInfo
function ReqTunnelInfo._setmeta( obj )
  setmetatable( obj, { __index = ReqTunnelInfo  } )
end
function ReqTunnelInfo._new( host, port, mode, tunnelArgList )
   local obj = {}
   ReqTunnelInfo._setmeta( obj )
   if obj.__init then
      obj:__init( host, port, mode, tunnelArgList )
   end
   return obj
end
function ReqTunnelInfo:__init( host, port, mode, tunnelArgList )

   self.host = host
   self.port = port
   self.mode = mode
   self.tunnelArgList = tunnelArgList
end
function ReqTunnelInfo:get_host()
   return self.host
end
function ReqTunnelInfo:get_port()
   return self.port
end
function ReqTunnelInfo:get_mode()
   return self.mode
end
function ReqTunnelInfo:get_tunnelArgList()
   return self.tunnelArgList
end
function ReqTunnelInfo:_toMap()
  return self
end
function ReqTunnelInfo._fromMap( val )
  local obj, mes = ReqTunnelInfo._fromMapSub( {}, val )
  if obj then
     ReqTunnelInfo._setmeta( obj )
  end
  return obj, mes
end
function ReqTunnelInfo._fromStem( val )
  return ReqTunnelInfo._fromMap( val )
end

function ReqTunnelInfo._fromMapSub( obj, val )
   local memInfo = {}
   table.insert( memInfo, { name = "host", func = _lune._toStr, nilable = false, child = {} } )
   table.insert( memInfo, { name = "port", func = _lune._toInt, nilable = false, child = {} } )
   table.insert( memInfo, { name = "mode", func = _lune._toStr, nilable = false, child = {} } )
   table.insert( memInfo, { name = "tunnelArgList", func = _lune._toList, nilable = false, child = { { func = _lune._toStr, nilable = false, child = {} } } } )
   local result, mess = _lune._fromMap( obj, val, memInfo )
   if not result then
      return nil, mess
   end
   return obj
end


local HandleIF = {}
_moduleObj.HandleIF = HandleIF
function HandleIF._setmeta( obj )
  setmetatable( obj, { __index = HandleIF  } )
end
function HandleIF._new(  )
   local obj = {}
   HandleIF._setmeta( obj )
   if obj.__init then
      obj:__init(  )
   end
   return obj
end
function HandleIF:__init(  )

end




local AsyncHandleIF = {}
_moduleObj.AsyncHandleIF = AsyncHandleIF
function AsyncHandleIF._setmeta( obj )
  setmetatable( obj, { __index = AsyncHandleIF  } )
end
function AsyncHandleIF._new(  )
   local obj = {}
   AsyncHandleIF._setmeta( obj )
   if obj.__init then
      obj:__init(  )
   end
   return obj
end
function AsyncHandleIF:__init(  )

end


return _moduleObj
