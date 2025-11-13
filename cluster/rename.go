package cluster

import (
	"go_redis/interface/resp"
	"go_redis/resp/reply"
)

// 暂时不考虑节点迁移问题

//type CmdFunc func(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply

func Rename(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply {
	if len(cmdArgs) != 3 {
		return reply.MakeArgNumErrReply("rename")
	}
	old := string(cmdArgs[1])
	cur := string(cmdArgs[2])
	oldPeer := clusterDatabase.peerPicker.PickNode(old)
	curPeer := clusterDatabase.peerPicker.PickNode(cur)
	if oldPeer != curPeer {
		return reply.MakeErrReply("ERR cross slot rename is not allowed")
	}
	return clusterDatabase.relay(cur, c, cmdArgs)
}
