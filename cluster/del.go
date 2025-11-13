package cluster

import (
	"go_redis/interface/resp"
	"go_redis/resp/reply"
)

// del k1 k2 ...
func Del(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply {
	replies := clusterDatabase.boardcast(c, cmdArgs)
	var errReply resp.Reply
	var successCount int64 = 0
	for _, r := range replies {
		if reply.IsErrReply(r) {
			errReply = r.(reply.ErrorReply)
			break
		}
		intReply, ok := r.(*reply.IntReply)
		if !ok {
			errReply = reply.MakeErrReply("ERR Del command failed")
			break
		}
		successCount += intReply.Code
	}
	if errReply != nil {
		return errReply
	}
	return reply.MakeIntReply(successCount)
}
