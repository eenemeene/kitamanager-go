package store

import (
	"context"
	"time"

	"github.com/eenemeene/kitamanager-go/internal/models"
)

// UserOrganizationStorer defines the interface for user-organization relationship operations
type UserOrganizationStorer interface {
	AddUserToOrg(ctx context.Context, userID, orgID uint, role models.Role, createdBy string) (*models.UserOrganization, error)
	UpdateRole(ctx context.Context, userID, orgID uint, role models.Role) error
	RemoveUserFromOrg(ctx context.Context, userID, orgID uint) error
	FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*models.UserOrganization, error)
	FindByUser(ctx context.Context, userID uint) ([]models.UserOrganization, error)
	GetRoleInOrg(ctx context.Context, userID, orgID uint) (models.Role, error)
	GetUserOrganizationsWithRoles(ctx context.Context, userID uint) (map[uint]models.Role, error)
	SetSuperAdmin(ctx context.Context, userID uint, isSuperAdmin bool) error
	IsSuperAdmin(ctx context.Context, userID uint) (bool, error)
	CountSuperAdmins(ctx context.Context) (int64, error)
	Exists(ctx context.Context, userID, orgID uint) (bool, error)
}

// UserStorer defines the interface for user storage operations
type UserStorer interface {
	FindAll(ctx context.Context, search string, limit, offset int) ([]models.User, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.User, int64, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	EmailExistsForOtherUser(ctx context.Context, email string, excludeUserID uint) (bool, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID uint) error
	Delete(ctx context.Context, id uint) error
	// HardDelete bypasses the soft-delete tombstone for purge paths.
	// FindByIDUnscoped returns the row regardless of deleted_at —
	// used by HardDelete and admin trash views.
	HardDelete(ctx context.Context, id uint) error
	FindByIDUnscoped(ctx context.Context, id uint, out *models.User) error
	GetUserOrganizations(ctx context.Context, userID uint) ([]models.Organization, error)
	FindByOrganizations(ctx context.Context, orgIDs []uint, search string, limit, offset int) ([]models.User, int64, error)
	SharesOrganization(ctx context.Context, userID1, userID2 uint) (bool, error)
	IsAdminInSharedOrg(ctx context.Context, requesterID, targetUserID uint) (bool, error)
}

// OrganizationStorer defines the interface for organization storage operations
type OrganizationStorer interface {
	FindAll(ctx context.Context, search string, limit, offset int) ([]models.Organization, int64, error)
	FindByID(ctx context.Context, id uint) (*models.Organization, error)
	Create(ctx context.Context, org *models.Organization) error
	CreateWithDefaultSection(ctx context.Context, org *models.Organization, defaultSection *models.Section) error
	Update(ctx context.Context, org *models.Organization) error
	Delete(ctx context.Context, id uint) error
	// HardDelete / FindByIDUnscoped mirror the User soft-delete
	// surface — purge path and tombstone-aware lookup.
	HardDelete(ctx context.Context, id uint) error
	FindByIDUnscoped(ctx context.Context, id uint, out *models.Organization) error
}

// EmployeeStorer defines the interface for employee storage operations
type EmployeeStorer interface {
	FindAll(ctx context.Context, limit, offset int) ([]models.Employee, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Employee, int64, error)
	FindByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, search string, staffCategory *string, limit, offset int) ([]models.Employee, int64, error)
	FindByID(ctx context.Context, id uint) (*models.Employee, error)
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*models.Employee, error)
	FindByIDMinimal(ctx context.Context, id uint) (*models.Employee, error) // Without preloads, for org checks
	FindByIDMinimalAndOrg(ctx context.Context, id, orgID uint) (*models.Employee, error)
	// FindByIDsAndOrg is the batch form: returns the subset of ids that
	// exist AND belong to orgID, keyed by id. For one-shot existence
	// checks (forecast validation, etc.) — collapses N+1 to 1.
	FindByIDsAndOrg(ctx context.Context, ids []uint, orgID uint) (map[uint]*models.Employee, error)
	Create(ctx context.Context, emp *models.Employee) error
	Update(ctx context.Context, emp *models.Employee) error
	Delete(ctx context.Context, id uint) error
	CreateContract(ctx context.Context, contract *models.EmployeeContract) error
	FindContractByID(ctx context.Context, id uint) (*models.EmployeeContract, error)
	UpdateContract(ctx context.Context, contract *models.EmployeeContract) error
	DeleteContract(ctx context.Context, id uint) error
	Contracts() PeriodStorer[models.EmployeeContract]
	FindByOrganizationWithContracts(ctx context.Context, orgID uint, date time.Time) ([]models.Employee, error)
	FindContractsByEmployeePaginated(ctx context.Context, employeeID uint, limit, offset int) ([]models.EmployeeContract, int64, error)
	FindByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, staffCategories []string, sectionID *uint) ([]models.Employee, error)
	FindByNameBirthdateAndOrg(ctx context.Context, firstName, lastName string, birthdate time.Time, orgID uint) (*models.Employee, error)
	DeleteContractsByEmployee(ctx context.Context, employeeID uint) error
}

