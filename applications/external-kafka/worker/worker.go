package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/segmentio/kafka-go"
	kafkas "whatsdare.com/fullstack/aimx/backend/kafka"
)

type FilePathMsg struct {
	FilePath   string `json:"file_path"`
	ChunkTopic string `json:"chunk_topic"`
}

func StartFileChunkWorker() {
	reader := kafkas.GetKafkaReader("file-paths", os.Getenv("KAFKA_BROKER_ADDRESS"))
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("Kafka read error:", err)
			continue
		}

		var msg FilePathMsg
		err = json.Unmarshal(m.Value, &msg)
		if err != nil {
			log.Println("Unmarshal error:", err)
			continue
		}

		streamFileChunks(msg.FilePath, msg.ChunkTopic)
	}
}

func streamFileChunks(filePath, topic string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Cannot open file:", err)
		return
	}
	defer file.Close()

	writer := kafkas.GetKafkaWriter(topic, os.Getenv("KAFKA_BROKER_ADDRESS"))
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024*512) // 500kb chunks
	chunkIndex := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		chunkMsg := map[string]interface{}{
			"file_name":   filepath.Base(filePath),
			"chunk_index": chunkIndex,
			"data":        buffer[:n],
		}
		chunkData, _ := json.Marshal(chunkMsg)
		err = writer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(filepath.Base(filePath)),
			Value: chunkData,
		})
		if err != nil {
			log.Println("Kafka chunk send error:", err)
			break
		}

		chunkIndex++
	}

	log.Printf("Finished sending chunks for: %s\n", filePath)
}

// FIXME: Path needs to be checked whether the same path is stored in both the external and internal applications.
