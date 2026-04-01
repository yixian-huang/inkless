package qa

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// QAFeedback represents a thumbs up/down rating.
type QAFeedback string

const (
	QAFeedbackNone     QAFeedback = ""
	QAFeedbackPositive QAFeedback = "positive"
	QAFeedbackNegative QAFeedback = "negative"
)

// QALog records a knowledge base Q&A interaction.
type QALog struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Question  string     `gorm:"type:text;not null" json:"question"`
	Answer    string     `gorm:"type:text;not null" json:"answer"`
	Sources   JSONArray  `gorm:"type:jsonb" json:"sources"`
	Locale    string     `gorm:"size:10" json:"locale"`
	IPAddress string     `gorm:"size:45" json:"ipAddress"`
	Rating    QAFeedback `gorm:"size:20" json:"rating"`
	CreatedAt time.Time  `gorm:"autoCreateTime;index" json:"createdAt"`
}

// TableName returns the table name for QALog.
func (QALog) TableName() string {
	return "qa_logs"
}

// JSONArray represents a JSON array stored in the database.
type JSONArray []interface{}

// Value implements the driver.Valuer interface for database serialization.
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal([]interface{}{})
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database deserialization.
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONArray, 0)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, j)
}
