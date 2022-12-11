(function() {
    function Log() {
        console.log( "front: ", ...arguments );
    }

    let worker = new Worker('frontend_worker.js');
    worker.postMessage(
        { "kind":"init", "isMaster": true,
          "url1": 'ws://172.17.203.72:10000',
          "url2": 'ws://172.17.203.72:10001'} );

    
    window.addEventListener(
        'load',
        (event) => {
            let buttom = document.getElementById( "debug" );
            buttom.addEventListener(
                'click',
                (event) => {
                    worker.postMessage( { "kind":"dump" } );
                }
            );
        });
})();
