package main

import (
	"github.com/loveandpeople/hive.go/node"

	"github.com/loveandpeople/lpnet/pkg/config"
	"github.com/loveandpeople/lpnet/pkg/toolset"
	"github.com/loveandpeople/lpnet/plugins/autopeering"
	"github.com/loveandpeople/lpnet/plugins/cli"
	"github.com/loveandpeople/lpnet/plugins/coordinator"
	"github.com/loveandpeople/lpnet/plugins/dashboard"
	"github.com/loveandpeople/lpnet/plugins/database"
	"github.com/loveandpeople/lpnet/plugins/gossip"
	"github.com/loveandpeople/lpnet/plugins/gracefulshutdown"
	"github.com/loveandpeople/lpnet/plugins/metrics"
	"github.com/loveandpeople/lpnet/plugins/mqtt"
	"github.com/loveandpeople/lpnet/plugins/peering"
	"github.com/loveandpeople/lpnet/plugins/pow"
	"github.com/loveandpeople/lpnet/plugins/profiling"
	"github.com/loveandpeople/lpnet/plugins/prometheus"
	"github.com/loveandpeople/lpnet/plugins/snapshot"
	"github.com/loveandpeople/lpnet/plugins/spammer"
	"github.com/loveandpeople/lpnet/plugins/tangle"
	"github.com/loveandpeople/lpnet/plugins/urts"
	"github.com/loveandpeople/lpnet/plugins/warpsync"
	"github.com/loveandpeople/lpnet/plugins/webapi"
	"github.com/loveandpeople/lpnet/plugins/zmq"
)

func main() {
	cli.HideConfigFlags()
	cli.PrintVersion()
	cli.ParseConfig()
	toolset.HandleTools()
	cli.PrintConfig()

	plugins := []*node.Plugin{
		cli.PLUGIN,
		gracefulshutdown.PLUGIN,
		profiling.PLUGIN,
		database.PLUGIN,
		autopeering.PLUGIN,
		webapi.PLUGIN,
	}

	if !config.NodeConfig.GetBool(config.CfgNetAutopeeringRunAsEntryNode) {
		plugins = append(plugins, []*node.Plugin{
			pow.PLUGIN,
			gossip.PLUGIN,
			tangle.PLUGIN,
			peering.PLUGIN,
			warpsync.PLUGIN,
			urts.PLUGIN,
			metrics.PLUGIN,
			snapshot.PLUGIN,
			dashboard.PLUGIN,
			zmq.PLUGIN,
			mqtt.PLUGIN,
			spammer.PLUGIN,
			coordinator.PLUGIN,
			prometheus.PLUGIN,
		}...)
	}

	node.Run(node.Plugins(plugins...))
}
