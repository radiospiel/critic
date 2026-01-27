/*
htmx WebSocket Extension
Based on htmx-ext-ws@2.0.2
BSD 2-Clause License
*/
(function() {
    var api;
    var sockets = {};
    var reconnectDelays = {};

    function createWebSocket(socketId, url) {
        var socket = new WebSocket(url);
        sockets[socketId] = socket;

        socket.onopen = function(e) {
            reconnectDelays[socketId] = 0;
            var evt = new CustomEvent('htmx:wsOpen', {
                detail: { socketId: socketId, event: e }
            });
            document.body.dispatchEvent(evt);
        };

        socket.onmessage = function(e) {
            var evt = new CustomEvent('ws:message', {
                bubbles: true,
                detail: { message: e.data, socketId: socketId }
            });
            document.body.dispatchEvent(evt);

            // Also dispatch htmx-style event
            var htmxEvt = new CustomEvent('htmx:wsMessage', {
                bubbles: true,
                detail: { message: e.data, socketId: socketId }
            });
            document.body.dispatchEvent(htmxEvt);
        };

        socket.onclose = function(e) {
            var evt = new CustomEvent('htmx:wsClose', {
                detail: { socketId: socketId, event: e }
            });
            document.body.dispatchEvent(evt);

            // Reconnect with exponential backoff
            var delay = reconnectDelays[socketId] || 1000;
            reconnectDelays[socketId] = Math.min(delay * 2, 30000);

            setTimeout(function() {
                if (document.querySelector('[ws-connect="' + url + '"]')) {
                    createWebSocket(socketId, url);
                }
            }, delay);
        };

        socket.onerror = function(e) {
            var evt = new CustomEvent('htmx:wsError', {
                detail: { socketId: socketId, event: e }
            });
            document.body.dispatchEvent(evt);
        };

        return socket;
    }

    function processNode(elt) {
        var wsConnect = elt.getAttribute('ws-connect');
        if (wsConnect) {
            var url = wsConnect;
            if (url.startsWith('/')) {
                var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                url = protocol + '//' + window.location.host + url;
            }
            createWebSocket(url, url);
        }
    }

    function sendMessage(socketId, message) {
        var socket = sockets[socketId];
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(message);
        }
    }

    // Register with htmx if available
    if (typeof htmx !== 'undefined') {
        htmx.defineExtension('ws', {
            onEvent: function(name, evt) {
                if (name === 'htmx:beforeProcessNode') {
                    var elt = evt.detail.elt;
                    if (elt.getAttribute) {
                        processNode(elt);
                    }
                }
            }
        });

        // Process existing elements
        htmx.onLoad(function(elt) {
            if (elt.getAttribute && elt.getAttribute('ws-connect')) {
                processNode(elt);
            }
            var wsElts = elt.querySelectorAll ? elt.querySelectorAll('[ws-connect]') : [];
            wsElts.forEach(processNode);
        });
    }

    // Also process on DOMContentLoaded for standalone use
    document.addEventListener('DOMContentLoaded', function() {
        document.querySelectorAll('[ws-connect]').forEach(processNode);
    });

    // Expose API
    window.htmxWs = {
        send: sendMessage,
        sockets: sockets
    };
})();
