package service

import (
	"math"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/eenemeene/kitamanager-go/internal/models"
	"github.com/eenemeene/kitamanager-go/internal/store"
)

func createStatisticsService(db *gorm.DB) *StatisticsService {
	childStore := store.NewChildStore(db)
	employeeStore := store.NewEmployeeStore(db)
	orgStore := store.NewOrganizationStore(db)
	fundingStore := store.NewGovernmentFundingStore(db)
	payPlanStore := store.NewPayPlanStore(db)
	budgetItemStore := store.NewBudgetItemStore(db)
	sectionStore := store.NewSectionStore(db)
	billStore := store.NewGovernmentFundingBillPeriodStore(db)
	return NewStatisticsService(childStore, employeeStore, orgStore, fundingStore, payPlanStore, budgetItemStore, sectionStore, billStore)
}

func createTestEmployeeContractWithCategory(t *testing.T, db *gorm.DB, employeeID uint, payplanID uint, from time.Time, to *time.Time, weeklyHours float64, staffCategory string, sectionID uint) *models.EmployeeContract {
	t.Helper()
	contract := &models.EmployeeContract{
		EmployeeID: employeeID,
		BaseContract: models.BaseContract{
			Period:    models.Period{From: from, To: to},
			SectionID: sectionID,
		},
		StaffCategory: staffCategory,
		WeeklyHours:   weeklyHours,
		PayPlanID:     payplanID,
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create test employee contract: %v", err)
	}
	return contract
}

func createTestChildContract(t *testing.T, db *gorm.DB, childID uint, from time.Time, to *time.Time, sectionID uint, properties models.ContractProperties) *models.ChildContract {
	t.Helper()
	contract := &models.ChildContract{
		ChildID: childID,
		BaseContract: models.BaseContract{
			Period:     models.Period{From: from, To: to},
			SectionID:  sectionID,
			Properties: properties,
		},
	}
	if err := db.Create(contract).Error; err != nil {
		t.Fatalf("failed to create test child contract: %v", err)
	}
	return contract
}

func createTestFundingPropertyWithRequirement(t *testing.T, db *gorm.DB, periodID uint, key, value string, requirement float64, minAge, maxAge int) *models.GovernmentFundingProperty {
	t.Helper()
	var minAgePtr, maxAgePtr *int
	if minAge >= 0 {
		minAgePtr = &minAge
	}
	if maxAge >= 0 {
		maxAgePtr = &maxAge
	}
	prop := &models.GovernmentFundingProperty{
		PeriodID:    periodID,
		Key:         key,
		Value:       value,
		Label:       strings.ToUpper(value[:1]) + value[1:],
		Payment:     10000,
		Requirement: requirement,
		MinAge:      minAgePtr,
		MaxAge:      maxAgePtr,
	}
	if err := db.Create(prop).Error; err != nil {
		t.Fatalf("failed to create test funding property: %v", err)
	}
	return prop
}

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func findDataPoint(t *testing.T, dps []models.FinancialDataPoint, date string) models.FinancialDataPoint {
	t.Helper()
	for _, dp := range dps {
		if dp.Date == date {
			return dp
		}
	}
	t.Fatalf("data point for %s not found", date)
	return models.FinancialDataPoint{}
}
