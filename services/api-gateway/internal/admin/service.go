package admin

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// AdminService abstracts all gateway-state persistence so Handler never touches SQL directly.
type AdminService interface {
	ListRoutes(ctx context.Context) ([]routeRow, error)
	CreateRoute(ctx context.Context, pathPrefix, upstream, authMode string, stripPrefix bool) (string, error)
	UpdateRoute(ctx context.Context, id string, req updateRouteReq) (bool, error)
	DisableRoute(ctx context.Context, id string) (bool, error)
	QueryAudit(ctx context.Context, f auditFilters) ([]auditRow, error)
	RouteCount(ctx context.Context) (active, total int, error error)
	RecentAudit(ctx context.Context, limit int) ([]auditRow, error)
	FindRouteUpstream(ctx context.Context, path string) (string, error)
}

type auditFilters struct {
	UserID  string
	Event   string
	From    string
	To      string
	AfterID string
	Limit   int
}

// postgresAdminService implements AdminService against a PostgreSQL database.
type postgresAdminService struct {
	db *sql.DB
}

func NewPostgresAdminService(db *sql.DB) AdminService {
	return &postgresAdminService{db: db}
}

func (s *postgresAdminService) ListRoutes(ctx context.Context) ([]routeRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path_prefix, upstream, auth_mode, strip_prefix, enabled, created_at, updated_at
		 FROM gateway_routes ORDER BY length(path_prefix) DESC, path_prefix`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []routeRow
	for rows.Next() {
		var row routeRow
		if err := rows.Scan(&row.ID, &row.PathPrefix, &row.Upstream, &row.AuthMode,
			&row.StripPrefix, &row.Enabled, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *postgresAdminService) CreateRoute(ctx context.Context, pathPrefix, upstream, authMode string, stripPrefix bool) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO gateway_routes (path_prefix, upstream, auth_mode, strip_prefix)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		pathPrefix, upstream, authMode, stripPrefix,
	).Scan(&id)
	return id, err
}

func (s *postgresAdminService) UpdateRoute(ctx context.Context, id string, req updateRouteReq) (bool, error) {
	sets := []string{}
	args := []any{}
	idx := 1
	if req.Upstream != nil {
		sets = append(sets, "upstream = $"+strconv.Itoa(idx))
		args = append(args, *req.Upstream)
		idx++
	}
	if req.AuthMode != nil {
		sets = append(sets, "auth_mode = $"+strconv.Itoa(idx))
		args = append(args, *req.AuthMode)
		idx++
	}
	if req.StripPrefix != nil {
		sets = append(sets, "strip_prefix = $"+strconv.Itoa(idx))
		args = append(args, *req.StripPrefix)
		idx++
	}
	if req.Enabled != nil {
		sets = append(sets, "enabled = $"+strconv.Itoa(idx))
		args = append(args, *req.Enabled)
		idx++
	}
	if len(sets) == 0 {
		return false, nil
	}

	var sb strings.Builder
	sb.WriteString("UPDATE gateway_routes SET ")
	for i, s := range sets {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(s)
	}
	sb.WriteString(" WHERE id = $")
	sb.WriteString(strconv.Itoa(idx))
	args = append(args, id)

	res, err := s.db.ExecContext(ctx, sb.String(), args...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *postgresAdminService) DisableRoute(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE gateway_routes SET enabled = false WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *postgresAdminService) QueryAudit(ctx context.Context, f auditFilters) ([]auditRow, error) {
	args := []any{}
	where := "WHERE 1=1"
	idx := 1
	if f.UserID != "" {
		where += " AND user_id = $" + strconv.Itoa(idx)
		args = append(args, f.UserID)
		idx++
	}
	if f.Event != "" {
		where += " AND event = $" + strconv.Itoa(idx)
		args = append(args, f.Event)
		idx++
	}
	if f.From != "" {
		where += " AND ts >= $" + strconv.Itoa(idx)
		args = append(args, f.From)
		idx++
	}
	if f.To != "" {
		where += " AND ts <= $" + strconv.Itoa(idx)
		args = append(args, f.To)
		idx++
	}
	if f.AfterID != "" {
		where += " AND id > $" + strconv.Itoa(idx)
		args = append(args, f.AfterID)
		idx++
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
	args = append(args, f.Limit)

	query := `SELECT id, ts, request_id, user_id, ip, method, path, upstream, status_code, event, detail
			  FROM gateway_audit_log ` + where + ` ORDER BY ts DESC LIMIT $` + strconv.Itoa(idx)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []auditRow
	for rows.Next() {
		var row auditRow
		if err := rows.Scan(&row.ID, &row.Ts, &row.RequestID, &row.UserID, &row.IP,
			&row.Method, &row.Path, &row.Upstream, &row.StatusCode, &row.Event, &row.Detail); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (s *postgresAdminService) RouteCount(ctx context.Context) (active, total int, err error) {
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gateway_routes`).Scan(&total); err != nil {
		return
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM gateway_routes WHERE enabled = true`).Scan(&active)
	return
}

func (s *postgresAdminService) RecentAudit(ctx context.Context, limit int) ([]auditRow, error) {
	return s.QueryAudit(ctx, auditFilters{Limit: limit})
}

func (s *postgresAdminService) FindRouteUpstream(ctx context.Context, path string) (string, error) {
	var upstream string
	err := s.db.QueryRowContext(ctx,
		`SELECT upstream FROM gateway_routes
		 WHERE enabled = true AND ($1 = path_prefix OR $1 LIKE path_prefix || '/%')
		 ORDER BY length(path_prefix) DESC LIMIT 1`,
		path,
	).Scan(&upstream)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no enabled route matches %s", path)
	}
	return upstream, err
}
