package thermal

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Monitor struct {
	throttled bool
	mu        sync.RWMutex
	stopChan  chan struct{}
	threshold int // Temperature threshold in millidegrees (85000 = 85°C)
}

func NewMonitor() *Monitor {
	return &Monitor{
		throttled: false,
		stopChan:  make(chan struct{}),
		threshold: 85000, // 85°C for OrangePi Zero3
	}
}

func (m *Monitor) Start() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.checkTemperature()
		}
	}
}

func (m *Monitor) checkTemperature() {
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		// If thermal zone doesn't exist, assume OK
		m.mu.Lock()
		m.throttled = false
		m.mu.Unlock()
		return
	}

	tempStr := strings.TrimSpace(string(data))
	temp, err := strconv.Atoi(tempStr)
	if err != nil {
		return
	}

	m.mu.Lock()
	m.throttled = temp > m.threshold
	m.mu.Unlock()
}

func (m *Monitor) IsThrottled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.throttled
}

func (m *Monitor) Stop() {
	close(m.stopChan)
}
