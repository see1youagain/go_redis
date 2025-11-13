package cluster

import (
	"go_redis/interface/resp"
	"go_redis/resp/reply"
)

//type CmdFunc func(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply

func flushdb(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply {
	replies := clusterDatabase.boardcast(c, cmdArgs)
	var errReply resp.Reply
	for _, r := range replies {
		if reply.IsErrReply(r) {
			errReply = r.(reply.ErrorReply)
			break
		}
	}
	if errReply != nil {
		return errReply
	}
	return reply.MakeOkReply()
}
