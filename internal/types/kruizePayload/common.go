package kruizePayload

import "fmt"

type kubernetesObject struct {
	K8stype    string      `json:"type,omitempty"`
	Name       string      `json:"name,omitempty"`
	Namespace  string      `json:"namespace,omitempty"`
	Containers []container `json:"containers,omitempty"`
}

type container struct {
	Container_image_name string   `json:"container_image_name,omitempty"`
	Container_name       string   `json:"container_name,omitempty"`
	Metrics              []metric `json:"metrics,omitempty"`
}

type metric struct {
	Name    string `json:"name,omitempty"`
	Results result `json:"results,omitempty"`
}

type result struct {
	Aggregation_info aggregation_info `json:"aggregation_info,omitempty"`
}

type aggregation_info struct {
	Min    string `json:"min,omitempty"`
	Max    string `json:"max,omitempty"`
	Sum    string `json:"sum,omitempty"`
	Avg    string `json:"avg,omitempty"`
	Format string `json:"format,omitempty"`
}

func convertMetricToString(data interface{}) string {
	if metric, ok := data.(float64); ok {
		return fmt.Sprintf("%.2f", metric)
	} else {
		return "-1"
	}
}

func make_container_data(c map[string]interface{}) container {
	container_data := container{
		Container_image_name: c["image_name"].(string),
		Container_name:       c["container_name"].(string),
		Metrics: []metric{
			{
				Name: "cpuRequest",
				Results: result{
					Aggregation_info: aggregation_info{
						Sum:    convertMetricToString(c["cpu_request_container_sum_SUM"]),
						Avg:    convertMetricToString(c["cpu_request_container_avg_MEAN"]),
						Format: "cores",
					},
				},
			},
			{
				Name: "cpuLimit",
				Results: result{
					Aggregation_info: aggregation_info{
						Sum:    convertMetricToString(c["cpu_limit_container_sum_SUM"]),
						Avg:    convertMetricToString(c["cpu_limit_container_avg_MEAN"]),
						Format: "cores",
					},
				},
			},
			{
				Name: "cpuUsage",
				Results: result{
					Aggregation_info: aggregation_info{
						Min:    convertMetricToString(c["cpu_usage_container_min_MIN"]),
						Max:    convertMetricToString(c["cpu_usage_container_max_MAX"]),
						Sum:    convertMetricToString(c["cpu_usage_container_sum_SUM"]),
						Avg:    convertMetricToString(c["cpu_usage_container_avg_MEAN"]),
						Format: "cores",
					},
				},
			},
			{
				Name: "cpuThrottle",
				Results: result{
					Aggregation_info: aggregation_info{
						Max:    convertMetricToString(c["cpu_throttle_container_max_MAX"]),
						Sum:    convertMetricToString(c["cpu_throttle_container_sum_SUM"]),
						Avg:    convertMetricToString(c["cpu_throttle_container_avg_MEAN"]),
						Format: "cores",
					},
				},
			},
			{
				Name: "memoryRequest",
				Results: result{
					Aggregation_info: aggregation_info{
						Sum:    convertMetricToString(c["memory_request_container_sum_SUM"]),
						Avg:    convertMetricToString(c["memory_request_container_avg_MEAN"]),
						Format: "MiB",
					},
				},
			},
			{
				Name: "memoryLimit",
				Results: result{
					Aggregation_info: aggregation_info{
						Sum:    convertMetricToString(c["memory_limit_container_sum_SUM"]),
						Avg:    convertMetricToString(c["memory_limit_container_avg_MEAN"]),
						Format: "MiB",
					},
				},
			},
			{
				Name: "memoryUsage",
				Results: result{
					Aggregation_info: aggregation_info{
						Min:    convertMetricToString(c["memory_usage_container_min_MIN"]),
						Max:    convertMetricToString(c["memory_usage_container_max_MAX"]),
						Sum:    convertMetricToString(c["memory_usage_container_sum_SUM"]),
						Avg:    convertMetricToString(c["memory_usage_container_avg_MEAN"]),
						Format: "MiB",
					},
				},
			},
			{
				Name: "memoryRSS",
				Results: result{
					Aggregation_info: aggregation_info{
						Min:    convertMetricToString(c["memory_rss_usage_container_min_MIN"]),
						Max:    convertMetricToString(c["memory_rss_usage_container_max_MAX"]),
						Sum:    convertMetricToString(c["memory_rss_usage_container_sum_SUM"]),
						Avg:    convertMetricToString(c["memory_rss_usage_container_avg_MEAN"]),
						Format: "MiB",
					},
				},
			},
		},
	}

	return container_data
}
