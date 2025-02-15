package bgp

import (
	"github.com/czerwonk/junos_exporter/collector"
	"github.com/czerwonk/junos_exporter/rpc"
	"github.com/prometheus/client_golang/prometheus"

	"strings"
)

const prefix string = "junos_bgp_session_"

var (
	upDesc                 *prometheus.Desc
	receivedPrefixesDesc   *prometheus.Desc
	acceptedPrefixesDesc   *prometheus.Desc
	rejectedPrefixesDesc   *prometheus.Desc
	activePrefixesDesc     *prometheus.Desc
	advertisedPrefixesDesc *prometheus.Desc
	inputMessagesDesc      *prometheus.Desc
	outputMessagesDesc     *prometheus.Desc
	flapsDesc              *prometheus.Desc
)

func init() {
	l := []string{"target", "asn", "ip", "description"}
	upDesc = prometheus.NewDesc(prefix+"up", "Session is up (1 = Established)", l, nil)
	inputMessagesDesc = prometheus.NewDesc(prefix+"messages_input_count", "Number of received messages", l, nil)
	outputMessagesDesc = prometheus.NewDesc(prefix+"messages_output_count", "Number of transmitted messages", l, nil)
	flapsDesc = prometheus.NewDesc(prefix+"flap_count", "Number of session flaps", l, nil)

	l = append(l, "table")
	receivedPrefixesDesc = prometheus.NewDesc(prefix+"prefixes_received_count", "Number of received prefixes", l, nil)
	acceptedPrefixesDesc = prometheus.NewDesc(prefix+"prefixes_accepted_count", "Number of accepted prefixes", l, nil)
	rejectedPrefixesDesc = prometheus.NewDesc(prefix+"prefixes_rejected_count", "Number of rejected prefixes", l, nil)
	activePrefixesDesc = prometheus.NewDesc(prefix+"prefixes_active_count", "Number of active prefixes (best route in RIB)", l, nil)
	advertisedPrefixesDesc = prometheus.NewDesc(prefix+"prefixes_advertised_count", "Number of prefixes announced to peer", l, nil)
}

type bgpCollector struct {
}

// NewCollector creates a new collector
func NewCollector() collector.RPCCollector {
	return &bgpCollector{}
}

// Describe describes the metrics
func (*bgpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- receivedPrefixesDesc
	ch <- acceptedPrefixesDesc
	ch <- rejectedPrefixesDesc
	ch <- activePrefixesDesc
	ch <- advertisedPrefixesDesc
	ch <- inputMessagesDesc
	ch <- outputMessagesDesc
	ch <- flapsDesc
}

// Collect collects metrics from JunOS
func (c *bgpCollector) Collect(client *rpc.Client, ch chan<- prometheus.Metric, labelValues []string) error {
	err := c.collect(client, ch, labelValues)
	if err != nil {
		return err
	}

	return nil
}

func (c *bgpCollector) collect(client *rpc.Client, ch chan<- prometheus.Metric, labelValues []string) error {
	var x = BGPRPC{}
	err := client.RunCommandAndParse("show bgp neighbor", &x)
	if err != nil {
		return err
	}

	for _, peer := range x.Information.Peers {
		c.collectForPeer(peer, ch, labelValues)
	}

	return nil
}

func (c *bgpCollector) collectForPeer(p BGPPeer, ch chan<- prometheus.Metric, labelValues []string) {
	ip := strings.Split(p.IP, "+")
	l := append(labelValues, []string{p.ASN, ip[0], p.Description}...)

	up := 0
	if p.State == "Established" {
		up = 1
	}

	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, float64(up), l...)
	ch <- prometheus.MustNewConstMetric(inputMessagesDesc, prometheus.GaugeValue, float64(p.InputMessages), l...)
	ch <- prometheus.MustNewConstMetric(outputMessagesDesc, prometheus.GaugeValue, float64(p.OutputMessages), l...)
	ch <- prometheus.MustNewConstMetric(flapsDesc, prometheus.GaugeValue, float64(p.Flaps), l...)

	c.collectRIBForPeer(p, ch, l)
}

func (*bgpCollector) collectRIBForPeer(p BGPPeer, ch chan<- prometheus.Metric, labelValues []string) {
	l := append(labelValues, p.RIB.Name)
	ch <- prometheus.MustNewConstMetric(receivedPrefixesDesc, prometheus.GaugeValue, float64(p.RIB.ReceivedPrefixes), l...)
	ch <- prometheus.MustNewConstMetric(acceptedPrefixesDesc, prometheus.GaugeValue, float64(p.RIB.AcceptedPrefixes), l...)
	ch <- prometheus.MustNewConstMetric(rejectedPrefixesDesc, prometheus.GaugeValue, float64(p.RIB.RejectedPrefixes), l...)
	ch <- prometheus.MustNewConstMetric(activePrefixesDesc, prometheus.GaugeValue, float64(p.RIB.ActivePrefixes), l...)
	ch <- prometheus.MustNewConstMetric(advertisedPrefixesDesc, prometheus.GaugeValue, float64(p.RIB.AdvertisedPrefixes), l...)
}
