package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)
	httpResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_time_seconds",
			Help:    "HTTP response time",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)
	cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "cpu_usage",
		Help: "CPU usage percentage",
	})
	memoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_usage",
		Help: "Memory usage percentage",
	})
	networkTraffic = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "network_traffic_bytes_total",
		Help: "Network traffic in bytes",
	},
		[]string{"direction"},
	)
)

var log = logrus.New()

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpResponseTime)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memoryUsage)
	prometheus.MustRegister(networkTraffic)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetReportCaller(true)
}

type RequestBody struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Message     string `json:"message" binding:"required"`
}

func main() {
	go collectSystemMetrics()
	r := gin.Default()
	r.Use(PrometheusMiddleware())

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.POST("/sms", func(c *gin.Context) {
		var requestBody RequestBody

		if err := c.ShouldBindJSON(&requestBody); err != nil {
			log.Warnf("Falha ao analisar o corpo da solicitação: %s", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Corpo de solicitação inválido",
			})
			return
		}

		phoneNumber := requestBody.PhoneNumber
		message := requestBody.Message

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-east-1"),
		})

		if err != nil {
			log.Errorf("Falha ao criar uma nova sessão da AWS: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Falha ao criar uma nova sessão da AWS",
			})
			return
		}

		snsClient := sns.New(sess)

		params := &sns.PublishInput{
			PhoneNumber: aws.String(phoneNumber),
			Message:     aws.String(message),
		}

		response, err := snsClient.Publish(params)
		if err != nil {
			log.Errorf("Falha ao enviar SMS: %s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "Falha ao enviar SMS",
			})
			return
		}

		log.Infof("SMS enviado com sucesso para %s. MessageId: %s", phoneNumber, *response.MessageId)

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("SMS enviado com sucesso para %s. MessageId: %s", phoneNumber, *response.MessageId),
		})
	})

	err := r.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		timer := prometheus.NewTimer(httpResponseTime.WithLabelValues(c.Request.Method, c.Request.URL.Path, "0"))
		defer timer.ObserveDuration()

		c.Next()

		statusCode := fmt.Sprint(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.Request.URL.Path, statusCode).Inc()
	}
}

func collectSystemMetrics() {
	for {
		cpuPercent, _ := cpu.Percent(0, false)
		if len(cpuPercent) > 0 {
			cpuUsage.Set(cpuPercent[0])
		}

		memStat, _ := mem.VirtualMemory()
		memoryUsage.Set(memStat.UsedPercent)

		netStat, _ := net.IOCounters(false)
		for _, stat := range netStat {
			networkTraffic.WithLabelValues("sent").Add(float64(stat.BytesSent))
			networkTraffic.WithLabelValues("received").Add(float64(stat.BytesRecv))
		}

		time.Sleep(5 * time.Second)
	}
}
