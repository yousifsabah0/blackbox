package data

import (
	"context"
	"database/sql"

	"github.com/lib/pq"
)

type Permissions []string

func (p Permissions) Contains(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}

	return false
}

type PermissionModel struct {
	DB *sql.DB
}

func (p PermissionModel) GrantUser(userID int64, codes ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					INSERT INTO users_permissions
					SELECT $1, permissions.id FROM permissions WHERE
					permissions.code = ANY($2)
	`

	_, err := p.DB.ExecContext(ctx, query, userID, pq.Array(codes))
	return err
}

func (p PermissionModel) GetUserPermissions(userID int64) (Permissions, error) {
	var permissions Permissions

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	query := `
					SELECT permissions.code FROM permissions
					INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
					INNER JOIN users ON users_permissions.user_id = users.id
					WHERE users.id = $1
	`
	rows, err := p.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}

		permissions = append(permissions, permission)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
