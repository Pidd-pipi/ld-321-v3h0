package service

import (
	"math"
	"testing"

	"github.com/agridispatch/agridispatch/internal/model"
)

func TestBoardComputation(t *testing.T) {
	machines := []model.Machine{
		{Code: "NJ-2026-001", Status: "作业中", CurrentTask: "春耕翻地"},
		{Code: "NJ-2026-002", Status: "空闲"},
		{Code: "NJ-2026-003", Status: "维修中"},
	}
	idle, working := 0, 0
	var workingList []string
	for _, m := range machines {
		switch m.Status {
		case "空闲":
			idle++
		case "作业中":
			working++
			workingList = append(workingList, m.Code+" "+m.CurrentTask)
		}
	}
	if idle != 1 || working != 1 {
		t.Errorf("idle=%d working=%d, want 1/1", idle, working)
	}
	if len(workingList) != 1 || workingList[0] != "NJ-2026-001 春耕翻地" {
		t.Errorf("workingList = %v", workingList)
	}
}

func TestStatsAggregation(t *testing.T) {
	records := []model.WorkRecord{
		{AreaMu: 156, ActualHours: 8.5, FuelCost: 562.4},
		{AreaMu: 88, ActualHours: 5.8, FuelCost: 318.2},
	}
	var s model.Stats
	for _, r := range records {
		s.TotalAreaMu += r.AreaMu
		s.TotalHours += r.ActualHours
		s.FuelCost += r.FuelCost
	}
	if s.TotalAreaMu != 244 || abs(s.TotalHours-14.3) > 1e-9 || abs(s.FuelCost-880.6) > 1e-9 {
		t.Errorf("stats = %+v", s)
	}
}

func abs(x float64) float64 { return math.Abs(x) }
