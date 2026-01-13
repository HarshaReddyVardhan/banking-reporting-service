package security

import (
	"github.com/banking/reporting-service/internal/domain"
)

// Role represents a user role in the system
type Role string

const (
	RoleAdmin             Role = "ADMIN"
	RoleExecutive         Role = "EXECUTIVE"
	RoleComplianceOfficer Role = "COMPLIANCE_OFFICER"
	RoleFinanceTeam       Role = "FINANCE_TEAM"
	RoleOperationsTeam    Role = "OPERATIONS_TEAM"
	RoleFraudTeam         Role = "FRAUD_TEAM"
	RoleExternalAuditor   Role = "EXTERNAL_AUDITOR"
	RoleRegulator         Role = "REGULATOR"
)

// Permission defines what actions a role can perform on report types
type Permission struct {
	AllowedReportTypes []domain.ReportType
	CanDownload        bool
	CanExport          bool
	MaskPII            bool // Whether PII should be masked for this role
}

// RBACManager handles role-based access control
type RBACManager struct {
	permissions map[Role]Permission
}

func NewRBACManager() *RBACManager {
	return &RBACManager{
		permissions: map[Role]Permission{
			RoleAdmin: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeTransactionSummary,
					domain.ReportTypeUserActivity,
					domain.ReportTypeFinancialStatement,
					domain.ReportTypeFraudAnalysis,
				},
				CanDownload: true,
				CanExport:   true,
				MaskPII:     false, // Full access
			},
			RoleExecutive: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeTransactionSummary,
					domain.ReportTypeFinancialStatement,
				},
				CanDownload: true,
				CanExport:   true,
				MaskPII:     true,
			},
			RoleComplianceOfficer: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeFraudAnalysis,
				},
				CanDownload: true,
				CanExport:   true,
				MaskPII:     false, // Compliance needs full data
			},
			RoleFinanceTeam: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeFinancialStatement,
					domain.ReportTypeTransactionSummary,
				},
				CanDownload: true,
				CanExport:   true,
				MaskPII:     true,
			},
			RoleOperationsTeam: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeTransactionSummary,
				},
				CanDownload: true,
				CanExport:   false,
				MaskPII:     true,
			},
			RoleFraudTeam: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeFraudAnalysis,
				},
				CanDownload: true,
				CanExport:   true,
				MaskPII:     false,
			},
			RoleExternalAuditor: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeFinancialStatement,
				},
				CanDownload: true,
				CanExport:   false, // Read-only
				MaskPII:     true,
			},
			RoleRegulator: {
				AllowedReportTypes: []domain.ReportType{
					domain.ReportTypeFinancialStatement,
					domain.ReportTypeFraudAnalysis,
				},
				CanDownload: true,
				CanExport:   false,
				MaskPII:     true,
			},
		},
	}
}

// CanAccessReportType checks if a role can access a specific report type
func (r *RBACManager) CanAccessReportType(role Role, reportType domain.ReportType) bool {
	perm, ok := r.permissions[role]
	if !ok {
		return false
	}
	for _, allowed := range perm.AllowedReportTypes {
		if allowed == reportType {
			return true
		}
	}
	return false
}

// CanDownload checks if a role can download reports
func (r *RBACManager) CanDownload(role Role) bool {
	perm, ok := r.permissions[role]
	if !ok {
		return false
	}
	return perm.CanDownload
}

// CanExport checks if a role can export data
func (r *RBACManager) CanExport(role Role) bool {
	perm, ok := r.permissions[role]
	if !ok {
		return false
	}
	return perm.CanExport
}

// ShouldMaskPII checks if PII should be masked for this role
func (r *RBACManager) ShouldMaskPII(role Role) bool {
	perm, ok := r.permissions[role]
	if !ok {
		return true // Default to masking
	}
	return perm.MaskPII
}

// GetPermission returns the full permission set for a role
func (r *RBACManager) GetPermission(role Role) (Permission, bool) {
	perm, ok := r.permissions[role]
	return perm, ok
}
