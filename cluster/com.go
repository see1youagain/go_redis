package cluster

import (
	"context"
	"errors"
	"go_redis/interface/resp"
	"go_redis/lib/utils"
	"go_redis/resp/client"
	"go_redis/resp/reply"
	"strconv"
)

// 通信文件

func (cluster *ClusterDatabase) getPeerClient(peer string) (*client.Client, error) {
	pool, ok := cluster.peerConnection[peer]
	if !ok {
		return nil, errors.New("getPeerClient connection not found")
	}
	object, err := pool.BorrowObject(context.Background())
	if err != nil {
		return nil, err
	}
	c, ok := object.(*client.Client)
	if !ok {
		return nil, errors.New("getPeerClient type assert failed")
	}
	return c, nil
}

func (cluster *ClusterDatabase) returnPeerClient(peer string, c *client.Client) error {
	pool, ok := cluster.peerConnection[peer]
	if !ok {
		return errors.New("returnPeerClient connection not found")
	}
	return pool.ReturnObject(context.Background(), c)
}

// 转发请求
func (cluster *ClusterDatabase) relay(peer string, c resp.Connection, args [][]byte) resp.Reply {
	if peer == cluster.self {
		return cluster.db.Exec(c, args)
	}
	peerClient, err := cluster.getPeerClient(peer)
	if err != nil {
		return reply.MakeErrReply("relay failed" + err.Error())
	}
	defer func() {
		_ = cluster.returnPeerClient(peer, peerClient)
	}()
	// 需要手动将select db发给peers
	peerClient.Send(utils.ToCmdLine2("SELECT", strconv.Itoa(c.GetDBIndex())))
	return peerClient.Send(args)
}

func (cluster *ClusterDatabase) boardcast(c resp.Connection, args [][]byte) map[string]resp.Reply {
	results := make(map[string]resp.Reply)
	for _, peer := range cluster.nodes {
		results[peer] = cluster.relay(peer, c, args)
	}
	return results
}
