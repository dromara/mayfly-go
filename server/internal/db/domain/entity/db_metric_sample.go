package entity

import (
	"time"

	"mayfly-go/pkg/model"
)

// DbMetricSample 实例指标采样点。
//
// 为什么落表而不是每次现算：趋势（「连接数是什么时候涨上来的」）必须靠历史点，
// 现算只能给当下值；而告警评估器的现算结果也不该被图表复用——评估是按规则需求采的，
// 采样频率与保留期都与图表不同，混用会让两边互相绑架。
type DbMetricSample struct {
	model.Model

	InstanceId uint64    `json:"instanceId" gorm:"not null;index:idx_db_metric_sample,priority:1;comment:实例id"`
	Metric     string    `json:"metric" gorm:"size:48;not null;index:idx_db_metric_sample,priority:2;comment:指标名"`
	Value      float64   `json:"value" gorm:"not null;comment:采样值"`
	SampleTime time.Time `json:"sampleTime" gorm:"not null;index:idx_db_metric_sample,priority:3;comment:采样时间"`
}

func (DbMetricSample) TableName() string {
	return "t_db_metric_sample"
}
