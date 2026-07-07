package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/StephenShao90/Fynora/services/api/internal/auth"
	"github.com/StephenShao90/Fynora/services/api/internal/authz"
	"github.com/StephenShao90/Fynora/services/api/internal/models"
)

var (
	ErrDuplicateEmail  = errors.New("email is already registered")
	ErrDuplicateMember = errors.New("user is already a member of this organization")
	ErrLastOwner       = errors.New("cannot remove the last owner")
	ErrNotFound        = sql.ErrNoRows
)

func (r *ClearflowRepository) CreateUserWithDefaultOrganization(ctx context.Context, email, passwordHash, organizationName string) (models.User, []models.OrganizationMember, error) {
	now := time.Now().UTC()
	user := models.User{ID: auth.NewID(), Email: strings.ToLower(strings.TrimSpace(email)), PasswordHash: passwordHash, CreatedAt: now}
	if organizationName == "" {
		organizationName = "Default Organization"
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return models.User{}, nil, err
	}
	defer rollback(tx)
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, email, password_hash, created_at) VALUES ($1, $2, $3, $4)`, user.ID, user.Email, user.PasswordHash, user.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "users_email_key") || strings.Contains(err.Error(), "duplicate key") {
			return models.User{}, nil, ErrDuplicateEmail
		}
		return models.User{}, nil, err
	}
	org := models.Organization{ID: auth.NewID(), UserID: user.ID, Name: organizationName, Type: "small_business", Currency: "USD", CreatedAt: now, UpdatedAt: now}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organizations (id, user_id, name, type, currency, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, org.ID, org.UserID, org.Name, org.Type, org.Currency, org.CreatedAt, org.UpdatedAt); err != nil {
		return models.User{}, nil, err
	}
	member := models.OrganizationMember{ID: auth.NewID(), OrganizationID: org.ID, UserID: user.ID, Role: authz.RoleOwner, CreatedAt: now, OrganizationName: org.Name, OrganizationType: org.Type, Currency: org.Currency, Email: user.Email}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO organization_members (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, member.ID, member.OrganizationID, member.UserID, member.Role, member.CreatedAt); err != nil {
		return models.User{}, nil, err
	}
	if err := insertAudit(ctx, tx, org.ID, user.ID, "organization.created", "organization", org.ID); err != nil {
		return models.User{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return models.User{}, nil, err
	}
	return user, []models.OrganizationMember{member}, nil
}

func (r *ClearflowRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := r.db.QueryRowContext(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE lower(email) = lower($1)`, strings.TrimSpace(email)).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return user, err
}

func (r *ClearflowRepository) GetUserByID(ctx context.Context, id string) (models.User, error) {
	var user models.User
	err := r.db.QueryRowContext(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return user, err
}

func (r *ClearflowRepository) ListUserMemberships(ctx context.Context, userID string) ([]models.OrganizationMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, o.name, o.type, o.currency, u.email
		FROM organization_members m
		JOIN organizations o ON o.id = m.organization_id
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = $1
		ORDER BY o.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMembers(rows)
}

func (r *ClearflowRepository) GetMembership(ctx context.Context, userID, orgID string) (models.OrganizationMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, o.name, o.type, o.currency, u.email
		FROM organization_members m
		JOIN organizations o ON o.id = m.organization_id
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = $1 AND m.organization_id = $2
	`, userID, orgID)
	if err != nil {
		return models.OrganizationMember{}, err
	}
	defer rows.Close()
	members, err := scanMembers(rows)
	if err != nil {
		return models.OrganizationMember{}, err
	}
	if len(members) == 0 {
		return models.OrganizationMember{}, ErrNotFound
	}
	return members[0], nil
}

func (r *ClearflowRepository) ListOrganizationMembers(ctx context.Context, orgID string) ([]models.OrganizationMember, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.organization_id, m.user_id, m.role, m.created_at, o.name, o.type, o.currency, u.email
		FROM organization_members m
		JOIN organizations o ON o.id = m.organization_id
		JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY m.created_at
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMembers(rows)
}

func (r *ClearflowRepository) AddOrganizationMember(ctx context.Context, orgID, email, role string) (models.OrganizationMember, error) {
	user, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		return models.OrganizationMember{}, err
	}
	now := time.Now().UTC()
	member := models.OrganizationMember{ID: auth.NewID(), OrganizationID: orgID, UserID: user.ID, Role: role, CreatedAt: now, Email: user.Email}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO organization_members (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, member.ID, member.OrganizationID, member.UserID, member.Role, member.CreatedAt); err != nil {
		if strings.Contains(err.Error(), "organization_members_organization_id_user_id_key") || strings.Contains(err.Error(), "duplicate key") {
			return models.OrganizationMember{}, ErrDuplicateMember
		}
		return models.OrganizationMember{}, err
	}
	return r.GetMembership(ctx, user.ID, orgID)
}

func (r *ClearflowRepository) UpdateOrganizationMemberRole(ctx context.Context, orgID, targetUserID, role string) (models.OrganizationMember, error) {
	current, err := r.GetMembership(ctx, targetUserID, orgID)
	if err != nil {
		return models.OrganizationMember{}, err
	}
	if current.Role == authz.RoleOwner && role != authz.RoleOwner {
		if err := r.ensureNotLastOwner(ctx, orgID, targetUserID); err != nil {
			return models.OrganizationMember{}, err
		}
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE organization_members SET role = $1 WHERE organization_id = $2 AND user_id = $3`, role, orgID, targetUserID); err != nil {
		return models.OrganizationMember{}, err
	}
	return r.GetMembership(ctx, targetUserID, orgID)
}

func (r *ClearflowRepository) RemoveOrganizationMember(ctx context.Context, orgID, targetUserID string) error {
	current, err := r.GetMembership(ctx, targetUserID, orgID)
	if err != nil {
		return err
	}
	if current.Role == authz.RoleOwner {
		if err := r.ensureNotLastOwner(ctx, orgID, targetUserID); err != nil {
			return err
		}
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM organization_members WHERE organization_id = $1 AND user_id = $2`, orgID, targetUserID)
	return err
}

func (r *ClearflowRepository) ensureNotLastOwner(ctx context.Context, orgID, targetUserID string) error {
	var owners int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organization_members WHERE organization_id = $1 AND role = 'owner' AND user_id <> $2`, orgID, targetUserID).Scan(&owners); err != nil {
		return err
	}
	if owners == 0 {
		return ErrLastOwner
	}
	return nil
}

func scanMembers(rows *sql.Rows) ([]models.OrganizationMember, error) {
	out := []models.OrganizationMember{}
	for rows.Next() {
		var member models.OrganizationMember
		if err := rows.Scan(&member.ID, &member.OrganizationID, &member.UserID, &member.Role, &member.CreatedAt, &member.OrganizationName, &member.OrganizationType, &member.Currency, &member.Email); err != nil {
			return nil, err
		}
		out = append(out, member)
	}
	return out, rows.Err()
}