// ChildStorer defines the interface for child storage operations
type ChildStorer interface {
	FindAll(ctx context.Context, limit, offset int) ([]models.Child, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, limit, offset int) ([]models.Child, int64, error)
	FindByOrganizationAndSection(ctx context.Context, orgID uint, sectionID *uint, activeOn *time.Time, contractAfter *time.Time, search string, limit, offset int) ([]models.Child, int64, error)
	FindByOrganizationWithActiveOn(ctx context.Context, orgID uint, date time.Time) ([]models.Child, error)
	CountByOrganizationWithActiveOn(ctx context.Context, orgID uint, date time.Time) (int64, error)
	FindContractsByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time) ([]models.ChildContract, error)
	FindByOrganizationInDateRange(ctx context.Context, orgID uint, rangeStart, rangeEnd time.Time, sectionID *uint) ([]models.Child, error)
	FindByID(ctx context.Context, id uint) (*models.Child, error)
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*models.Child, error)
	FindByIDMinimal(ctx context.Context, id uint) (*models.Child, error) // Without preloads, for org checks
	FindByIDMinimalAndOrg(ctx context.Context, id, orgID uint) (*models.Child, error)
	// FindByIDsAndOrg is the batch form: returns the subset of ids that
	// exist AND belong to orgID, keyed by id. Used by forecast validation
	// to collapse per-id existence checks into one round trip.
	FindByIDsAndOrg(ctx context.Context, ids []uint, orgID uint) (map[uint]*models.Child, error)
	Create(ctx context.Context, child *models.Child) error
	Update(ctx context.Context, child *models.Child) error
	Delete(ctx context.Context, id uint) error
	CreateContract(ctx context.Context, contract *models.ChildContract) error
	FindContractByID(ctx context.Context, id uint) (*models.ChildContract, error)
	UpdateContract(ctx context.Context, contract *models.ChildContract) error
	DeleteContract(ctx context.Context, id uint) error
	Contracts() PeriodStorer[models.ChildContract]
	FindContractsByChildPaginated(ctx context.Context, childID uint, limit, offset int) ([]models.ChildContract, int64, error)
	FindByNameBirthdateAndOrg(ctx context.Context, firstName, lastName string, birthdate time.Time, orgID uint) (*models.Child, error)
	DeleteContractsByChild(ctx context.Context, childID uint) error
}

// ChildVoucherStorer defines the interface for child voucher storage operations
type ChildVoucherStorer interface {
	FindChildIDsByVoucherNumbers(ctx context.Context, orgID uint, voucherNumbers []string) (map[string]uint, error)
	FindVouchersByChildID(ctx context.Context, childID uint) ([]models.ChildVoucher, error)
	FindVouchersByChildIDs(ctx context.Context, childIDs []uint) ([]models.ChildVoucher, error)
	FindVouchersByOrganization(ctx context.Context, orgID uint) ([]models.ChildVoucher, error)
	CreateVoucher(ctx context.Context, voucher *models.ChildVoucher) error
	DeleteVouchersByChild(ctx context.Context, childID uint) error
	FindActiveContractsByChildIDsAndDate(ctx context.Context, orgID uint, childIDs []uint, date time.Time) (map[uint]models.ChildContract, error)
	FindChildrenWithoutVouchers(ctx context.Context, orgID uint, activeOn time.Time) ([]models.Child, error)
	FindChildByNameAndBirthMonth(ctx context.Context, orgID uint, firstName, lastName string, birthMonth time.Month, birthYear int) ([]models.Child, error)
}

