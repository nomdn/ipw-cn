package webtest

import (
	"fmt"
	"math"
	"net"
	"time"
)

type TCPingResult struct {
	IP        string    `json:"ip"`
	Port      string    `json:"port"`
	Success   bool      `json:"success"`
	RTT       float64   `json:"rtt"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

type TCPingStats struct {
	IP       string         `json:"ip"`
	Port     string         `json:"port"`
	Sent     int            `json:"sent"`
	Success  int            `json:"success"`
	LossRate float64        `json:"loss_rate"`
	MaxRTT   float64        `json:"max_rtt"`
	MinRTT   float64        `json:"min_rtt"`
	AvgRTT   float64        `json:"avg_rtt"`
	Results  []TCPingResult `json:"results"`
}

func resolveHost(host string, version string) (string, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed: %w", err)
	}

	for _, ip := range ips {
		if version == "v4" && ip.To4() != nil {
			return ip.String(), nil
		}
		if version == "v6" && ip.To4() == nil && ip.To16() != nil {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no %s address found for %s", version, host)
}

func TCPing(host string, port string, version string, timeout time.Duration) (*TCPingResult, error) {
	ip, err := resolveHost(host, version)
	if err != nil {
		return nil, err
	}
	addr := ""
	switch version {
	case "v4":
		addr = fmt.Sprintf("%s:%s", ip, port)
	case "v6":
		addr = fmt.Sprintf("[%s]:%s", ip, port)
	}

	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, timeout)
	rtt := time.Since(start)

	result := &TCPingResult{
		IP:        ip,
		Port:      port,
		Timestamp: start,
	}

	if err != nil {
		result.Success = false
		result.RTT = -1
		result.Error = err.Error()
	} else {
		conn.Close()
		result.Success = true
		result.RTT = float64(rtt.Microseconds()) / 1000.0
	}

	return result, nil
}

func TCPingRun(host string, port string, count int, version string, timeout time.Duration, interval time.Duration) (*TCPingStats, error) {
	ip, err := resolveHost(host, version)
	if err != nil {
		return nil, err
	}

	stats := &TCPingStats{
		IP:      ip,
		Port:    port,
		Sent:    count,
		MinRTT:  math.MaxFloat64,
		Results: make([]TCPingResult, 0, count),
	}

	var totalRTT float64
	successCount := 0

	for i := 0; i < count; i++ {
		result, err := TCPing(host, port, version, timeout)
		if err != nil {
			return nil, err
		}

		stats.Results = append(stats.Results, *result)

		if result.Success {
			successCount++
			totalRTT += result.RTT
			if result.RTT > stats.MaxRTT {
				stats.MaxRTT = result.RTT
			}
			if result.RTT < stats.MinRTT {
				stats.MinRTT = result.RTT
			}
		}

		if i < count-1 && interval > 0 {
			time.Sleep(interval)
		}
	}

	stats.Success = successCount
	stats.LossRate = math.Round(float64(count-successCount)*10000/float64(count)) / 100

	if successCount > 0 {
		stats.AvgRTT = math.Round(totalRTT*100/float64(successCount)) / 100
	} else {
		stats.MinRTT = -1
		stats.MaxRTT = -1
		stats.AvgRTT = -1
	}

	return stats, nil
}