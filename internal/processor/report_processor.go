package processor

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-gota/gota/dataframe"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"

	"github.com/redhatinsights/ros-ocp-backend/internal/config"
	p "github.com/redhatinsights/ros-ocp-backend/internal/kafka"
	"github.com/redhatinsights/ros-ocp-backend/internal/logging"
	"github.com/redhatinsights/ros-ocp-backend/internal/model"
	"github.com/redhatinsights/ros-ocp-backend/internal/types"
	"github.com/redhatinsights/ros-ocp-backend/internal/types/workload"
)

var log *logrus.Logger = logging.GetLogger()
var cfg *config.Config = config.GetConfig()

func ProcessReport(msg *kafka.Message) {
	validate := validator.New()
	var kafkaMsg KafkaMsg
	if !json.Valid([]byte(msg.Value)) {
		log.Errorf("Received message on kafka topic is not vaild JSON: %s", msg.Value)
		return
	}
	if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
		log.Errorf("Unable to decode kafka message: %s", msg.Value)
		return
	}
	if err := validate.Struct(kafkaMsg); err != nil {
		log.Errorf("Invalid kafka message: %s", err)
		return
	}

	// Create user account(if not present) for incoming archive.
	rh_account := model.RHAccount{
		Account: kafkaMsg.Metadata.Account,
		OrgId:   kafkaMsg.Metadata.Org_id,
	}
	if err := rh_account.CreateRHAccount(); err != nil {
		log.Errorf("unable to get or add record to rh_accounts table: %v. Error: %v", rh_account, err)
		return
	}

	// Create cluster record(if not present) for incoming archive.
	cluster := model.Cluster{
		TenantID:       rh_account.ID,
		SourceId:       kafkaMsg.Metadata.Source_id,
		ClusterUUID:    kafkaMsg.Metadata.Cluster_uuid,
		ClusterAlias:   kafkaMsg.Metadata.Cluster_alias,
		LastReportedAt: time.Now(),
	}
	if err := cluster.CreateCluster(); err != nil {
		log.Errorf("unable to get or add record to clusters table: %v. Error: %v", cluster, err)
		return
	}

	for _, file := range kafkaMsg.Files {
		data, err := readCSVFromUrl(file)
		if err != nil {
			log.Errorf("Unable to read CSV from URL. Error: %s", err)
			return
		}
		df := dataframe.LoadRecords(data)
		df = Aggregate_data(df)

		// grouping container(row in csv) by there deployement.
		k8s_object_groups := df.GroupBy("namespace", "k8s_object_type", "k8s_object_name", "interval_end").GetGroups()

		// looping over each group.
		for _, k8s_object_group := range k8s_object_groups {

			k8s_object := k8s_object_group.Maps()
			namespace := k8s_object[0]["namespace"].(string)
			k8s_object_type := k8s_object[0]["k8s_object_type"].(string)
			k8s_object_name := k8s_object[0]["k8s_object_name"].(string)
			interval_start, err := time.Parse("2006-01-02 15:04:05 -0700 MST", k8s_object[0]["interval_start"].(string))
			if err != nil {
				log.Errorf("unable to convert string to time: %s", err)
				continue
			}
			interval_end, err := time.Parse("2006-01-02 15:04:05 -0700 MST", k8s_object[0]["interval_end"].(string))
			if err != nil {
				log.Errorf("unable to convert string to time: %s", err)
				continue
			}

			experiment_name := generateExperimentName(
				kafkaMsg.Metadata.Org_id,
				kafkaMsg.Metadata.Source_id,
				kafkaMsg.Metadata.Cluster_uuid,
				namespace,
				k8s_object_type,
				k8s_object_name,
			)

			container_names, err := create_kruize_experiments(experiment_name, k8s_object)
			if err != nil {
				log.Error(err)
				continue
			}

			// Create workload entry into the table.
			workload := model.Workload{
				ClusterID:       cluster.ID,
				ExperimentName:  experiment_name,
				Namespace:       namespace,
				WorkloadType:    workload.WorkloadType(k8s_object_type),
				WorkloadName:    k8s_object_name,
				Containers:      container_names,
				MetricsUploadAt: time.Now(),
			}
			if err := workload.CreateWorkload(); err != nil {
				log.Errorf("unable to get or add record to workloads table: %v. Error: %v", workload, err)
				return
			}

			usage_data_byte, err := Update_results(experiment_name, k8s_object)
			if err != nil {
				log.Error(err)
				continue
			}

			for _, container := range usage_data_byte[0].Kubernetes_objects[0].Containers {
				container_usage_metrics, err := json.Marshal(container.Metrics)
				if err != nil {
					log.Errorf("Unable to marshal container usage data: %v", err)
				}
				workload_metric := model.WorkloadMetrics{
					WorkloadID:    workload.ID,
					ContainerName: container.Container_name,
					IntervalStart: interval_start,
					IntervalEnd:   interval_end,
					UsageMetrics:  container_usage_metrics,
				}
				if err := workload_metric.CreateWorkloadMetrics(); err != nil {
					log.Errorf("unable to add record to workload_metrics table: %v. Error: %v", workload_metric, err)
					return
				}
			}

			waittime, err := strconv.Atoi(cfg.KruizeWaitTime)
			if err != nil {
				log.Error(err)
			}
			// Sending list_of_experiments to rosocp.kruize.experiments topic.
			experimentEventMsg := types.ExperimentEvent{
				WorkloadID:          workload.ID,
				Experiment_name:     experiment_name,
				K8s_object_name:     k8s_object[0]["k8s_object_name"].(string),
				K8s_object_type:     k8s_object[0]["k8s_object_type"].(string),
				Namespace:           k8s_object[0]["namespace"].(string),
				Fetch_time:          time.Now().UTC().Add(time.Second * time.Duration(waittime)),
				Monitoring_end_time: interval_end.String(),
				K8s_object:          k8s_object,
			}

			msgBytes, err := json.Marshal(experimentEventMsg)
			if err != nil {
				log.Errorf("Unable convert list_of_experiments to json: %s", err)
			}
			p.SendMessage(msgBytes, &cfg.ExperimentsTopic)

		}

	}

}