// PeriodStorer defines the interface for time-bounded record operations
type PeriodStorer[T models.PeriodRecord] interface {
	GetCurrentRecord(ctx context.Context, ownerID uint) (*T, error)
	GetRecordOn(ctx context.Context, ownerID uint, date time.Time) (*T, error)
	ListRecordsPaginated(ctx context.Context, ownerID uint, limit, offset int) ([]T, int64, error)
	HasActiveRecord(ctx context.Context, ownerID uint, date time.Time) (bool, error)
	ValidateNoOverlap(ctx context.Context, ownerID uint, from time.Time, to *time.Time, excludeID *uint) error
	CloseCurrentRecord(ctx context.Context, ownerID uint, endDate time.Time) error
}

// SectionStorer defines the interface for section storage operations
type SectionStorer interface {
	FindByID(ctx context.Context, id uint) (*models.Section, error)
	// FindByIDsAndOrg returns the subset of `ids` that exist AND belong
	// to orgID, keyed by id. Used for batched validation; missing or
	// wrong-org IDs are simply absent from the result.
	FindByIDsAndOrg(ctx context.Context, ids []uint, orgID uint) (map[uint]*models.Section, error)
	FindByOrganizationPaginated(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.Section, int64, error)
	FindDefaultSection(ctx context.Context, orgID uint) (*models.Section, error)
	FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.Section, error)
	Create(ctx context.Context, section *models.Section) error
	Update(ctx context.Context, section *models.Section) error
	Delete(ctx context.Context, id uint) error
	// PromoteToDefault flips is_default so that the given section
	// becomes the only default in its org. Two-statement
	// implementation (clear-then-set) keeps the partial unique
	// index from migration 000019 happy at every statement boundary.
	// Caller wraps in a transaction.
	PromoteToDefault(ctx context.Context, id, orgID uint) error
	// HasActiveChildren / HasActiveEmployees report whether any
	// contract that is active on `asOf` references this section. Used
	// by the delete-guard in SectionService — only currently-assigned
	// contracts block deletion. Historical (ended) contracts under a
	// soft-deleted section keep their FK valid because the section
	// row physically still exists.
	HasActiveChildren(ctx context.Context, id uint, asOf time.Time) (bool, error)
	HasActiveEmployees(ctx context.Context, id uint, asOf time.Time) (bool, error)
	// CountActive* returns the exact count, used by the
	// delete-rejection path to put a number in the error message.
	// Slower than HasActive* (no short-circuit), so reserved for the
	// already-rejected case.
	CountActiveChildren(ctx context.Context, id uint, asOf time.Time) (int64, error)
	CountActiveEmployees(ctx context.Context, id uint, asOf time.Time) (int64, error)
}

// GovernmentFundingStorer defines the interface for government funding storage operations
type GovernmentFundingStorer interface {
	// GovernmentFunding CRUD
	FindAll(ctx context.Context, search string, limit, offset int) ([]models.GovernmentFunding, int64, error)
	FindByID(ctx context.Context, id uint) (*models.GovernmentFunding, error)
	FindByIDWithDetails(ctx context.Context, id uint, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error)
	FindByState(ctx context.Context, state string) (*models.GovernmentFunding, error)
	FindByStateWithDetails(ctx context.Context, state string, periodsLimit int, activeOn *time.Time) (*models.GovernmentFunding, error)
	CountPeriods(ctx context.Context, fundingID uint) (int64, error)
	Create(ctx context.Context, funding *models.GovernmentFunding) error
	Update(ctx context.Context, funding *models.GovernmentFunding) error
	Delete(ctx context.Context, id uint) error

	// Period CRUD
	FindPeriodByID(ctx context.Context, id uint) (*models.GovernmentFundingPeriod, error)
	FindPeriodsByGovernmentFundingID(ctx context.Context, fundingID uint) ([]models.GovernmentFundingPeriod, error)
	CreatePeriod(ctx context.Context, period *models.GovernmentFundingPeriod) error
	UpdatePeriod(ctx context.Context, period *models.GovernmentFundingPeriod) error
	DeletePeriod(ctx context.Context, id uint) error

	// Property CRUD
	FindPropertyByID(ctx context.Context, id uint) (*models.GovernmentFundingProperty, error)
	CreateProperty(ctx context.Context, property *models.GovernmentFundingProperty) error
	UpdateProperty(ctx context.Context, property *models.GovernmentFundingProperty) error
	DeleteProperty(ctx context.Context, id uint) error

	// Paginated nested-resource queries
	FindPeriodsByGovernmentFundingIDPaginated(ctx context.Context, fundingID uint, limit, offset int) ([]models.GovernmentFundingPeriod, int64, error)
	FindPropertiesByPeriodPaginated(ctx context.Context, periodID uint, limit, offset int) ([]models.GovernmentFundingProperty, int64, error)
}

