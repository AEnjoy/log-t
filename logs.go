package log

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Log struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`

	TestTimes   map[string]int32     `json:"test_times"`
	Success     map[string]int32     `json:"success"`
	Fail        map[string]int32     `json:"fail"`
	SuccessRate map[string]float32   `json:"success_rate"`
	TimeRecords map[string][]float64 `json:"time_records"`

	FailedDetails map[string][][]any `json:"failed_details,omitempty"`
	FailedCount   map[string]int32   `json:"failed_count,omitempty"`

	Tags map[string]any `json:"tags,omitempty"`

	Object any `json:"object,omitempty"` // 任意信息
	l      sync.Mutex
}

func NewLog() *Log {
	return &Log{
		FailedDetails: make(map[string][][]any),
		FailedCount:   make(map[string]int32),
		TestTimes:     make(map[string]int32),
		SuccessRate:   make(map[string]float32),
		Success:       make(map[string]int32),
		Fail:          make(map[string]int32),
		Tags:          make(map[string]any),
		TimeRecords:   make(map[string][]float64),
		StartTime:     time.Now().Format("2006-01-02 15:04:05"),
	}
}

func (m *Log) AddTimeRecord(recordType string, duration time.Duration) {
	m.l.Lock()
	defer m.l.Unlock()

	if recordType == "" {
		recordType = "Total"
	}

	if _, ok := m.TimeRecords[recordType]; !ok {
		m.TimeRecords[recordType] = []float64{}
	}

	m.TimeRecords[recordType] = append(m.TimeRecords[recordType], duration.Seconds())
}
func (m *Log) AddTag(key, value string) {
	m.l.Lock()
	defer m.l.Unlock()
	m.Tags[key] = value
}
func (m *Log) AddRecord(success bool, recordType string, detail ...any) {
	m.l.Lock()
	defer m.l.Unlock()

	if recordType == "" {
		recordType = "Total"
	}
	_, ok := m.TestTimes[recordType]
	if !ok {
		m.TestTimes[recordType] = 0
		m.Success[recordType] = 0
		m.Fail[recordType] = 0
		m.FailedCount[recordType] = 0
		m.FailedDetails[recordType] = make([][]any, 0)

		m.SuccessRate[recordType] = 0
	}
	m.TestTimes[recordType]++

	if success {
		m.Success[recordType]++
		return
	}

	if Debug {
		fmt.Printf(ErrorDebugLogFormat, time.Now().Format("15:04:05"), recordType, detail)
	}

	m.Fail[recordType]++
	m.FailedCount[recordType]++
	m.FailedDetails[recordType] = append(m.FailedDetails[recordType], detail)
}
func (m *Log) CountResult() {
	m.EndTime = time.Now().Format("2006-01-02 15:04:05")
	m.l.Lock()
	defer m.l.Unlock()
	for key, times := range m.TestTimes {
		m.SuccessRate[key] = float32(m.Success[key]) / float32(times)
	}

	for recordType, durations := range m.TimeRecords {
		if len(durations) > 0 {
			// 计算平均时间
			var totalDuration float64
			for _, duration := range durations {
				totalDuration += duration
			}
			averageDuration := totalDuration / float64(len(durations))

			m.Tags[recordType+"_Average"] = averageDuration
		}
	}

	if oper, ok := m.Object.(Count); ok {
		oper.CountResult()
	}
}
func (m *Log) WriteResult(fileName string) error {
	m.CountResult()
	// if fileName is empty, set it to result.json
	if fileName == "" {
		fileName = "result.json"
	}
	// if file is exist, delete it
	if _, err := os.Stat(fileName); err == nil {
		err := os.Remove(fileName)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	if Debug {
		indent, err := json.MarshalIndent(m, "", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(indent))
		_, err = fmt.Fprintln(file, string(indent))
		return err
	}
	return json.NewEncoder(file).Encode(m)
}
