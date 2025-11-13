package cluster

import (
	"context"
	pool "github.com/jolestar/go-commons-pool/v2"
	"go_redis/config"
	database2 "go_redis/database"
	"go_redis/interface/database"
	"go_redis/interface/resp"
	"go_redis/lib/consistenthash"
	"go_redis/lib/logger"
	"go_redis/resp/reply"
	"strings"
)

type ClusterDatabase struct {
	self           string
	nodes          []string
	peerPicker     *consistenthash.NodeMap
	peerConnection map[string]*pool.ObjectPool
	db             database.Database
}

func MakeClusterDatabase() *ClusterDatabase {
	cluster := ClusterDatabase{
		self:           config.Properties.Self,
		db:             database2.NewStandaloneDatabase(),
		peerPicker:     consistenthash.NewNodeMap(nil),
		peerConnection: make(map[string]*pool.ObjectPool),
	}
	nodes := make([]string, 0, len(config.Properties.Peers)+1)
	for _, peer := range config.Properties.Peers {
		nodes = append(nodes, peer)
	}
	nodes = append(nodes, config.Properties.Self)
	cluster.peerPicker.AddNodes(nodes...)
	ctx := context.Background()
	for _, node := range config.Properties.Peers {
		cluster.peerConnection[node] = pool.NewObjectPoolWithDefaultConfig(ctx, &connectionFactory{Peer: node})
	}
	cluster.nodes = nodes
	return &cluster
}

type CmdFunc func(clusterDatabase *ClusterDatabase, c resp.Connection, cmdArgs [][]byte) resp.Reply

var router = makeRouter()

func (cluster *ClusterDatabase) Exec(client resp.Connection, args [][]byte) (result resp.Reply) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(r)
			result = reply.UnknowErrReply{}
		}
	}()

	CmdName := strings.ToLower(string(args[0]))
	if cmdFunc, ok := router[CmdName]; ok {
		return cmdFunc(cluster, client, args)
	} else {
		return reply.MakeErrReply("ERR unknown command '" + CmdName + "'")
	}
}

func (cluster *ClusterDatabase) Close() {
	cluster.db.Close()
}

func (cluster *ClusterDatabase) AfterClientClose(client resp.Connection) {
	cluster.db.AfterClientClose(client)
}
