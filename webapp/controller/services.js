(function () {

  "use strict";

  var App = angular.module("App.services", []);

  App.value('version', '0.1');

  // here is a declaration of simple utility function to know if an given param is a String.
  App.service("UtilSrvc", function () {
    return {
      isAString: function (o) {
        return typeof o == "string" || (typeof o == "object" && o.constructor === String);
      },
      helloWorld: function (name) {
        var result = "Hum, Hello you, but your name is too weird...";
        if (this.isAString(name)) {
          result = "Hello, " + name;
        }
        return result;
      }
    }
  });

  let socket
  let connected = false

  function setupSocket() {
    socket = io()
    socket.on('error', function () {
      socket.close()
      console.log('error')
      setTimeout(() => {
        setupSocket()
      }, 200);
    })
    socket.on('connect', function () {
      console.log('initial connect')
      socket.emit('get-node-state')
      socket.once('node-state-info', function (answer) {
        console.log("final connect")
        connected = true
        setTimeout(() => {
          queue.forEach(args => {
            console.log('item in queue')
            socket[args.fn].apply(socket, args.args)
          })
          queue = []
        }, 200);
      })
    })
  }
  setupSocket()
  let queue = []

  App.service("Socket", function () {
    return {
      emit() {
        if (!connected) {
          queue.push({ fn: "emit", args: arguments })
        } else {
          return socket.emit.apply(socket, arguments)
        }
      },
      on() {
        if (!connected) {
          queue.push({ fn: "on", args: arguments })
        } else {
          return socket.on.apply(socket, arguments)
        }
      }
    }
  })

  

}());