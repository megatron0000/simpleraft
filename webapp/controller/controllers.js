(function () {

  function urlInvalid(address) {
    let url
    try {
      url = new URL("http://" + address)
      if (!url.port) {
        throw new Error()
      }
    } catch {
      url = null
    }
    return url === null
  }

  "use strict";

  var App = angular.module("App.controllers", []);

  App.controller("MyCtrl1", ["$scope", function ($scope, UtilSrvc) {
    $scope.aVariable = 'anExampleValueWithinScope';
    $scope.valueFromService = UtilSrvc.helloWorld("User");
  }]);

  App.controller("MyCtrl2", ["$scope", function ($scope) {
    // if you have many controllers, it's better to separate them into files
  }]);

  App.controller("ClientController", function ($scope, Socket) {
    $scope.waitingResponse = false
    $scope.response = "none"


    $scope.commandInvalid = function (command) {
      return ['read', 'write'].indexOf(command) === -1
    }
    /**
     * @param {string} key
     */
    $scope.keyInvalid = function (key) {
      return !key || key.trim() === ''
    }
    /**
     * @param {string} value
     */
    $scope.valueInvalid = function (value) {
      return $scope.command == 'write' && (!value || value.trim() === '')
    }

    $scope.sendCommand = function (command, key, value) {
      $scope.waitingResponse = true
      $scope.response = "......"
      setTimeout(function () {
        Socket.emit('new-client-command', command, key, value || '')
      }, 1000);
    }

    Socket.on('ack', function () {
      $scope.waitingResponse = false
      $scope.$apply()
    })

    /**
     * @param {string} info
     */
    Socket.on('client-info', function (info) {
      let res = info
      try {
        res = atob(info)
      } catch (e) {
        res = JSON.stringify(info, null, 2)
      }
      $scope.response = res
      $scope.$apply()
    })


    $scope.indexInvalid = function (index) {
      return Math.round(index) !== index || index < 0
    }

    $scope.termInvalid = function (term) {
      return Math.round(term) !== term || term < 0
    }

    $scope.sendProbe = function (index, term) {
      $scope.waitingResponse = true
      $scope.response = "......"
      setTimeout(function () {
        Socket.emit('new-client-probe', index, term)
      }, 1000);
    }

    $scope.typeInvalid = function (type) {
      return !type || (type !== 'addserver' && type !== 'rmserver')
    }

    $scope.addressInvalid = function (address) {
      return urlInvalid(address)
    }

    $scope.sendClusterChange = function (type, address) {
      $scope.waitingResponse = true
      $scope.response = "......"
      setTimeout(function () {
        Socket.emit('new-client-cluster-change', type, address)
      }, 1000);
    }

  })

  App.controller("StatusController", function ($scope, Socket) {

    // if ($('[data-toggle="switch"]').length) {
    //   $('[data-toggle="switch"]').bootstrapSwitch();
    // }



    $scope.executorRunning = false
    Socket.emit('get-executor-is-running')
    Socket.on('executor-is-running-info', function (isRunning) {
      $scope.executorRunning = isRunning
      $scope.$apply()
    })
    $scope.setExecutorIsRunning = function () {
      console.log($scope.executorRunning)
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-executor-is-running', $scope.executorRunning)
      }, 1000);
    }

    $scope.waitingResponse = false
    Socket.on('ack', function () {
      $scope.waitingResponse = false
      $scope.$apply()
    })
    $scope.test = function () { console.log('test') }

    // $scope.somevar = "hola"
    $scope.nodeState = ""
    Socket.emit('get-node-state')
    Socket.on('node-state-info', function (newState) {
      $scope.nodeState = newState
      $scope.$apply()
    })
    $scope.nodeStateInvalid = function nodeStateInvalid(nodeState) {
      return nodeState !== "follower" && nodeState !== "leader" && nodeState !== "candidate"
    }
    $scope.setNodeState = function setNodeState(nodeState) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-node-state', nodeState)
      }, 1000)
    }

    $scope.nodeAddress = ""
    Socket.emit('get-node-address')
    Socket.on('node-address-info', function (newNodeAddress) {
      $scope.nodeAddress = newNodeAddress
      $scope.$apply()
    })
    $scope.nodeAddressInvalid = function nodeAddressInvalid(nodeAddress) {
      return urlInvalid(nodeAddress)
    }
    $scope.setNodeAddress = function setNodeAddress(nodeAddress) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-node-address', nodeAddress)
      }, 1000)
    }

    $scope.currentTerm = -1
    Socket.emit('get-current-term')
    Socket.on('current-term-info', function (newCurrentTerm) {
      $scope.currentTerm = newCurrentTerm
      $scope.$apply()
    })
    $scope.currentTermInvalid = function currentTermInvalid(currentTerm) {
      return Math.round(currentTerm) !== currentTerm || currentTerm < 0
    }
    $scope.setCurrentTerm = function setCurrentTerm(currentTerm) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-current-term', currentTerm)
      }, 1000)
    }

    $scope.votedFor = ""
    Socket.emit('get-voted-for')
    Socket.on('voted-for-info', function (newVotedFor) {
      $scope.votedFor = newVotedFor
      $scope.$apply()
    })
    $scope.votedForInvalid = function (votedFor) {
      return urlInvalid(votedFor) && votedFor !== ""
    }
    $scope.setVotedFor = function (votedFor) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-voted-for', votedFor)
      }, 1000)
    }

    $scope.voteCount = -1
    Socket.emit('get-vote-count')
    Socket.on('vote-count-info', function (newVoteCount) {
      $scope.voteCount = newVoteCount
      $scope.$apply()
    })
    $scope.voteCountInvalid = function (voteCount) {
      return Math.round(voteCount) !== voteCount || voteCount < 0
    }
    $scope.setVoteCount = function (voteCount) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-vote-count', voteCount)
      }, 1000)
    }

    $scope.commitIndex = -1
    Socket.emit('get-commit-index')
    Socket.on('commit-index-info', function (newCommitIndex) {
      console.log(Socket)
      $scope.commitIndex = newCommitIndex
      $scope.$apply()
    })
    $scope.commitIndexInvalid = function (commitIndex) {
      return Math.round(commitIndex) !== commitIndex || commitIndex < -1
    }
    $scope.setCommitIndex = function (commitIndex) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-commit-index', commitIndex)
      }, 1000)
    }

    $scope.lastApplied = -1
    Socket.emit('get-last-applied')
    Socket.on('last-applied-info', function (newLastApplied) {
      $scope.lastApplied = newLastApplied
      $scope.$apply()
    })
    $scope.lastAppliedInvalid = function (lastApplied) {
      return Math.round(lastApplied) !== lastApplied || lastApplied < -1
    }
    $scope.setLastApplied = function (lastApplied) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-last-applied', lastApplied)
      }, 1000)
    }

    $scope.peerAddresses = ""
    Socket.emit('get-peer-addresses')
    Socket.on('peer-addresses-info', function (newPeerAddresses) {
      $scope.peerAddresses = newPeerAddresses
      console.log($scope)
      $scope.$apply()
    })
    $scope.peerAddressesInvalid = function (peerAddresses) {
      for (const addr of peerAddresses) {
        if (urlInvalid(addr)) {
          return true
        }
      }
      return false
    }
    $scope.setPeerAddresses = function (peerAddresses) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-peer-addresses', JSON.stringify(peerAddresses))
      }, 1000)
    }

    $scope.nextIndexes = []
    Socket.emit('get-next-indexes')
    Socket.on('next-indexes-info', function (newNextIndexes) {
      $scope.nextIndexes = newNextIndexes
      $scope.$apply()
    })
    $scope.nextIndexInvalid = function (nextIndex) {
      return Math.round(nextIndex) !== nextIndex || nextIndex < 0
    }
    $scope.setNextIndex = function (addr, nextIndex) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-next-index', addr, nextIndex)
      }, 1000)
    }

    $scope.matchIndexes = []
    Socket.emit('get-match-indexes')
    Socket.on('match-indexes-info', function (newMatchIndexes) {
      $scope.matchIndexes = newMatchIndexes
      $scope.$apply()
    })
    $scope.matchIndexInvalid = function (matchIndex) {
      return Math.round(matchIndex) !== matchIndex || matchIndex < -1
    }
    $scope.setMatchIndex = function (addr, matchIndex) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-match-index', addr, matchIndex)
      }, 1000)
    }

    $scope.clusterChangeIndex = ""
    Socket.emit('get-cluster-change-index')
    Socket.on('cluster-change-index-info', function (newClusterChangeIndex) {
      $scope.clusterChangeIndex = newClusterChangeIndex
      $scope.$apply()
    })
    $scope.clusterChangeIndexInvalid = function (clusterChangeIndex) {
      return Math.round(clusterChangeIndex) !== clusterChangeIndex || clusterChangeIndex < -1
    }
    $scope.setClusterChangeIndex = function (clusterChangeIndex) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-cluster-change-index', clusterChangeIndex)
      }, 1000)
    }

    $scope.clusterChangeTerm = ""
    Socket.emit('get-cluster-change-term')
    Socket.on('cluster-change-term-info', function (newClusterChangeTerm) {
      $scope.clusterChangeTerm = newClusterChangeTerm
      $scope.$apply()
    })
    $scope.clusterChangeTermInvalid = function (clusterChangeTerm) {
      return Math.round(clusterChangeTerm) !== clusterChangeTerm || clusterChangeTerm < -1
    }
    $scope.setClusterChangeTerm = function (clusterChangeTerm) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('set-cluster-change-term', clusterChangeTerm)
      }, 1000)
    }

    $scope.logs = []
    Socket.emit('get-logs', JSON.stringify({ StartIndex: 0, EndIndex: -1 }))
    Socket.on('logs-info', function (newLogs) {
      console.log(newLogs)
      $scope.logs = newLogs.map(x => JSON.stringify(x, null, 2))
      $scope.$apply()
    })

    $scope.logDeleteCount = 1
    $scope.logDeleteCountInvalid = function (logDeleteCount) {
      return Math.round(logDeleteCount) !== logDeleteCount || logDeleteCount <= 0
    }
    $scope.deleteFromLog = function (logDeleteCount) {
      $scope.waitingResponse = true
      setTimeout(function () {
        Socket.emit('remove-logs', logDeleteCount)
      }, 1000)
    }

    setInterval(() => {
      if (!$scope.executorRunning) return
      // Socket.emit('get-executor-is-running')
      Socket.emit('get-node-state')
      Socket.emit('get-node-address')
      Socket.emit('get-current-term')
      Socket.emit('get-voted-for')
      Socket.emit('get-vote-count')
      Socket.emit('get-commit-index')
      Socket.emit('get-last-applied')
      Socket.emit('get-peer-addresses')
      Socket.emit('get-next-indexes')
      Socket.emit('get-match-indexes')
      Socket.emit('get-cluster-change-index')
      Socket.emit('get-cluster-change-term')
      Socket.emit('get-logs', JSON.stringify({ StartIndex: 0, EndIndex: -1 }))
    }, 1000)

  })

}());