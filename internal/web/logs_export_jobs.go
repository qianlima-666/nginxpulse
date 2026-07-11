package web

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/analytics"
	"github.com/qianlima-666/nginxpulse/internal/config"
)

const (
	logsExportJobTTL = 24 * time.Hour
)

type LogsExportJobStatus string

const (
	logsExportPending  LogsExportJobStatus = "pending"
	logsExportRunning  LogsExportJobStatus = "running"
	logsExportSuccess  LogsExportJobStatus = "success"
	logsExportFailed   LogsExportJobStatus = "failed"
	logsExportCanceled LogsExportJobStatus = "canceled"
)

type LogsExportJob struct {
	ID        string              `json:"id"`
	Status    LogsExportJobStatus `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	FileName  string              `json:"file_name,omitempty"`
	FilePath  string              `json:"-"`
	Error     string              `json:"error,omitempty"`
	Processed int64               `json:"processed,omitempty"`
	Total     int64               `json:"total,omitempty"`
	WebsiteID string              `json:"website_id,omitempty"`
	Canceled  bool                `json:"-"`
	Params    map[string]string   `json:"-"`
}

type logsExportManager struct {
	mu   sync.Mutex
	jobs map[string]*LogsExportJob
}

var exportJobs = &logsExportManager{
	jobs: make(map[string]*LogsExportJob),
}

func (m *logsExportManager) Create(statsFactory *analytics.StatsFactory, query analytics.StatsQuery, lang string, params map[string]string) (*LogsExportJob, error) {
	if statsFactory == nil {
		return nil, fmt.Errorf("统计模块暂不可用")
	}

	jobID, err := newExportJobID()
	if err != nil {
		return nil, err
	}
	fileName := fmt.Sprintf("nginxpulse_logs_%s.xlsx", time.Now().Format("20060102_150405"))
	exportDir := filepath.Join(config.DataDir, "exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return nil, err
	}
	filePath := filepath.Join(exportDir, fmt.Sprintf("%s.xlsx", jobID))

	job := &LogsExportJob{
		ID:        jobID,
		Status:    logsExportPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		FileName:  fileName,
		FilePath:  filePath,
	}

	job.WebsiteID = query.WebsiteID
	if len(params) > 0 {
		job.Params = make(map[string]string, len(params))
		for k, v := range params {
			job.Params[k] = v
		}
	}

	m.mu.Lock()
	m.cleanupLocked(time.Now())
	m.jobs[jobID] = job
	m.mu.Unlock()

	go m.run(jobID, statsFactory, query, lang)
	return job, nil
}

func (m *logsExportManager) Get(id string) (*LogsExportJob, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(time.Now())
	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	snapshot := *job
	return &snapshot, true
}

func (m *logsExportManager) List(websiteID string, page, pageSize int) ([]LogsExportJob, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(time.Now())
	items := make([]LogsExportJob, 0, len(m.jobs))
	for _, job := range m.jobs {
		if websiteID != "" && job.WebsiteID != websiteID {
			continue
		}
		items = append(items, *job)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []LogsExportJob{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

func (m *logsExportManager) GetParams(id string) (map[string]string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok || len(job.Params) == 0 {
		return nil, ok
	}
	params := make(map[string]string, len(job.Params))
	for k, v := range job.Params {
		params[k] = v
	}
	return params, true
}

func (m *logsExportManager) Cancel(id string) (*LogsExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, fmt.Errorf("任务不存在")
	}
	if job.Status == logsExportSuccess || job.Status == logsExportFailed || job.Status == logsExportCanceled {
		snapshot := *job
		return &snapshot, fmt.Errorf("任务无法取消")
	}
	job.Canceled = true
	if job.Status == logsExportPending {
		job.Status = logsExportCanceled
		job.UpdatedAt = time.Now()
	}
	snapshot := *job
	return &snapshot, nil
}

func (m *logsExportManager) run(jobID string, statsFactory *analytics.StatsFactory, query analytics.StatsQuery, lang string) {
	if m.isCanceled(jobID) {
		m.update(jobID, func(job *LogsExportJob) {
			job.Status = logsExportCanceled
			job.UpdatedAt = time.Now()
		})
		return
	}

	m.update(jobID, func(job *LogsExportJob) {
		job.Status = logsExportRunning
		job.UpdatedAt = time.Now()
	})

	filePath, ok := m.getFilePath(jobID)
	if !ok || filePath == "" {
		m.fail(jobID, fmt.Errorf("导出任务不存在"))
		return
	}

	file, err := os.Create(filePath)
	if err != nil {
		m.fail(jobID, err)
		return
	}
	defer file.Close()

	buffered := bufio.NewWriter(file)
	err = exportLogsXLSXWithProgress(
		buffered,
		statsFactory,
		query,
		lang,
		func(processed, total int64) {
			m.update(jobID, func(job *LogsExportJob) {
				job.Processed = processed
				if total > 0 {
					job.Total = total
				}
				job.UpdatedAt = time.Now()
			})
		},
		func() bool {
			return m.isCanceled(jobID)
		},
	)
	if flushErr := buffered.Flush(); err == nil && flushErr != nil {
		err = flushErr
	}
	if err != nil {
		_ = os.Remove(filePath)
		if err == ErrExportCanceled {
			m.update(jobID, func(job *LogsExportJob) {
				job.Status = logsExportCanceled
				job.UpdatedAt = time.Now()
			})
		} else {
			m.fail(jobID, err)
		}
		return
	}

	m.update(jobID, func(job *LogsExportJob) {
		job.Status = logsExportSuccess
		job.UpdatedAt = time.Now()
	})
}

func (m *logsExportManager) isCanceled(jobID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return true
	}
	return job.Canceled || job.Status == logsExportCanceled
}

func (m *logsExportManager) getFilePath(jobID string) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return "", false
	}
	return job.FilePath, true
}

func (m *logsExportManager) update(jobID string, updater func(job *LogsExportJob)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	updater(job)
}

func (m *logsExportManager) fail(jobID string, err error) {
	m.update(jobID, func(job *LogsExportJob) {
		job.Status = logsExportFailed
		job.Error = err.Error()
		job.UpdatedAt = time.Now()
	})
}

func (m *logsExportManager) cleanupLocked(now time.Time) {
	for id, job := range m.jobs {
		if now.Sub(job.UpdatedAt) <= logsExportJobTTL {
			continue
		}
		if job.FilePath != "" {
			_ = os.Remove(job.FilePath)
		}
		delete(m.jobs, id)
	}
}

func newExportJobID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
