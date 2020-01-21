package prometheus

type Configuration struct {
	PrometheusURL                 string
	MetricsUpdateIntervalInMS     int
	HealthcheckUpdateIntervalInMS int
}

type Query struct {
	Description string
	Query       string
	Threshold   float64
	Alert       bool
}
