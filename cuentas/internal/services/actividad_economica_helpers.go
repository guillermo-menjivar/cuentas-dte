package services

import (
	"cuentas/internal/codigos"
	"sort"
)

func sortCategoriesByCode(categories []codigos.ActivityCategory) []codigos.ActivityCategory {
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Code < categories[j].Code
	})
	return categories
}

func sortActivitiesByCode(activities []codigos.EconomicActivity) []codigos.EconomicActivity {
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Code < activities[j].Code
	})
	return activities
}

func sortByRelevance(results []SearchResult) []SearchResult {
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})
	return results
}
