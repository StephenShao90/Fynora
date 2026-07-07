package main

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
	"github.com/StephenShao90/Fynora/services/api/internal/repository"
	"github.com/StephenShao90/Fynora/services/api/internal/validation"
)

func (a *app) listOrganizationsV1(w http.ResponseWriter, r *http.Request) {
	memberships, err := a.userMemberships(r.Context(), userID(r))
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not list organizations")
		return
	}
	writeJSON(w, 200, membershipOrganizations(memberships))
}

func (a *app) createOrganizationV1(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Currency string `json:"currency"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := validation.RequiredString(req.Name, "name"); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if err := validation.Currency(req.Currency); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Type == "" {
		req.Type = "small_business"
	}
	if a.cfRepo != nil {
		u, ok := a.currentUser(r)
		if !ok {
			errorJSON(w, r, 404, "NOT_FOUND", "user not found")
			return
		}
		org, err := a.cfRepo.CreateOrganization(r.Context(), u, models.Organization{Name: req.Name, Type: req.Type, Currency: req.Currency})
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not create organization")
			return
		}
		writeJSON(w, 201, org)
		return
	}
	a.store.mu.Lock()
	org := a.ensureOrganizationLocked(userID(r), req.Name)
	org.Type = req.Type
	org.Currency = req.Currency
	org.UpdatedAt = time.Now().UTC()
	a.store.organizations[org.ID] = org
	a.addOrganizationMemberLocked(org.ID, userID(r), authz.RoleOwner)
	a.store.mu.Unlock()
	writeJSON(w, 201, org)
}

func (a *app) listOrganizationMembersV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := pathOrganizationID(w, r)
	if !ok {
		return
	}
	if _, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers); !ok {
		return
	}
	members, err := a.organizationMembers(r.Context(), orgID)
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not list members")
		return
	}
	writeJSON(w, 200, members)
}

func (a *app) addOrganizationMemberV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := pathOrganizationID(w, r)
	if !ok {
		return
	}
	actor, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decode(w, r, &req) {
		return
	}
	if err := validation.Email(req.Email); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Role == "" {
		req.Role = authz.RoleViewer
	}
	if !authz.CanGrantRole(actor.Role, req.Role) {
		errorJSON(w, r, 403, "FORBIDDEN", "you cannot grant that role")
		return
	}
	if a.cfRepo != nil {
		member, err := a.cfRepo.AddOrganizationMember(r.Context(), orgID, req.Email, req.Role)
		if err == repository.ErrDuplicateMember {
			errorJSON(w, r, 409, "CONFLICT", "user is already a member")
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			errorJSON(w, r, 404, "NOT_FOUND", "user not found")
			return
		}
		if err != nil {
			errorJSON(w, r, 500, "DATABASE_ERROR", "could not add member")
			return
		}
		a.writeAudit(r.Context(), r, orgID, userID(r), "organization.member_added", "user", member.UserID, `{"role":"`+member.Role+`"}`)
		writeJSON(w, 201, member)
		return
	}
	member, err := a.addMemoryMemberByEmail(orgID, req.Email, req.Role)
	if err == repository.ErrDuplicateMember {
		errorJSON(w, r, 409, "CONFLICT", "user is already a member")
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		errorJSON(w, r, 404, "NOT_FOUND", "user not found")
		return
	}
	if err != nil {
		errorJSON(w, r, 500, "INTERNAL_ERROR", "could not add member")
		return
	}
	a.writeAudit(r.Context(), r, orgID, userID(r), "organization.member_added", "user", member.UserID, `{"role":"`+member.Role+`"}`)
	writeJSON(w, 201, member)
}

func (a *app) updateOrganizationMemberV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := pathOrganizationID(w, r)
	if !ok {
		return
	}
	targetUserID := r.PathValue("userId")
	if err := validation.UUID(targetUserID, "userId"); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	actor, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers)
	if !ok {
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &req) {
		return
	}
	if !authz.CanGrantRole(actor.Role, req.Role) {
		errorJSON(w, r, 403, "FORBIDDEN", "you cannot grant that role")
		return
	}
	if actor.Role == authz.RoleAdmin {
		current, err := a.membership(r.Context(), targetUserID, orgID)
		if err == nil && current.Role == authz.RoleOwner {
			errorJSON(w, r, 403, "FORBIDDEN", "admin cannot modify an owner")
			return
		}
	}
	member, err := a.updateMemberRole(r.Context(), orgID, targetUserID, req.Role)
	if err == repository.ErrLastOwner {
		errorJSON(w, r, 409, "CONFLICT", "cannot remove the last owner")
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		errorJSON(w, r, 404, "NOT_FOUND", "member not found")
		return
	}
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not update member")
		return
	}
	a.writeAudit(r.Context(), r, orgID, userID(r), "organization.member_role_changed", "user", member.UserID, `{"role":"`+member.Role+`"}`)
	writeJSON(w, 200, member)
}

func (a *app) deleteOrganizationMemberV1(w http.ResponseWriter, r *http.Request) {
	orgID, ok := pathOrganizationID(w, r)
	if !ok {
		return
	}
	targetUserID := r.PathValue("userId")
	if err := validation.UUID(targetUserID, "userId"); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return
	}
	actor, ok := a.requireOrgRole(w, r, orgID, authz.CanManageMembers)
	if !ok {
		return
	}
	if actor.Role == authz.RoleAdmin {
		current, err := a.membership(r.Context(), targetUserID, orgID)
		if err == nil && current.Role == authz.RoleOwner {
			errorJSON(w, r, 403, "FORBIDDEN", "admin cannot remove an owner")
			return
		}
	}
	err := a.removeMember(r.Context(), orgID, targetUserID)
	if err == repository.ErrLastOwner {
		errorJSON(w, r, 409, "CONFLICT", "cannot remove the last owner")
		return
	}
	if errors.Is(err, repository.ErrNotFound) {
		errorJSON(w, r, 404, "NOT_FOUND", "member not found")
		return
	}
	if err != nil {
		errorJSON(w, r, 500, "DATABASE_ERROR", "could not remove member")
		return
	}
	a.writeAudit(r.Context(), r, orgID, userID(r), "organization.member_removed", "user", targetUserID, "{}")
	w.WriteHeader(http.StatusNoContent)
}

func pathOrganizationID(w http.ResponseWriter, r *http.Request) (string, bool) {
	orgID := r.PathValue("organizationId")
	if err := validation.UUID(orgID, "organizationId"); err != nil {
		errorJSON(w, r, 400, "VALIDATION_ERROR", err.Error())
		return "", false
	}
	return orgID, true
}

func publicUser(u models.User) map[string]string {
	return map[string]string{"id": u.ID, "email": u.Email, "name": displayName(u.Email)}
}

func displayName(email string) string {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i]
	}
	return email
}

func membershipOrganizations(memberships []models.OrganizationMember) []map[string]interface{} {
	out := []map[string]interface{}{}
	for _, member := range memberships {
		out = append(out, map[string]interface{}{
			"id":       member.OrganizationID,
			"name":     member.OrganizationName,
			"type":     member.OrganizationType,
			"currency": member.Currency,
			"role":     member.Role,
		})
	}
	return out
}

func (a *app) userMemberships(ctx context.Context, uid string) ([]models.OrganizationMember, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListUserMemberships(ctx, uid)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	return a.userMembershipsLocked(uid), nil
}

func (a *app) organizationMembers(ctx context.Context, orgID string) ([]models.OrganizationMember, error) {
	if a.cfRepo != nil {
		return a.cfRepo.ListOrganizationMembers(ctx, orgID)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	out := []models.OrganizationMember{}
	for _, member := range a.store.organizationMembers {
		if member.OrganizationID == orgID {
			out = append(out, a.hydrateMemberLocked(member))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (a *app) membership(ctx context.Context, uid, orgID string) (models.OrganizationMember, error) {
	if a.cfRepo != nil {
		return a.cfRepo.GetMembership(ctx, uid, orgID)
	}
	a.store.mu.RLock()
	defer a.store.mu.RUnlock()
	for _, member := range a.store.organizationMembers {
		if member.UserID == uid && member.OrganizationID == orgID {
			return a.hydrateMemberLocked(member), nil
		}
	}
	return models.OrganizationMember{}, repository.ErrNotFound
}

func (a *app) addMemoryMemberByEmail(orgID, email, role string) (models.OrganizationMember, error) {
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	uid, ok := a.store.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return models.OrganizationMember{}, repository.ErrNotFound
	}
	for _, member := range a.store.organizationMembers {
		if member.OrganizationID == orgID && member.UserID == uid {
			return models.OrganizationMember{}, repository.ErrDuplicateMember
		}
	}
	member := a.addOrganizationMemberLocked(orgID, uid, role)
	return a.hydrateMemberLocked(member), nil
}

func (a *app) updateMemberRole(ctx context.Context, orgID, targetUserID, role string) (models.OrganizationMember, error) {
	if a.cfRepo != nil {
		return a.cfRepo.UpdateOrganizationMemberRole(ctx, orgID, targetUserID, role)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	var foundID string
	for id, member := range a.store.organizationMembers {
		if member.OrganizationID == orgID && member.UserID == targetUserID {
			foundID = id
			if member.Role == authz.RoleOwner && role != authz.RoleOwner && a.otherOwnerCountLocked(orgID, targetUserID) == 0 {
				return models.OrganizationMember{}, repository.ErrLastOwner
			}
			member.Role = role
			a.store.organizationMembers[id] = member
			return a.hydrateMemberLocked(member), nil
		}
	}
	if foundID == "" {
		return models.OrganizationMember{}, repository.ErrNotFound
	}
	return models.OrganizationMember{}, nil
}

func (a *app) removeMember(ctx context.Context, orgID, targetUserID string) error {
	if a.cfRepo != nil {
		return a.cfRepo.RemoveOrganizationMember(ctx, orgID, targetUserID)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for id, member := range a.store.organizationMembers {
		if member.OrganizationID == orgID && member.UserID == targetUserID {
			if member.Role == authz.RoleOwner && a.otherOwnerCountLocked(orgID, targetUserID) == 0 {
				return repository.ErrLastOwner
			}
			delete(a.store.organizationMembers, id)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (a *app) userMembershipsLocked(uid string) []models.OrganizationMember {
	out := []models.OrganizationMember{}
	for _, member := range a.store.organizationMembers {
		if member.UserID == uid {
			out = append(out, a.hydrateMemberLocked(member))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (a *app) ensureOrganizationLocked(uid, name string) models.Organization {
	for _, org := range a.store.organizations {
		if org.UserID == uid {
			return org
		}
	}
	if name == "" {
		name = "Clearflow Demo Organization"
	}
	now := time.Now().UTC()
	org := models.Organization{ID: auth.NewID(), UserID: uid, Name: name, Type: "student_organization", Currency: "USD", CreatedAt: now, UpdatedAt: now}
	a.store.organizations[org.ID] = org
	a.auditLocked(org.ID, uid, "organization.created", "organization", org.ID)
	return org
}

func (a *app) addOrganizationMemberLocked(orgID, uid, role string) models.OrganizationMember {
	for _, member := range a.store.organizationMembers {
		if member.OrganizationID == orgID && member.UserID == uid {
			return member
		}
	}
	member := models.OrganizationMember{ID: auth.NewID(), OrganizationID: orgID, UserID: uid, Role: role, CreatedAt: time.Now().UTC()}
	a.store.organizationMembers[member.ID] = member
	return member
}

func (a *app) hydrateMemberLocked(member models.OrganizationMember) models.OrganizationMember {
	if u, ok := a.store.users[member.UserID]; ok {
		member.Email = u.Email
	}
	if org, ok := a.store.organizations[member.OrganizationID]; ok {
		member.OrganizationName = org.Name
		member.OrganizationType = org.Type
		member.Currency = org.Currency
	}
	return member
}

func (a *app) otherOwnerCountLocked(orgID, targetUserID string) int {
	count := 0
	for _, member := range a.store.organizationMembers {
		if member.OrganizationID == orgID && member.UserID != targetUserID && member.Role == authz.RoleOwner {
			count++
		}
	}
	return count
}
