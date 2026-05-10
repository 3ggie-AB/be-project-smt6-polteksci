package ml

import (
	"math"
	"strconv"
	"sync"
	"time"

	"project_smt6/domain"
)

type FeatureEngine struct {
	mu         sync.RWMutex
	windowSize int
	state      map[string]*deviceState
}

type deviceState struct {
	workspace       string
	latencies       []float64
	packetLosses    []float64
	clientCounts    []float64
	throughputs     []float64
	lastClientCount int
	roamingEvents   int
	lastUpdated     time.Time
}

func NewFeatureEngine(windowSize int) *FeatureEngine {
	if windowSize <= 0 {
		windowSize = 60
	}
	return &FeatureEngine{
		windowSize: windowSize,
		state:      make(map[string]*deviceState),
	}
}

func (e *FeatureEngine) AddPing(metric domain.PingMetric) domain.FeatureVector {
	e.mu.Lock()
	defer e.mu.Unlock()

	state := e.getState(entityKey(metric.DeviceID, metric.TargetID), metric.Workspace)
	state.latencies = appendBounded(state.latencies, metric.LatencyMS, e.windowSize)
	state.packetLosses = appendBounded(state.packetLosses, metric.PacketLoss, e.windowSize)
	state.lastUpdated = metric.Timestamp
	return e.vectorLocked(metric.DeviceID, metric.TargetID, state, metric.Timestamp)
}

func (e *FeatureEngine) AddAP(metric domain.APMetric) domain.FeatureVector {
	e.mu.Lock()
	defer e.mu.Unlock()

	state := e.getState(entityKey(metric.DeviceID, 0), metric.Workspace)
	if state.lastClientCount > 0 && math.Abs(float64(metric.ClientCount-state.lastClientCount)) >= 5 {
		state.roamingEvents++
	}
	state.lastClientCount = metric.ClientCount
	state.clientCounts = appendBounded(state.clientCounts, float64(metric.ClientCount), e.windowSize)
	state.throughputs = appendBounded(state.throughputs, metric.ThroughputBPS, e.windowSize)
	state.lastUpdated = metric.Timestamp
	return e.vectorLocked(metric.DeviceID, 0, state, metric.Timestamp)
}

func (e *FeatureEngine) VectorFor(deviceID uint) (domain.FeatureVector, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.state[entityKey(deviceID, 0)]
	if !ok {
		return domain.FeatureVector{}, false
	}
	return e.vectorLocked(deviceID, 0, state, state.lastUpdated), true
}

func (e *FeatureEngine) VectorForTarget(targetID uint) (domain.FeatureVector, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	state, ok := e.state[entityKey(0, targetID)]
	if !ok {
		return domain.FeatureVector{}, false
	}
	return e.vectorLocked(0, targetID, state, state.lastUpdated), true
}

func (e *FeatureEngine) getState(key, workspace string) *deviceState {
	state, ok := e.state[key]
	if ok {
		if workspace != "" {
			state.workspace = workspace
		}
		return state
	}
	state = &deviceState{workspace: workspace}
	e.state[key] = state
	return state
}

func (e *FeatureEngine) vectorLocked(deviceID, targetID uint, state *deviceState, ts time.Time) domain.FeatureVector {
	return domain.FeatureVector{
		DeviceID:            deviceID,
		TargetID:            targetID,
		Workspace:           state.workspace,
		LatencyRollingAvgMS: avg(state.latencies),
		PacketLossRatio:     avg(state.packetLosses) / 100,
		APLoadScore:         normalize(avg(state.clientCounts), 0, 250),
		RoamingFrequency:    float64(state.roamingEvents) / math.Max(1, float64(len(state.clientCounts))),
		TrafficAnomalyScore: zScoreLast(state.throughputs),
		Timestamp:           ts,
	}
}

func entityKey(deviceID, targetID uint) string {
	if targetID > 0 {
		return "target:" + strconv.FormatUint(uint64(targetID), 10)
	}
	return "device:" + strconv.FormatUint(uint64(deviceID), 10)
}

func appendBounded(values []float64, value float64, limit int) []float64 {
	values = append(values, value)
	if len(values) > limit {
		copy(values, values[len(values)-limit:])
		values = values[:limit]
	}
	return values
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func normalize(value, minValue, maxValue float64) float64 {
	if maxValue <= minValue {
		return 0
	}
	n := (value - minValue) / (maxValue - minValue)
	if n < 0 {
		return 0
	}
	if n > 1 {
		return 1
	}
	return n
}

func zScoreLast(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := avg(values)
	var variance float64
	for _, value := range values {
		diff := value - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(len(values)))
	if stddev == 0 {
		return 0
	}
	return math.Abs((values[len(values)-1] - mean) / stddev)
}
