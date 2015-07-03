(function() {
  var app = angular.module('websocket', []);

  app.service('WS', ['$rootScope', function($rootScope) {
    var handlers = {};

    var newConnection = function(onopen, onclose) {
      if (!("WebSocket" in window)) {
        // TODO fallback to ajax
        alert("Sorry, your browser does not support WebSockets.");
        return;
      }

      console.log("Trying to connect to WebSocket server...");
      var ws = new WebSocket("wss://" + location.host + "/events");

      ws.onopen = function() {
        if (typeof onopen === "function")
          onopen();
      };

      ws.onclose = function() {
        window.setTimeout(function() {
          newConnection(onopen, onclose);
        }, 5000);

        if (typeof onclose === "function")
          onclose();
      };

      ws.onerror = function() {
        console.log("WebSocket error!");
      };

      ws.onmessage = function(event) {
        var data = $.parseJSON(event.data);

        // call the handler for the event if it exists
        if (handlers.hasOwnProperty(data.type)) {
          $rootScope.$apply(handlers[data.type](data));
        } else {
          console.log("[WS] no handler for event", data.type);
        }
      };
    };

    this.newConnection = newConnection;

    this.registerHandler = function(event, handler) {
      if (typeof event !== "string") {
        console.log("'event' has to be a string");
        return;
      }

      if (typeof handler !== "function") {
        console.log("'handler' has to be a function");
        return;
      }

      handlers[event] = handler;
    };
  }]);
})();
