package discv5

import "github.com/wanchain/go-wanchain/metrics"

var (
	ingressTrafficMeter = metrics.NewMeter("discv5/InboundTraffic")
	egressTrafficMeter  = metrics.NewMeter("discv5/OutboundTraffic")
)
