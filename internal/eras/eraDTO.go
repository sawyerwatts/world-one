package eras

import (
	"time"

	"github.com/sawyerwatts/world-one/internal/db"
)

type EraDTO struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

func MakeEraDTO(era db.Era) EraDTO {
	return EraDTO{
		ID:         era.ID,
		Name:       era.Name,
		StartTime:  era.StartTime,
		EndTime:    era.EndTime,
		CreateTime: era.CreateTime,
		UpdateTime: era.UpdateTime,
	}
}
