package dashboard

import (
	"time"

	"github.com/loveandpeople/hive.go/daemon"
	"github.com/loveandpeople/hive.go/events"
	"github.com/loveandpeople/hive.go/workerpool"
	"github.com/loveandpeople/iota.go/transaction"

	"github.com/loveandpeople/lpnet/pkg/model/hornet"
	"github.com/loveandpeople/lpnet/pkg/model/milestone"
	tanglemodel "github.com/loveandpeople/lpnet/pkg/model/tangle"
	"github.com/loveandpeople/lpnet/pkg/shutdown"
	"github.com/loveandpeople/lpnet/plugins/tangle"
)

var (
	liveFeedWorkerCount     = 1
	liveFeedWorkerQueueSize = 50
	liveFeedWorkerPool      *workerpool.WorkerPool
)

func configureLiveFeed() {
	liveFeedWorkerPool = workerpool.New(func(task workerpool.Task) {
		switch x := task.Param(0).(type) {
		case *transaction.Transaction:
			if x.Value == 0 {
				hub.BroadcastMsg(&msg{MsgTypeTxZeroValue, &tx{x.Hash, x.Value}})
			} else {
				hub.BroadcastMsg(&msg{MsgTypeTxValue, &tx{x.Hash, x.Value}})
			}
		case milestone.Index:
			if msTailTxHash := getMilestoneTailHash(x); msTailTxHash != nil {
				hub.BroadcastMsg(&msg{MsgTypeMs, &ms{msTailTxHash.Trytes(), x}})
			}
		}
		task.Return(nil)
	}, workerpool.WorkerCount(liveFeedWorkerCount), workerpool.QueueSize(liveFeedWorkerQueueSize))
}

func runLiveFeed() {

	newTxRateLimiter := time.NewTicker(time.Second / 10)

	onReceivedNewTransaction := events.NewClosure(func(cachedTx *tanglemodel.CachedTransaction, latestMilestoneIndex milestone.Index, latestSolidMilestoneIndex milestone.Index) {
		cachedTx.ConsumeTransaction(func(tx *hornet.Transaction) {
			if !tanglemodel.IsNodeSyncedWithThreshold() {
				return
			}
			select {
			case <-newTxRateLimiter.C:
				liveFeedWorkerPool.TrySubmit(tx.Tx)
			default:
			}
		})
	})

	onLatestMilestoneIndexChanged := events.NewClosure(func(msIndex milestone.Index) {
		liveFeedWorkerPool.TrySubmit(msIndex)
	})

	daemon.BackgroundWorker("Dashboard[TxUpdater]", func(shutdownSignal <-chan struct{}) {
		tangle.Events.ReceivedNewTransaction.Attach(onReceivedNewTransaction)
		defer tangle.Events.ReceivedNewTransaction.Detach(onReceivedNewTransaction)
		tangle.Events.LatestMilestoneIndexChanged.Attach(onLatestMilestoneIndexChanged)
		defer tangle.Events.LatestMilestoneIndexChanged.Detach(onLatestMilestoneIndexChanged)

		liveFeedWorkerPool.Start()
		<-shutdownSignal

		log.Info("Stopping Dashboard[TxUpdater] ...")
		newTxRateLimiter.Stop()
		liveFeedWorkerPool.StopAndWait()
		log.Info("Stopping Dashboard[TxUpdater] ... done")
	}, shutdown.PriorityDashboard)
}
