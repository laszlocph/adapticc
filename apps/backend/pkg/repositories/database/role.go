package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/followthepattern/adapticc/pkg/models"
	"github.com/followthepattern/adapticc/pkg/repositories/database/sqlbuilder"
	"github.com/followthepattern/adapticc/pkg/types"
	. "github.com/followthepattern/goqu/v9"
	"github.com/followthepattern/goqu/v9/exp"
)

var (
	roleTableName     = S("usr").Table("roles")
	userRoleTableName = S("usr").Table("user_role")
)

type Role struct {
	db *Database
}

func NewRole(database *sql.DB) Role {
	db := New("postgres", database)

	return Role{
		db: db,
	}
}

func (repo Role) GetByID(id string) (*models.Role, error) {
	var data models.Role

	_, err := repo.db.From(roleTableName).
		Where(Ex{"id": id}).
		ScanStruct(&data)

	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (repo Role) Get(request models.RoleListRequestParams) (*models.RoleListResponse, error) {
	data := []models.Role{}

	query := repo.db.From(roleTableName)

	if request.Filter.Search.IsValid() {
		pattern := fmt.Sprintf("%%%s%%", request.Filter.Search)
		query = query.Where(
			Or(
				I("id").Like(pattern),
				I("name").Like(pattern),
				I("code").Like(pattern),
			))
	}

	count, err := query.Count()
	if err != nil {
		return nil, err
	}

	query = sqlbuilder.WithPagination(query, request.Pagination)

	query = sqlbuilder.WithOrderBy(query, request.OrderBy)

	err = query.ScanStructs(&data)
	if err != nil {
		return nil, err
	}

	result := models.RoleListResponse{
		Count:    types.Int64From(count),
		PageSize: request.Pagination.PageSize,
		Page:     request.Pagination.Page,
		Data:     data,
	}

	return &result, nil
}

func (repo Role) GetRolesByUserID(userID types.String) ([]models.Role, error) {
	var data []models.Role

	err := repo.db.From(userRoleTableName.As("ur")).
		Join(roleTableName.As("r"),
			On(Ex{"r.id": I("ur.role_id")})).
		Where(Ex{"user_id": userID}).
		Select(T("r").Col(exp.Star())).
		ScanStructs(&data)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (repo Role) AddRoleToUser(values []models.UserRole) error {
	for i, _ := range values {
		values[i].Userlog.CreatedAt = types.TimeNow()
	}

	insertion := repo.db.Insert(userRoleTableName)

	_, err := insertion.Rows(values).Executor().Exec()
	return err
}

func (repo Role) RemoveRoleFromUser(value models.UserRole) error {
	res, err := repo.db.
		Delete(userRoleTableName).
		Where(Ex{
			"user_id": value.UserID,
			"role_id": value.RoleID}).
		Executor().
		Exec()

	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rows < 1 {
		return errors.New("no rows been deleted")
	}

	return err
}

func (repo Role) GetRoleCodes(userID types.String) ([]string, error) {
	roles, err := repo.GetRolesByUserID(userID)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(roles))

	for i, role := range roles {
		result[i] = role.Code.Data
	}

	return result, nil
}
