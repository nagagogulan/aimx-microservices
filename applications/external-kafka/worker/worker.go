package worker

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"

	kafkas "github.com/PecozQ/aimx-library/kafka"
	"github.com/segmentio/kafka-go"
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

	// Get file size to determine when we're at the end
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Cannot get file stats:", err)
		return
	}
	fileSize := fileInfo.Size()

	writer := kafkas.GetKafkaWriter(topic, os.Getenv("KAFKA_BROKER_ADDRESS"))
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024*512) // 500kb chunks
	chunkIndex := 0
	bytesRead := int64(0)

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			log.Println("Read error:", err)
			break
		}

		if n > 0 {
			bytesRead += int64(n)
			isLastChunk := bytesRead >= fileSize || err == io.EOF

			chunkMsg := map[string]interface{}{
				"file_name":     filepath.Base(filePath),
				"chunk_index":   chunkIndex,
				"data":          buffer[:n],
				"is_last_chunk": isLastChunk,
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

		if err == io.EOF {
			break
		}
	}

	log.Printf("Finished sending chunks for: %s\n", filePath)
}

// FIXME: Path needs to be checked whether the same path is stored in both the external and internal applications.
