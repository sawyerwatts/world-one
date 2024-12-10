package eras

import (
	"fmt"
	"time"

	"github.com/sawyerwatts/world-one/internal/db"
)

type EraDTO struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}

func MakeEraDTO(era db.Era) EraDTO {
	return EraDTO{
		ID:         fmt.Sprintf("%d", era.ID),
		Name:       era.Name,
		StartTime:  era.StartTime,
		EndTime:    era.EndTime,
		CreateTime: era.CreateTime,
		UpdateTime: era.UpdateTime,
	}
}