// ChildAttendanceStorer defines the interface for child attendance storage operations
type ChildAttendanceStorer interface {
	FindByID(ctx context.Context, id uint) (*models.ChildAttendance, error)
	FindByOrganizationAndDate(ctx context.Context, orgID uint, date time.Time, limit, offset int) ([]models.ChildAttendance, int64, error)
	FindByChildAndDate(ctx context.Context, childID uint, date time.Time) (*models.ChildAttendance, error)
	FindByChildAndDateRange(ctx context.Context, childID uint, from, to time.Time, limit, offset int) ([]models.ChildAttendance, int64, error)
	Create(ctx context.Context, attendance *models.ChildAttendance) error
	Update(ctx context.Context, attendance *models.ChildAttendance) error
	Delete(ctx context.Context, id uint) error
	GetDailySummary(ctx context.Context, orgID uint, date time.Time) (*models.ChildAttendanceDailySummaryResponse, error)
}

// PayPlanStorer defines the interface for pay plan storage operations
type PayPlanStorer interface {
	Create(ctx context.Context, payplan *models.PayPlan) error
	FindByID(ctx context.Context, id uint) (*models.PayPlan, error)
	FindByIDWithPeriods(ctx context.Context, id uint, activeOn *time.Time) (*models.PayPlan, error)
	FindByIDsWithPeriods(ctx context.Context, ids []uint) (map[uint]*models.PayPlan, error)
	FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.PayPlan, int64, error)
	FindByNameAndOrg(ctx context.Context, name string, orgID uint) (*models.PayPlan, error)
	Update(ctx context.Context, payplan *models.PayPlan) error
	Delete(ctx context.Context, id uint) error
	DeletePeriodsAndEntries(ctx context.Context, payplanID uint) error

	// Period operations
	CreatePeriod(ctx context.Context, period *models.PayPlanPeriod) error
	FindPeriodByID(ctx context.Context, id uint) (*models.PayPlanPeriod, error)
	FindPeriodByIDWithEntries(ctx context.Context, id uint) (*models.PayPlanPeriod, error)
	FindPeriodsByPayPlan(ctx context.Context, payplanID uint) ([]models.PayPlanPeriod, error)
	FindActivePeriod(ctx context.Context, payplanID uint, date time.Time) (*models.PayPlanPeriod, error)
	UpdatePeriod(ctx context.Context, period *models.PayPlanPeriod) error
	DeletePeriod(ctx context.Context, id uint) error

	// Entry operations
	CreateEntry(ctx context.Context, entry *models.PayPlanEntry) error
	FindEntryByID(ctx context.Context, id uint) (*models.PayPlanEntry, error)
	FindEntriesByPeriod(ctx context.Context, periodID uint) ([]models.PayPlanEntry, error)
	FindEntry(ctx context.Context, periodID uint, grade string, step int) (*models.PayPlanEntry, error)
	UpdateEntry(ctx context.Context, entry *models.PayPlanEntry) error
	DeleteEntry(ctx context.Context, id uint) error

	// Paginated nested-resource queries
	FindPeriodsByPayPlanPaginated(ctx context.Context, payplanID uint, limit, offset int) ([]models.PayPlanPeriod, int64, error)
	FindEntriesByPeriodPaginated(ctx context.Context, periodID uint, limit, offset int) ([]models.PayPlanEntry, int64, error)
}

// BudgetItemStorer defines the interface for budget item storage operations
type BudgetItemStorer interface {
	Create(ctx context.Context, item *models.BudgetItem) error
	FindByID(ctx context.Context, id uint) (*models.BudgetItem, error)
	FindByIDWithEntries(ctx context.Context, id uint) (*models.BudgetItem, error)
	FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.BudgetItem, int64, error)
	FindByOrganizationWithEntries(ctx context.Context, orgID uint) ([]models.BudgetItem, error)
	Update(ctx context.Context, item *models.BudgetItem) error
	Delete(ctx context.Context, id uint) error

	// Entry operations
	CreateEntry(ctx context.Context, entry *models.BudgetItemEntry) error
	FindEntryByID(ctx context.Context, id uint) (*models.BudgetItemEntry, error)
	FindEntriesByBudgetItemPaginated(ctx context.Context, budgetItemID uint, limit, offset int) ([]models.BudgetItemEntry, int64, error)
	UpdateEntry(ctx context.Context, entry *models.BudgetItemEntry) error
	DeleteEntry(ctx context.Context, id uint) error
	// CountEntries returns how many entries exist for a budget item.
	// Used by the service-layer toggle guard: changing category or
	// per_child after entries exist would silently flip the meaning
	// of every historical row in financials. Cheap COUNT(*) — avoids
	// loading entries just to check len().
	CountEntries(ctx context.Context, budgetItemID uint) (int64, error)
	Entries() PeriodStorer[models.BudgetItemEntry]
}

