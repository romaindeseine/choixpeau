package pearcut

import (
	"sort"
	"strings"
)

func hasAllTags(expTags []string, required []string) bool {
	set := make(map[string]struct{}, len(expTags))
	for _, t := range expTags {
		set[t] = struct{}{}
	}
	for _, t := range required {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

func filterExperiments(exps []Experiment, filter ExperimentFilter) []Experiment {
	result := make([]Experiment, 0, len(exps))

	for _, exp := range exps {
		if filter.Status != nil && exp.Status != *filter.Status {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(exp.Slug), strings.ToLower(filter.Search)) {
			continue
		}
		if len(filter.Tags) > 0 && !hasAllTags(exp.Tags, filter.Tags) {
			continue
		}
		result = append(result, exp)
	}

	return result
}

func sortExperiments(exps []Experiment, opts ListOptions) {
	sort.Slice(exps, func(i, j int) bool {
		var less bool
		switch opts.Sort {
		case "created_at":
			less = exps[i].CreatedAt.Before(exps[j].CreatedAt)
		case "updated_at":
			less = exps[i].UpdatedAt.Before(exps[j].UpdatedAt)
		default:
			less = exps[i].Slug < exps[j].Slug
		}
		if opts.Order == "desc" {
			return !less
		}
		return less
	})
}

func paginateExperiments(exps []Experiment, opts ListOptions) ([]Experiment, int) {
	total := len(exps)

	if opts.PerPage <= 0 || opts.Page <= 0 {
		return exps, total
	}

	offset := (opts.Page - 1) * opts.PerPage
	if offset > len(exps) {
		return []Experiment{}, total
	}
	end := offset + opts.PerPage
	if end > len(exps) {
		end = len(exps)
	}

	return exps[offset:end], total
}
