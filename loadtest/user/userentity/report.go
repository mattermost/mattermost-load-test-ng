package userentity

import (
	"context"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

var (
	userAgents = []string{"desktop", "firefox", "chrome", "safari", "edge", "other"}
	platforms  = []string{"linux", "macos", "ios", "android", "windows", "other"}
)

func (ue *UserEntity) ObserveClientMetric(t model.MetricType, v float64) error {
	report, err := ue.store.PerformanceReport()
	if err != nil {
		return err
	}
	defer ue.store.SetPerformanceReport(report)

	switch t {
	case model.ClientTimeToFirstByte, model.ClientFirstContentfulPaint, model.ClientLargestContentfulPaint,
		model.ClientInteractionToNextPaint, model.ClientCumulativeLayoutShift, model.ClientChannelSwitchDuration,
		model.ClientTeamSwitchDuration, model.ClientRHSLoadDuration:
		if report.Histograms == nil {
			report.Histograms = make([]*model.MetricSample, 0)
		}

		report.Histograms = append(report.Histograms, &model.MetricSample{
			Metric:    t,
			Value:     v,
			Timestamp: float64(time.Now().UnixMilli()) / 1000,
		})
	default:
		// server also ignores the unkown typed metrics
	}
	return nil
}

func (ue *UserEntity) SubmitPerformanceReport() error {
	report, err := ue.store.PerformanceReport()
	if err != nil {
		return err
	}
	report.End = float64(time.Now().UnixMilli()) / 1000

	_, err = ue.client.SubmitClientMetrics(context.Background(), report)
	if err != nil {
		return err
	}
	ue.store.SetPerformanceReport(&model.PerformanceReport{
		Start: float64(time.Now().UnixMilli()) / 1000,
	})

	return nil
}

func randomUserAgent() string {
	i := rand.Intn(len(userAgents))
	return userAgents[i]
}

func randomPlatform() string {
	i := rand.Intn(len(platforms))
	return platforms[i]
}
