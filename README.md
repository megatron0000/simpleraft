# Features

Simpleraft implements:
- Leader election
- Log replication
- Cluster membership changes

# Architecture

<pre>                                            
    +--------------------------------------------+                           +-------------------------------------+
    |                                            |                           |                                     |
    |               RuleHandler                  |                           |       State Machine                 |
    |                                            |     (2) respond           |                                     |
    |     Given a message, calculates the        |     with action           | Application+specific state machine. |
    |     appropriate response and               |                           | Example: For a key+> value store,   |
    |     returns it.                            |  +----+                   | this component would implement the  |
    |     Has no knowledge of implementation+    |       |                   | store. Raft commands passed to here |
    |     specific components                    |       |                   | would be Get()s and Put()s          |
    |                                            |       |                   |                                     |
    +-------------------+------------------------+       |                   +---------------------------+---------+
                        ^                                |￼                                              ^
           (1) ask for  |                                v                                               |  commit
           action       +------------+                                                                   |
                                           +---------------------------------------+   +-----------------+
                                           |               Executor                |
                                           |                                       |
                  +---------------------+  | Implements message sending/receiving, +----------------+
                  |                        | timeouts, state changes and log       |                |
                  |                        | write/read                            |                |
                  |   use                  |                                       |                |     use
                  |                        +-------------------+-------------------+                |
                  |                                            |                                    |
                  |                                            |  use                               |
                  v                                            v                                    v
                                            +------------------+-------------------+
+---------------------------------------+   |       TransportService               |  +---------------------------------------+
|         StatusController              |   |                                      |  |         LogController                 |
|                                       |   |                                      |  |                                       |
|  Stores state+specific information:   |   |   Implements message passing between |  |   Appends (or changes) entries        |
|  currentTerm, votedFor, commitIndex,  |   |   Raft nodes or between a client and |  |   to the log, saves the log to disk,  |
|  lastApplied                          |   |   a Raft node                        |  |   and reads the log.                  |
+---------------------------------------+   +--------------------------------------+  +--------+------------------------------+
                                                                                               |
                       +                                                                       |
                       |                                                                       |
                       |                                                                       |
                       |                       +---------------------------+                   |
                       |                       |    Storage Service        |                   |
                       |                       |                           |                   |
                       +------------------->   |                           | <-----------------+
                                               |  Implements the           |
                      use                      |  save+to+disk mechanism   |          use
                                               |                           |
                                               +---------------------------+

</pre>