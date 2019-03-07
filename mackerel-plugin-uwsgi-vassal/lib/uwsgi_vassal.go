package mpuwsgivassal

import (
	"encoding/json"
	"errors"
	"flag"
	"net"
	"net/http"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

// UWSGIVassalPlugin mackerel plugin for uWSGI
type UWSGIVassalPlugin struct {
	Socket      string
	Prefix      string
	LabelPrefix string
}

// {
//   "workers": [{
//     "id": 1,
//     "pid": 31759,
//     "requests": 0,
//     "exceptions": 0,
//     "status": "idle",
//     "rss": 0,
//     "vsz": 0,
//     "running_time": 0,
//     "last_spawn": 1317235041,
//     "respawn_count": 1,
//     "tx": 0,
//     "avg_rt": 0,
//     "apps": [{
//       "id": 0,
//       "modifier1": 0,
//       "mountpoint": "",
//       "requests": 0,
//       "exceptions": 0,
//       "chdir": ""
//     }]
//   }, {
//     "id": 2,
//     "pid": 31760,
//     "requests": 0,
//     "exceptions": 0,
//     "status": "idle",
//     "rss": 0,
//     "vsz": 0,
//     "running_time": 0,
//     "last_spawn": 1317235041,
//     "respawn_count": 1,
//     "tx": 0,
//     "avg_rt": 0,
//     "apps": [{
//       "id": 0,
//       "modifier1": 0,
//       "mountpoint": "",
//       "requests": 0,
//       "exceptions": 0,
//       "chdir": ""
//     }]
//   }, {
//     "id": 3,
//     "pid": 31761,
//     "requests": 0,
//     "exceptions": 0,
//     "status": "idle",
//     "rss": 0,
//     "vsz": 0,
//     "running_time": 0,
//     "last_spawn": 1317235041,
//     "respawn_count": 1,
//     "tx": 0,
//     "avg_rt": 0,
//     "apps": [{
//       "id": 0,
//       "modifier1": 0,
//       "mountpoint": "",
//       "requests": 0,
//       "exceptions": 0,
//       "chdir": ""
//     }]
//   }, {
//     "id": 4,
//     "pid": 31762,
//     "requests": 0,
//     "exceptions": 0,
//     "status": "idle",
//     "rss": 0,
//     "vsz": 0,
//     "running_time": 0,
//     "last_spawn": 1317235041,
//     "respawn_count": 1,
//     "tx": 0,
//     "avg_rt": 0,
//     "apps": [{
//       "id": 0,
//       "modifier1": 0,
//       "mountpoint": "",
//       "requests": 0,
//       "exceptions": 0,
//       "chdir": ""
//     }]
//   }
// }

// field types vary between versions

// UWSGIWorker struct
type UWSGIWorker struct {
	Requests       uint64 `json:"requests"`
	Status         string `json:"status"`
	Rss            uint64 `json:"rss"`
	Vsz            uint64 `json:"vsz"`
	Tx             uint64 `json:"tx"`
	AvgRequestTime uint64 `json:"avg_rt"`
	HarakiriCount  uint64 `json:"harakiri_count"`
	RespawnCount   uint64 `json:"respawn_count"`
}

// UWSGIWorkers sturct for json struct
type UWSGIWorkers struct {
	Workers []UWSGIWorker `json:"workers"`
}

// FetchMetrics interface for mackerelplugin
func (p UWSGIVassalPlugin) FetchMetrics() (map[string]float64, error) {
	stat := make(map[string]float64)
	stat["busy"] = 0.0
	stat["idle"] = 0.0
	stat["cheap"] = 0.0
	stat["pause"] = 0.0
	stat["requests"] = 0.0
	stat["rss"] = 0.0
	stat["vsz"] = 0.0
	stat["avg_rt"] = 0.0

	var decoder *json.Decoder
	if strings.HasPrefix(p.Socket, "unix://") {
		conn, err := net.Dial("unix", strings.TrimPrefix(p.Socket, "unix://"))
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		decoder = json.NewDecoder(conn)
	} else if strings.HasPrefix(p.Socket, "http://") {
		req, err := http.NewRequest(http.MethodGet, p.Socket, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "mackerel-plugin-uwsgi-vessal")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		decoder = json.NewDecoder(resp.Body)
	} else {
		err := errors.New("'--socket' is neither http endpoint nor the unix domain socket, try '--help' for more information")
		return nil, err
	}

	var workers UWSGIWorkers
	if err := decoder.Decode(&workers); err != nil {
		return nil, err
	}

	for _, worker := range workers.Workers {
		switch worker.Status {
		case "idle", "busy", "cheap", "pause":
			stat[worker.Status]++
		}
		stat["requests"] += float64(worker.Requests)
		stat["rss"] += float64(worker.Rss)
		stat["vsz"] += float64(worker.Vsz)
		stat["tx"] += float64(worker.Tx)
		stat["avg_rt"] += float64(worker.AvgRequestTime)
		stat["harakiri_count"] += float64(worker.HarakiriCount)
		stat["respawn_count"] += float64(worker.RespawnCount)
	}
	if len(workers.Workers) > 0 {
		stat["avg_rt"] = stat["avg_rt"] / float64(len(workers.Workers))
	}

	return stat, nil
}

// GraphDefinition interface for mackerelplugin
func (p UWSGIVassalPlugin) GraphDefinition() map[string]mp.Graphs {
	labelPrefix := strings.Title(p.Prefix)

	var graphdef = map[string]mp.Graphs{
		(p.Prefix + ".workers"): {
			Label: labelPrefix + " Workers",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "busy", Label: "Busy", Diff: false, Stacked: true},
				{Name: "idle", Label: "Idle", Diff: false, Stacked: true},
				{Name: "cheap", Label: "Cheap", Diff: false, Stacked: true},
				{Name: "pause", Label: "Pause", Diff: false, Stacked: true},
			},
		},
		(p.Prefix + ".req"): {
			Label: labelPrefix + " Requests",
			Unit:  "float",
			Metrics: []mp.Metrics{
				{Name: "requests", Label: "Requests", Diff: true},
			},
		},
		(p.Prefix + ".memory"): {
			Label: labelPrefix + " Memory",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "rss", Label: "RSS", Diff: false},
				{Name: "vsz", Label: "VSZ", Diff: false},
			},
		},
		(p.Prefix + ".network"): {
			Label: labelPrefix + " Network",
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "tx", Label: "TX", Diff: true},
			},
		},
		(p.Prefix + ".reqtime"): {
			Label: labelPrefix + " RequestsTime",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "avg_rt", Label: "Average[us]", Diff: false},
			},
		},
		(p.Prefix + ".counter"): {
			Label: labelPrefix + " Counter",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "harakiri_count", Label: "Harakiri", Diff: false},
				{Name: "respawn_count", Label: "Respawn", Diff: false},
			},
		},
	}

	return graphdef
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p UWSGIVassalPlugin) MetricKeyPrefix() string {
	if p.Prefix == "" {
		p.Prefix = "uWSGI"
	}
	return p.Prefix
}

// Do the plugin
func Do() {
	optSocket := flag.String("socket", "", "Socket (must be with prefix of 'http://' or 'unix://')")
	optPrefix := flag.String("metric-key-prefix", "uWSGI", "Prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	uwsgi := UWSGIVassalPlugin{Socket: *optSocket, Prefix: *optPrefix}
	uwsgi.LabelPrefix = strings.Title(uwsgi.Prefix)

	helper := mp.NewMackerelPlugin(uwsgi)
	helper.Tempfile = *optTempfile
	helper.Run()
}
