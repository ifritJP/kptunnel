importScripts("wasm_exec.js");


/**
   lnsc を制御する frontend。
   実際の lnsc は lnsc_frontendWorker.js で動かす。
 */
(function() {
    function toBinStr( bufArray ) {
        let binList = [];
        let array = new Uint8Array( bufArray );

        let size = 1024;
        for ( let index = 0; index < array.length; index += size ) {
            let rest = size;
            if ( index + size > array.length ) {
                rest = array.length - index;
            }
            binList.push( String.fromCharCode.apply(
                null, array.slice( index, index + rest ) ) );
        }
        return binList.join( '' );
    }
    function decB64( b64 ) {
        return Uint8Array.from(atob( b64 ), str => str.charCodeAt(0) );
    }
    

    async function load_wasm( isMaster, url1, url2 ) {
        function Log() {
            let mess;
            if ( isMaster ) {
                mess = "main: worker: ";
            } else {
                mess = "sub : worker: ";
            }
            console.log( mess, ...arguments );
        }
        
        let ifObj = null;

        const go = new Go(); // definition in wasm_exec.js
        let res = await WebAssembly.instantiateStreaming(
            fetch( "kptunnel.wasm" ), go.importObject);


        let worker = null;
        let citiObj = null;
        if ( isMaster ) {
            worker = new Worker('frontend_worker.js');
            worker.postMessage(
                { "kind": "init",
                  "isMaster": !isMaster, "url1": url2 } );
            citiObj = worker;
        } else {
            citiObj = self;
        }
        citiObj.addEventListener(
            'message',
            async function (mess) {
                if ( mess.data.kind == "pack" ) {
                    Log( "receive citi", mess.data.connId, mess.data.b64.length );
                    ifObj.fromCiti( mess.data.connId, mess.data.b64 );
                } else if ( mess.data.kind == "dial" ) {
                    Log( "dial", mess.data.connId );
                    ifObj.startCiti( mess.data.connId );
                    citiObj.postMessage( { "kind":"accept", connId:mess.data.connId } );
                } else if ( mess.data.kind == "accept" ) {
                    Log( "accept", mess.data.connId );
                    ifObj.onAccept( mess.data.connId );
                } else if ( mess.data.kind == "close" ) {
                    Log( "close", mess.data.connId );
                    ifObj.fromCiti( mess.data.connId, "" );
                }
            } );
        self.addEventListener(
            'message',
            (mess) => {
                if ( mess.data.kind == "dump" ) {
                    ifObj.dumpDebug();
                    if ( isMaster ) {
                        citiObj.postMessage( mess.data );
                    }
                }
            } );

        

        let socket = new WebSocket( url1 );
        socket.binaryType = "arraybuffer";
        socket.addEventListener( 'error', function( event ) {
            console.log( "websocket error", event );
            ifObj.fromTunnel( "" );
        });
        socket.addEventListener( 'message', async function( event ) {
            let src;
            if ( event.data instanceof ArrayBuffer ) {
                src = toBinStr( event.data );
            } else {
                src = event.data;
            }
            let b64 = btoa( src );
            ifObj.fromTunnel( b64 );
        });

        socket.addEventListener(
            'open',
            async function( event ) {
                // execute the go main method
                go.argv = [ "_ifObj" ];
                go.run(res.instance ).then( ()=> {
                    Log( "detect exit" );
                });

                ifObj = _ifObj( isMaster );
                
                ifObj.startClient(
                    function( connId, b64 ) {
                        Log( "send tunnel" );
                        let bin = decB64( b64 );
                        socket.send( bin );
                        // 通信可能な場合 true を返す
                        return socket.readyState == 1;
                    },
                    function( connId, kind ) {
                        Log( "send ctrl -- ", connId, kind );
                        citiObj.postMessage( { "kind": kind, "connId": connId } );
                    },
                    function( connId, b64 ) {
                        Log( "send citi -- ", connId, b64.length );
                        let mess = { kind: "pack", connId: connId, "b64": b64 };
                        citiObj.postMessage( mess );
                        // 通信可能な場合 true を返す
                        return socket.readyState == 1;
                    }
                );
            } );

    }

    self.addEventListener(
        'message',
        (mess) => {
            if ( mess.data.kind == "init" ) {
                load_wasm( mess.data.isMaster, mess.data.url1, mess.data.url2 );
            }
        }
    );
})();
