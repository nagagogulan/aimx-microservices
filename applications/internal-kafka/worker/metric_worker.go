package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	kafkas "github.com/PecozQ/aimx-library/kafka"
	"github.com/segmentio/kafka-go"
)

// MetricRequest represents the structure of a request message received from get-evaluated-metric topic
type MetricRequest struct {
	UUID string `json:"uuid"`
}

// MetricResponse represents the structure of a response message to be sent to get-evaluated-metric topic
type MetricResponse struct {
	UUID   string                 `json:"uuid"`
	Result map[string]interface{} `json:"result"`
}

// MLFlowExperimentResponse represents the response from MLflow get-by-name API
type MLFlowExperimentResponse struct {
	Experiment struct {
		ExperimentID string `json:"experiment_id"`
		Name         string `json:"name"`
	} `json:"experiment"`
}

// MLFlowRunsSearchResponse represents the response from MLflow runs/search API
type MLFlowRunsSearchResponse struct {
	Runs []struct {
		Info struct {
			RunID string `json:"run_id"`
		} `json:"info"`
	} `json:"runs"`
}

// MLFlowRunResponse represents the response from MLflow runs/get API
type MLFlowRunResponse struct {
	Run struct {
		Data struct {
			Metrics []struct {
				Key   string  `json:"key"`
				Value float64 `json:"value"`
			} `json:"metrics"`
		} `json:"data"`
	} `json:"run"`
}

// StartMetricWorker starts a worker to consume metric messages
func StartMetricWorker() {
	topic := "get-evaluated-metric"
	groupID := "metric-consumer-group"

	// Get Kafka broker address from environment variable
	brokerAddress := os.Getenv("KAFKA_INT_BROKER_ADDRESS")
	if brokerAddress == "" {
		brokerAddress = "54.251.96.179:9092" // Use the same IP as in main.go
		log.Printf("KAFKA_INT_BROKER_ADDRESS not set, using default: %s", brokerAddress)
	}

	reader := kafkas.GetKafkaReader(topic, groupID, brokerAddress)
	defer reader.Close()

	log.Printf("Started metric worker. Listening on topic: %s with group ID: %s", topic, groupID)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		m, err := reader.ReadMessage(ctx)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				continue
			}
			log.Printf("Error reading message from Kafka: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Received message: Key: %s", string(m.Key))

		// Try to unmarshal as a request first
		var metricReq MetricRequest
		if err := json.Unmarshal(m.Value, &metricReq); err != nil {
			log.Printf("Error unmarshalling message: %v", err)
			continue
		}

		// Check if this is a request (only has UUID) or a response (has UUID and result)
		var messageMap map[string]interface{}
		if err := json.Unmarshal(m.Value, &messageMap); err != nil {
			log.Printf("Error unmarshalling message to map: %v", err)
			continue
		}

		// If the message already has a result field, it's a response we've already processed
		if _, hasResult := messageMap["result"]; hasResult {
			log.Printf("Skipping message with result field (already processed)")
			continue
		}

		// This is a request message, process it
		log.Printf("Processing metric request for UUID: %s", metricReq.UUID)

		// Get MLflow metrics
		metrics, err := getMLFlowMetrics()
		if err != nil {
			log.Printf("Error getting MLflow metrics: %v", err)
			continue
		}

		// Publish metrics back to the same topic
		err = publishMetrics(metricReq.UUID, metrics)
		if err != nil {
			log.Printf("Error publishing metrics: %v", err)
			continue
		}

		log.Printf("Successfully processed metric request for UUID: %s", metricReq.UUID)
	}
}

// getMLFlowMetrics retrieves metrics from MLflow
func getMLFlowMetrics() (map[string]interface{}, error) {
	// Get MLflow base URL from environment variable
	mlflowBaseURL := os.Getenv("MLFLOW_BASE_URL")
	if mlflowBaseURL == "" {
		mlflowBaseURL = "http://localhost:8081" // Default if not set
	}

	// Step 1: Get experiment ID by name
	experimentName := "Text_Classification_Evaluation" // This could be configurable
	experimentURL := fmt.Sprintf("%s/api/2.0/mlflow/experiments/get-by-name?experiment_name=%s",
		mlflowBaseURL, experimentName)

	resp, err := http.Get(experimentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get experiment, status: %d, response: %s",
			resp.StatusCode, string(body))
	}

	var experimentResp MLFlowExperimentResponse
	if err := json.NewDecoder(resp.Body).Decode(&experimentResp); err != nil {
		return nil, fmt.Errorf("failed to decode experiment response: %w", err)
	}

	experimentID := experimentResp.Experiment.ExperimentID
	log.Printf("Found experiment ID: %s", experimentID)

	// Step 2: Search for runs in the experiment
	runsURL := fmt.Sprintf("%s/api/2.0/mlflow/runs/search", mlflowBaseURL)
	runsPayload := fmt.Sprintf(`{"experiment_ids":["%s"],"max_results":10}`, experimentID)

	runsResp, err := http.Post(runsURL, "application/json", strings.NewReader(runsPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to search runs: %w", err)
	}
	defer runsResp.Body.Close()

	if runsResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(runsResp.Body)
		return nil, fmt.Errorf("failed to search runs, status: %d, response: %s",
			runsResp.StatusCode, string(body))
	}

	var runsSearchResp MLFlowRunsSearchResponse
	if err := json.NewDecoder(runsResp.Body).Decode(&runsSearchResp); err != nil {
		return nil, fmt.Errorf("failed to decode runs search response: %w", err)
	}

	if len(runsSearchResp.Runs) == 0 {
		return nil, fmt.Errorf("no runs found for experiment ID: %s", experimentID)
	}

	runID := runsSearchResp.Runs[0].Info.RunID
	log.Printf("Using run ID: %s", runID)

	// Step 3: Get run details
	runURL := fmt.Sprintf("%s/api/2.0/mlflow/runs/get?run_id=%s", mlflowBaseURL, runID)

	runResp, err := http.Get(runURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	defer runResp.Body.Close()

	if runResp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(runResp.Body)
		return nil, fmt.Errorf("failed to get run, status: %d, response: %s",
			runResp.StatusCode, string(body))
	}

	var runResponse MLFlowRunResponse
	if err := json.NewDecoder(runResp.Body).Decode(&runResponse); err != nil {
		return nil, fmt.Errorf("failed to decode run response: %w", err)
	}

	// Extract metrics
	metrics := make(map[string]interface{})
	for _, metric := range runResponse.Run.Data.Metrics {
		metrics[metric.Key] = metric.Value
	}

	log.Printf("Retrieved metrics: %+v", metrics)
	return metrics, nil
}

// publishMetrics publishes metrics back to the get-evaluated-metric topic
func publishMetrics(docketUUID string, metrics map[string]interface{}) error {
	// Get Kafka broker address from environment variable
	brokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if brokerAddress == "" {
		brokerAddress = "localhost:9092" // Default if not set
	}

	// Create a Kafka writer for the topic - using the same topic
	topic := "get-evaluated-metric"
	writer := kafkas.GetKafkaWriter(topic, brokerAddress)
	defer writer.Close()

	// Create the message payload with uuid and result
	message := map[string]interface{}{
		"uuid":   docketUUID,
		"result": metrics,
	}

	// Marshal the message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Send the message to Kafka
	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(docketUUID),
		Value: messageBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to write message to Kafka: %w", err)
	}

	log.Printf("Published metrics for docket UUID: %s back to topic: %s", docketUUID, topic)
	return nil
}
