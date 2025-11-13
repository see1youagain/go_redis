package cluster

import "go_redis/interface/resp"

//type CmdFunc func(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply

func makeRouter() map[string]CmdFunc {
	router := make(map[string]CmdFunc)
	router["exists"] = defaultFunc
	router["del"] = defaultFunc
	router["get"] = defaultFunc
	router["type"] = defaultFunc
	router["set"] = defaultFunc
	router["setnx"] = defaultFunc
	router["getset"] = defaultFunc
	router["ping"] = ping
	router["rename"] = Rename
	router["renamenx"] = Rename
	router["flushdb"] = flushdb
	router["del"] = Del
	router["select"] = execSelect
	return router
}

func defaultFunc(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply {
	key := string(cmdArgs[1])
	peer := clusterDatabase.peerPicker.PickNode(key)
	return clusterDatabase.relay(peer, c, cmdArgs)
}
