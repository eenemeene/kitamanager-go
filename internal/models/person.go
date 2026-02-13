package models

import "time"

// Person represents the base person data shared by Employee and Child.
// Section assignment is tracked on contracts (BaseContract.SectionID), not on the person,
// so it can change over time as contracts change.
type Person struct {
	ID             uint          `gorm:"primaryKey" json:"id" example:"1"`
	OrganizationID uint          `gorm:"not null;index" json:"organization_id" example:"1"`
	Organization   *Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	FirstName      string        `gorm:"size:255;not null" json:"first_name" example:"Max"`
	LastName       string        `gorm:"size:255;not null" json:"last_name" example:"Mustermann"`
	Gender         string        `gorm:"size:20;not null" json:"gender" example:"male"`
	Birthdate      time.Time     `gorm:"type:date;not null" json:"birthdate" example:"1990-05-15"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// GetOrganizationID returns the organization ID for the OrgOwned interface.
func (p Person) GetOrganizationID() uint {
	return p.OrganizationID
}

// FullName returns the person's full name.
func (p Person) FullName() string {
	return p.FirstName + " " + p.LastName
}
