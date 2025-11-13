package cluster

import "go_redis/interface/resp"

//type CmdFunc func(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply

func execSelect(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply {
	return clusterDatabase.db.Exec(c, cmdArgs)
}
