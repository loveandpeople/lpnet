package gossip

import (
	"fmt"
	"sync"

	"github.com/loveandpeople/lpnet/pkg/model/tangle"
	"github.com/loveandpeople/lpnet/pkg/protocol/helpers"
	"github.com/loveandpeople/hive.go/daemon"
	"github.com/loveandpeople/hive.go/events"
	"github.com/loveandpeople/hive.go/logger"
	"github.com/loveandpeople/hive.go/node"

	"github.com/loveandpeople/lpnet/pkg/config"
	"github.com/loveandpeople/lpnet/pkg/peering"
	"github.com/loveandpeople/lpnet/pkg/peering/peer"
	"github.com/loveandpeople/lpnet/pkg/profile"
	"github.com/loveandpeople/lpnet/pkg/protocol/bqueue"
	"github.com/loveandpeople/lpnet/pkg/protocol/processor"
	"github.com/loveandpeople/lpnet/pkg/protocol/rqueue"
	"github.com/loveandpeople/lpnet/pkg/protocol/sting"
	"github.com/loveandpeople/lpnet/pkg/shutdown"
	peeringplugin "github.com/loveandpeople/lpnet/plugins/peering"
)

var (
	PLUGIN                 = node.NewPlugin("Gossip", node.Enabled, configure, run)
	log                    *logger.Logger
	manager                *peering.Manager
	msgProcessor           *processor.Processor
	msgProcessorOnce       sync.Once
	requestQueue           rqueue.Queue
	requestQueueOnce       sync.Once
	broadcastQueue         bqueue.Queue
	broadcastQueueOnce     sync.Once
	onBroadcastTransaction *events.Closure
)

// RequestQueue returns the request queue instance of the gossip plugin.
func RequestQueue() rqueue.Queue {
	requestQueueOnce.Do(func() {
		requestQueue = rqueue.New()
	})
	return requestQueue
}

// BroadcastQueue returns the broadcast queue instance of the gossip plugin.
func BroadcastQueue() bqueue.Queue {
	broadcastQueueOnce.Do(func() {
		broadcastQueue = bqueue.New(peeringplugin.Manager(), RequestQueue())
	})
	return broadcastQueue
}

// Processor returns the message processor instance of the gossip plugin.
func Processor() *processor.Processor {
	msgProcessorOnce.Do(func() {
		msgProcessor = processor.New(requestQueue, peeringplugin.Manager(), &processor.Options{
			ValidMWM:          config.NodeConfig.GetUint64(config.CfgCoordinatorMWM),
			WorkUnitCacheOpts: profile.LoadProfile().Caches.IncomingTransactionFilter,
		})
	})
	return msgProcessor
}

func configure(plugin *node.Plugin) {
	log = logger.NewLogger(plugin.Name)

	manager = peeringplugin.Manager()

	// create networking queues
	RequestQueue()
	BroadcastQueue()

	// create new message processor
	Processor()

	// handle broadcasts emitted by the message processor
	onBroadcastTransaction = events.NewClosure(broadcastQueue.EnqueueForBroadcast)

	// register event handlers for messages
	manager.Events.PeerConnected.Attach(events.NewClosure(func(p *peer.Peer) {

		if p.Protocol.Supports(sting.FeatureSet) {
			addSTINGMessageEventHandlers(p)

			// send heartbeat and latest milestone request
			if snapshotInfo := tangle.GetSnapshotInfo(); snapshotInfo != nil {
				connected, synced := manager.ConnectedAndSyncedPeerCount()
				helpers.SendHeartbeat(p, tangle.GetSolidMilestoneIndex(), snapshotInfo.PruningIndex, tangle.GetLatestMilestoneIndex(), connected, synced)
				helpers.SendLatestMilestoneRequest(p)
			}
		}

		disconnectSignal := make(chan struct{})
		p.Conn.Events.Close.Attach(events.NewClosure(func() {
			removeMessageEventHandlers(p)
			close(disconnectSignal)
		}))

		// fire up send queue consumer
		daemon.BackgroundWorker(fmt.Sprintf("send queue %s", p.ID), func(shutdownSignal <-chan struct{}) {
			for {
				select {
				case <-disconnectSignal:
					return
				case <-shutdownSignal:
					return
				case data := <-p.SendQueue:
					if err := p.Protocol.Send(data); err != nil {
						p.Protocol.Events.Error.Trigger(err)
					}
				}
			}
		}, shutdown.PriorityPeerSendQueue)
	}))
}

func run(_ *node.Plugin) {

	daemon.BackgroundWorker("BroadcastQueue", func(shutdownSignal <-chan struct{}) {
		log.Info("Running BroadcastQueue")
		broadcastQueue.Run(shutdownSignal)
		log.Info("Stopped BroadcastQueue")
	}, shutdown.PriorityBroadcastQueue)

	daemon.BackgroundWorker("MessageProcessor", func(shutdownSignal <-chan struct{}) {
		log.Info("Running MessageProcessor")
		msgProcessor.Events.BroadcastTransaction.Attach(onBroadcastTransaction)
		msgProcessor.Run(shutdownSignal)
		msgProcessor.Events.BroadcastTransaction.Detach(onBroadcastTransaction)
		log.Info("Stopped MessageProcessor")
	}, shutdown.PriorityMessageProcessor)

	runRequestWorkers()
}
