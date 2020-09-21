package dashboard

import (
	"github.com/loveandpeople/hive.go/daemon"
	"github.com/loveandpeople/hive.go/events"
	"github.com/loveandpeople/hive.go/workerpool"

	"github.com/loveandpeople/lpnet/pkg/shutdown"
	"github.com/loveandpeople/lpnet/pkg/spammer"
	spammerplugin "github.com/loveandpeople/lpnet/plugins/spammer"
)

var (
	spammerMetricWorkerCount     = 1
	spammerMetricWorkerQueueSize = 100
	spammerMetricWorkerPool      *workerpool.WorkerPool
)

func configureSpammerMetric() {
	spammerMetricWorkerPool = workerpool.New(func(task workerpool.Task) {
		hub.BroadcastMsg(task.Param(0))
		task.Return(nil)
	}, workerpool.WorkerCount(spammerMetricWorkerCount), workerpool.QueueSize(spammerMetricWorkerQueueSize))
}

func runSpammerMetricWorker() {

	onSpamPerformed := events.NewClosure(func(metrics *spammer.SpamStats) {
		spammerMetricWorkerPool.TrySubmit(&msg{Type: MsgTypeSpamMetrics, Data: metrics})
	})

	onAvgSpamMetricsUpdated := events.NewClosure(func(metrics *spammer.AvgSpamMetrics) {
		spammerMetricWorkerPool.TrySubmit(&msg{Type: MsgTypeAvgSpamMetrics, Data: metrics})
	})

	daemon.BackgroundWorker("Dashboard[SpammerMetricUpdater]", func(shutdownSignal <-chan struct{}) {
		spammerplugin.Events.SpamPerformed.Attach(onSpamPerformed)
		spammerplugin.Events.AvgSpamMetricsUpdated.Attach(onAvgSpamMetricsUpdated)
		spammerMetricWorkerPool.Start()
		<-shutdownSignal
		log.Info("Stopping Dashboard[SpammerMetricUpdater] ...")
		spammerplugin.Events.SpamPerformed.Detach(onSpamPerformed)
		spammerplugin.Events.AvgSpamMetricsUpdated.Detach(onAvgSpamMetricsUpdated)
		spammerMetricWorkerPool.StopAndWait()
		log.Info("Stopping Dashboard[SpammerMetricUpdater] ... done")
	}, shutdown.PriorityDashboard)
}
