package pearcut

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS experiments (
			slug        TEXT PRIMARY KEY,
			status      TEXT NOT NULL DEFAULT 'draft',
			variants    TEXT NOT NULL DEFAULT '[]',
			overrides   TEXT NOT NULL DEFAULT '{}',
			seed            TEXT NOT NULL DEFAULT '',
			targeting_rules TEXT NOT NULL DEFAULT '[]',
			description     TEXT NOT NULL DEFAULT '',
			tags        TEXT NOT NULL DEFAULT '[]',
			owner       TEXT NOT NULL DEFAULT '',
			hypothesis  TEXT NOT NULL DEFAULT '',
			exclusion_percentage INTEGER NOT NULL DEFAULT 0,
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		);
	`)
	return err
}

func (s *SQLiteStore) Get(slug string) (Experiment, error) {
	row := s.db.QueryRow(
		"SELECT slug, status, seed, variants, overrides, targeting_rules, description, tags, owner, hypothesis, exclusion_percentage, created_at, updated_at FROM experiments WHERE slug = ?",
		slug,
	)

	exp, err := scanExperiment(row)
	if err == sql.ErrNoRows {
		return Experiment{}, ErrExperimentNotFound
	}
	if err != nil {
		return Experiment{}, fmt.Errorf("get experiment: %w", err)
	}
	return exp, nil
}

func (s *SQLiteStore) List(filter ExperimentFilter, opts ListOptions) (ExperimentListResult, error) {
	where, args := buildWhereClause(filter)

	var total int
	countQuery := "SELECT COUNT(*) FROM experiments" + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return ExperimentListResult{}, fmt.Errorf("count experiments: %w", err)
	}

	query := "SELECT slug, status, seed, variants, overrides, targeting_rules, description, tags, owner, hypothesis, exclusion_percentage, created_at, updated_at FROM experiments" + where + " ORDER BY created_at DESC"
	if opts.Page > 0 && opts.PerPage > 0 {
		offset := (opts.Page - 1) * opts.PerPage
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", opts.PerPage, offset)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return ExperimentListResult{}, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()

	experiments := []Experiment{}
	for rows.Next() {
		var exp Experiment
		var variantsJSON, overridesJSON, targetingRulesJSON, tagsJSON, createdAt, updatedAt string
		if err := rows.Scan(&exp.Slug, &exp.Status, &exp.Seed, &variantsJSON, &overridesJSON, &targetingRulesJSON, &exp.Description, &tagsJSON, &exp.Owner, &exp.Hypothesis, &exp.ExclusionPercentage, &createdAt, &updatedAt); err != nil {
			return ExperimentListResult{}, fmt.Errorf("scan experiment: %w", err)
		}
		if err := json.Unmarshal([]byte(variantsJSON), &exp.Variants); err != nil {
			return ExperimentListResult{}, fmt.Errorf("decode variants for %q: %w", exp.Slug, err)
		}
		if err := json.Unmarshal([]byte(overridesJSON), &exp.Overrides); err != nil {
			return ExperimentListResult{}, fmt.Errorf("decode overrides for %q: %w", exp.Slug, err)
		}
		if err := json.Unmarshal([]byte(targetingRulesJSON), &exp.TargetingRules); err != nil {
			return ExperimentListResult{}, fmt.Errorf("decode targeting rules for %q: %w", exp.Slug, err)
		}
		if err := json.Unmarshal([]byte(tagsJSON), &exp.Tags); err != nil {
			return ExperimentListResult{}, fmt.Errorf("decode tags for %q: %w", exp.Slug, err)
		}
		exp.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		exp.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		experiments = append(experiments, exp)
	}
	if err := rows.Err(); err != nil {
		return ExperimentListResult{}, fmt.Errorf("iterate experiments: %w", err)
	}

	return ExperimentListResult{Experiments: experiments, Total: total}, nil
}

func buildWhereClause(filter ExperimentFilter) (string, []any) {
	var conditions []string
	var args []any

	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, string(*filter.Status))
	}
	for _, tag := range filter.Tags {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM json_each(tags) WHERE json_each.value = ?)")
		args = append(args, tag)
	}

	if len(conditions) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func (s *SQLiteStore) Create(exp Experiment) error {
	if exp.Seed == "" {
		exp.Seed = exp.Slug
	}
	if err := exp.Validate(); err != nil {
		return err
	}

	now := time.Now().UTC()
	exp.CreatedAt = now
	exp.UpdatedAt = now

	variantsJSON, err := json.Marshal(exp.Variants)
	if err != nil {
		return fmt.Errorf("encode variants: %w", err)
	}
	overridesJSON, err := json.Marshal(exp.Overrides)
	if err != nil {
		return fmt.Errorf("encode overrides: %w", err)
	}
	targetingRulesJSON, err := json.Marshal(exp.TargetingRules)
	if err != nil {
		return fmt.Errorf("encode targeting rules: %w", err)
	}
	tagsJSON, err := json.Marshal(exp.Tags)
	if err != nil {
		return fmt.Errorf("encode tags: %w", err)
	}

	_, err = s.db.Exec(
		"INSERT INTO experiments (slug, status, seed, variants, overrides, targeting_rules, description, tags, owner, hypothesis, exclusion_percentage, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		exp.Slug, string(exp.Status), exp.Seed,
		string(variantsJSON), string(overridesJSON), string(targetingRulesJSON),
		exp.Description, string(tagsJSON), exp.Owner, exp.Hypothesis,
		exp.ExclusionPercentage,
		exp.CreatedAt.Format(time.RFC3339), exp.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrExperimentExists
		}
		return fmt.Errorf("insert experiment: %w", err)
	}

	return nil
}

func (s *SQLiteStore) Update(exp Experiment) error {
	if exp.Seed == "" {
		exp.Seed = exp.Slug
	}
	if err := exp.Validate(); err != nil {
		return err
	}

	exp.UpdatedAt = time.Now().UTC()

	variantsJSON, err := json.Marshal(exp.Variants)
	if err != nil {
		return fmt.Errorf("encode variants: %w", err)
	}
	overridesJSON, err := json.Marshal(exp.Overrides)
	if err != nil {
		return fmt.Errorf("encode overrides: %w", err)
	}
	targetingRulesJSON, err := json.Marshal(exp.TargetingRules)
	if err != nil {
		return fmt.Errorf("encode targeting rules: %w", err)
	}
	tagsJSON, err := json.Marshal(exp.Tags)
	if err != nil {
		return fmt.Errorf("encode tags: %w", err)
	}

	res, err := s.db.Exec(
		"UPDATE experiments SET status = ?, seed = ?, variants = ?, overrides = ?, targeting_rules = ?, description = ?, tags = ?, owner = ?, hypothesis = ?, exclusion_percentage = ?, updated_at = ? WHERE slug = ?",
		string(exp.Status), exp.Seed,
		string(variantsJSON), string(overridesJSON), string(targetingRulesJSON),
		exp.Description, string(tagsJSON), exp.Owner, exp.Hypothesis,
		exp.ExclusionPercentage,
		exp.UpdatedAt.Format(time.RFC3339), exp.Slug,
	)
	if err != nil {
		return fmt.Errorf("update experiment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrExperimentNotFound
	}

	return nil
}

func (s *SQLiteStore) Delete(slug string) error {
	res, err := s.db.Exec("DELETE FROM experiments WHERE slug = ?", slug)
	if err != nil {
		return fmt.Errorf("delete experiment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrExperimentNotFound
	}
	return nil
}

// helpers

func scanExperiment(row *sql.Row) (Experiment, error) {
	var exp Experiment
	var variantsJSON, overridesJSON, targetingRulesJSON, tagsJSON, createdAt, updatedAt string
	if err := row.Scan(&exp.Slug, &exp.Status, &exp.Seed, &variantsJSON, &overridesJSON, &targetingRulesJSON, &exp.Description, &tagsJSON, &exp.Owner, &exp.Hypothesis, &exp.ExclusionPercentage, &createdAt, &updatedAt); err != nil {
		return Experiment{}, err
	}
	if err := json.Unmarshal([]byte(variantsJSON), &exp.Variants); err != nil {
		return Experiment{}, fmt.Errorf("decode variants: %w", err)
	}
	if err := json.Unmarshal([]byte(overridesJSON), &exp.Overrides); err != nil {
		return Experiment{}, fmt.Errorf("decode overrides: %w", err)
	}
	if err := json.Unmarshal([]byte(targetingRulesJSON), &exp.TargetingRules); err != nil {
		return Experiment{}, fmt.Errorf("decode targeting rules: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsJSON), &exp.Tags); err != nil {
		return Experiment{}, fmt.Errorf("decode tags: %w", err)
	}
	exp.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	exp.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return exp, nil
}