// FactorStorer defines the interface for multi-factor authentication
// storage. All methods that address a single factor by id take the
// owning userID too so cross-user access returns "not found" rather
// than leaking existence.
type FactorStorer interface {
	FindByUserID(ctx context.Context, userID uint) ([]models.Factor, error)
	FindActiveByUserID(ctx context.Context, userID uint) ([]models.Factor, error)
	FindByIDAndUser(ctx context.Context, id, userID uint) (*models.Factor, error)
	FindPendingByUserAndType(ctx context.Context, userID uint, factorType string) (*models.Factor, error)
	FindBackupCodesFactor(ctx context.Context, userID uint) (*models.Factor, error)
	CreateFactor(ctx context.Context, f *models.Factor) error
	ActivateFactor(ctx context.Context, id, userID uint) (bool, error)
	IncrementActivationFailures(ctx context.Context, id, userID uint) (int, error)
	DeleteFactor(ctx context.Context, id, userID uint) (int64, error)
	UpdateLabel(ctx context.Context, id, userID uint, label *string) (bool, error)
	TouchLastUsed(ctx context.Context, id uint) error
	CleanupAbandonedPending(ctx context.Context, olderThan time.Duration) (int64, error)

	CreateTOTPSecret(ctx context.Context, secret *models.FactorTOTPSecret) error
	FindTOTPSecret(ctx context.Context, factorID uint) (*models.FactorTOTPSecret, error)
	AcceptTOTPStep(ctx context.Context, factorID uint, step int64) (bool, error)

	InsertBackupCodes(ctx context.Context, codes []models.FactorBackupCode) error
	ListBackupCodes(ctx context.Context, factorID uint) ([]models.FactorBackupCode, error)
	CountUnusedBackupCodes(ctx context.Context, factorID uint) (int, error)
	ConsumeBackupCode(ctx context.Context, factorID uint, codeHash string) (bool, error)
	ReplaceBackupCodes(ctx context.Context, factorID uint, fresh []models.FactorBackupCode) error

	// WebAuthn subtable + registration-challenge lifecycle.
	CreateWebAuthnCredential(ctx context.Context, cred *models.FactorWebAuthnCredential) error
	FindWebAuthnCredential(ctx context.Context, factorID uint) (*models.FactorWebAuthnCredential, error)
	FindWebAuthnCredentialByID(ctx context.Context, credentialID []byte) (*models.FactorWebAuthnCredential, error)
	UpdateWebAuthnSignCount(ctx context.Context, factorID uint, newCount int64, backupState bool) error
	SetRegistrationChallenge(ctx context.Context, factorID uint, challenge []byte, expiresAt time.Time) error
	ClearRegistrationChallenge(ctx context.Context, factorID uint) error
}

// SessionStorer defines the interface for server-side session storage.
// The `idHash` parameters are sha256 hex of the raw cookie value.
type SessionStorer interface {
	Create(ctx context.Context, sess *models.Session) error
	Lookup(ctx context.Context, idHash string) (*SessionLookupResult, error)
	Delete(ctx context.Context, idHash string) error
	DeleteAllForUser(ctx context.Context, userID uint) error
	DeleteAllForUserExcept(ctx context.Context, userID uint, keepIDHash string) error
	CleanupExpired(ctx context.Context) error
	ListForUser(ctx context.Context, userID uint) ([]models.Session, error)
	DeleteForUser(ctx context.Context, idHash string, userID uint) (int64, error)

	// Pending-MFA (two-step login) helpers. Kept on the same interface
	// because they share the same storage, transaction boundary, and
	// cleanup job as regular sessions.
	LookupPendingMFA(ctx context.Context, idHash string) (*SessionPendingLookupResult, error)
	BumpMFAChallengeFailures(ctx context.Context, idHash string) (int, error)
	DeletePendingMFA(ctx context.Context, idHash string) error
	SetPendingMFAChallenge(ctx context.Context, idHash string, challenge []byte) error
}

