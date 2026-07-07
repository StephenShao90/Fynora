package authz

const (
	RoleOwner   = "owner"
	RoleAdmin   = "admin"
	RoleAnalyst = "analyst"
	RoleViewer  = "viewer"
)

func ValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleAnalyst, RoleViewer:
		return true
	default:
		return false
	}
}

func CanRead(role string) bool {
	return ValidRole(role)
}

func CanWriteFinancialData(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleAnalyst:
		return true
	default:
		return false
	}
}

func CanRunReconciliation(role string) bool {
	return CanWriteFinancialData(role)
}

func CanManageMembers(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin:
		return true
	default:
		return false
	}
}

func CanGrantRole(actorRole, targetRole string) bool {
	if actorRole == RoleOwner {
		return ValidRole(targetRole)
	}
	return actorRole == RoleAdmin && targetRole != RoleOwner && ValidRole(targetRole)
}
