package model

// Message represents the structure of a message received from the topic,
// matching the publisher's format.
type Message struct {
    FileName    string `json:"file_name"` 
    Data        []byte `json:"data"`
    ChunkIndex  int    `json:"chunk_index"`
    IsLastChunk bool   `json:"is_last_chunk"`
}

// DocketEvaluationMessage represents the structure of a message for docket evaluation
type DocketEvaluationMessage struct {
    DocketUUID string                 `json:"docket_uuid"`
    Metadata   map[string]interface{} `json:"metadata"`
    Timestamp  string                 `json:"timestamp"`
}