// AuditStorer defines the interface for audit log storage operations
type AuditStorer interface {
	Create(ctx context.Context, log *models.AuditLog) error
	FindByID(ctx context.Context, id uint) (*models.AuditLog, error)
	FindByUser(ctx context.Context, userID uint, limit, offset int) ([]models.AuditLog, int64, error)
	FindByAction(ctx context.Context, action models.AuditAction, limit, offset int) ([]models.AuditLog, int64, error)
	FindByDateRange(ctx context.Context, from, to time.Time, limit, offset int) ([]models.AuditLog, int64, error)
	FindAll(ctx context.Context, limit, offset int) ([]models.AuditLog, int64, error)
	FindAllFiltered(ctx context.Context, action string, userID *uint, from *time.Time, to *time.Time, limit, offset int) ([]models.AuditLog, int64, error)
	FindByOrganization(ctx context.Context, orgID uint, action string, userID *uint, from, to *time.Time, limit, offset int) ([]models.AuditLog, int64, error)
	FindFailedLogins(ctx context.Context, email string, since time.Time, limit int) ([]models.AuditLog, error)
	CountFailedLoginsSince(ctx context.Context, email string, since time.Time) (int64, error)
	CountFailedPasswordChangesSince(ctx context.Context, userID uint, since time.Time) (int64, error)
	CountFailedPasswordResetsSince(ctx context.Context, actorID uint, since time.Time) (int64, error)
	CountFailedMFAChallengesSince(ctx context.Context, userID uint, since time.Time) (int64, error)
	Cleanup(ctx context.Context, olderThan time.Time) (int64, error)
}

// GovernmentFundingBillPeriodStorer defines the interface for government funding bill period storage operations
type GovernmentFundingBillPeriodStorer interface {
	Create(ctx context.Context, period *models.GovernmentFundingBillPeriod) error
	FindByID(ctx context.Context, id uint) (*models.GovernmentFundingBillPeriod, error)
	FindByOrganization(ctx context.Context, orgID uint, search string, limit, offset int) ([]models.GovernmentFundingBillPeriod, int64, error)
	FindByOrganizationAndVoucherNumber(ctx context.Context, orgID uint, voucherNumber string) ([]models.BillAppearance, error)
	FindChildEntriesByOrgAndVoucherNumbers(ctx context.Context, orgID uint, voucherNumbers []string) ([]models.GovernmentFundingBillChildWithPeriod, error)
	FindBilledTotalsByOrg(ctx context.Context, orgID uint) ([]models.VoucherBilledTotal, error)
	FindAllBillDatesAndVouchersByOrg(ctx context.Context, orgID uint) ([]models.BillDateVoucher, error)
	FindFacilityTotalsByOrganizationInDateRange(ctx context.Context, orgID uint, from, to time.Time) (map[time.Time]int, error)
	FindBillTotalsByRowTypeInDateRange(ctx context.Context, orgID uint, from, to time.Time) (map[time.Time]BillTotalsByRowType, error)
	FindLatestByOrganization(ctx context.Context, orgID uint) (*models.GovernmentFundingBillPeriod, error)
	FindByOrgAndMonth(ctx context.Context, orgID uint, from time.Time) (*models.GovernmentFundingBillPeriod, error)
	ExistsByOrgAndHash(ctx context.Context, orgID uint, fileHash string) (bool, error)
	ExistsByOrgAndMonth(ctx context.Context, orgID uint, from time.Time) (bool, error)
	Delete(ctx context.Context, id uint) error
}

// Compile-time interface compliance checks
var (
	_ UserStorer                        = (*UserStore)(nil)
	_ OrganizationStorer                = (*OrganizationStore)(nil)
	_ EmployeeStorer                    = (*EmployeeStore)(nil)
	_ ChildStorer                       = (*ChildStore)(nil)
	_ UserOrganizationStorer            = (*UserOrganizationStore)(nil)
	_ GovernmentFundingStorer           = (*GovernmentFundingStore)(nil)
	_ GovernmentFundingBillPeriodStorer = (*GovernmentFundingBillPeriodStore)(nil)
	_ SectionStorer                     = (*SectionStore)(nil)
	_ ChildAttendanceStorer             = (*ChildAttendanceStore)(nil)
	_ PayPlanStorer                     = (*PayPlanStore)(nil)
	_ AuditStorer                       = (*AuditStore)(nil)
	_ ChildVoucherStorer                = (*ChildVoucherStore)(nil)
	_ BudgetItemStorer                  = (*BudgetItemStore)(nil)
	_ SessionStorer                     = (*SessionStore)(nil)
	_ FactorStorer                      = (*FactorStore)(nil)
	_ Transactor                        = (*GormTransactor)(nil)
)
