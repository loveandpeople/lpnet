package metrics

import (
	"github.com/loveandpeople/lpnet/pkg/metrics"
	"github.com/loveandpeople/lpnet/pkg/utils"
)

var (
	lastIncomingTxCnt    uint32
	lastIncomingNewTxCnt uint32
	lastOutgoingTxCnt    uint32
)

// measures the TPS values
func measureTPS() {
	incomingTxCnt := metrics.SharedServerMetrics.Transactions.Load()
	incomingNewTxCnt := metrics.SharedServerMetrics.NewTransactions.Load()
	outgoingTxCnt := metrics.SharedServerMetrics.SentTransactions.Load()

	tpsMetrics := &TPSMetrics{
		Incoming: utils.GetUint32Diff(incomingTxCnt, lastIncomingTxCnt),
		New:      utils.GetUint32Diff(incomingNewTxCnt, lastIncomingNewTxCnt),
		Outgoing: utils.GetUint32Diff(outgoingTxCnt, lastOutgoingTxCnt),
	}

	// store the new counters
	lastIncomingTxCnt = incomingTxCnt
	lastIncomingNewTxCnt = incomingNewTxCnt
	lastOutgoingTxCnt = outgoingTxCnt

	// trigger events for outside listeners
	Events.TPSMetricsUpdated.Trigger(tpsMetrics)
}
