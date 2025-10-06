package services

import (
	"cuentas/internal/codigos"
	"strings"
)

type ActividadEconomicaService struct{}

func NewActividadEconomicaService() *ActividadEconomicaService {
	return &ActividadEconomicaService{}
}

// GetAllCategories returns all top-level categories (Level 1)
func (s *ActividadEconomicaService) GetAllCategories() []codigos.ActivityCategory {
	categories := make([]codigos.ActivityCategory, 0)

	for _, cat := range codigos.ActividadEconomicaCategories {
		if cat.Level == 1 {
			categories = append(categories, cat)
		}
	}

	// Sort by code for consistent ordering
	return sortCategoriesByCode(categories)
}

// GetCategoryByCode returns a specific category
func (s *ActividadEconomicaService) GetCategoryByCode(code string) (*codigos.ActivityCategory, bool) {
	cat, exists := codigos.ActividadEconomicaCategories[code]
	return &cat, exists
}

// GetActivitiesByCategory returns all activities that belong to a category
// For example, code "46" returns all activities starting with "46"
func (s *ActividadEconomicaService) GetActivitiesByCategory(categoryCode string) []codigos.EconomicActivity {
	activities := make([]codigos.EconomicActivity, 0)

	for code, name := range codigos.EconomicActivities {
		// Check if activity code starts with category code
		if strings.HasPrefix(code, categoryCode) {
			activities = append(activities, codigos.EconomicActivity{
				Code:  code,
				Value: name,
			})
		}
	}

	return sortActivitiesByCode(activities)
}

// SearchActivities performs a search across both categories and activities
// Returns ranked results based on relevance
func (s *ActividadEconomicaService) SearchActivities(query string, limit int) []SearchResult {
	if query == "" || limit <= 0 {
		return []SearchResult{}
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	results := make([]SearchResult, 0)

	// Search in activity codes and names
	for code, name := range codigos.EconomicActivities {
		score := calculateRelevanceScore(queryLower, code, name)
		if score > 0 {
			results = append(results, SearchResult{
				Code:           code,
				Name:           name,
				Type:           "activity",
				RelevanceScore: score,
			})
		}
	}

	// Search in categories (keywords and names)
	for code, cat := range codigos.ActividadEconomicaCategories {
		score := calculateCategoryRelevance(queryLower, code, cat)
		if score > 0 {
			results = append(results, SearchResult{
				Code:           code,
				Name:           cat.Name,
				Type:           "category",
				RelevanceScore: score,
			})
		}
	}

	// Sort by relevance score (highest first)
	results = sortByRelevance(results)

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}

// GetActivityDetails returns full details of a specific activity including its category
func (s *ActividadEconomicaService) GetActivityDetails(code string) (*ActivityDetails, bool) {
	name, exists := codigos.EconomicActivities[code]
	if !exists {
		return nil, false
	}

	// Find the category this activity belongs to
	categoryCode := getCategoryCodeFromActivity(code)
	category, _ := codigos.ActividadEconomicaCategories[categoryCode]

	// Find related activities (same category)
	related := s.GetActivitiesByCategory(categoryCode)
	// Remove the current activity from related
	related = filterOutActivity(related, code)
	// Limit to 5 related
	if len(related) > 5 {
		related = related[:5]
	}

	return &ActivityDetails{
		Code:              code,
		Name:              name,
		CategoryCode:      categoryCode,
		CategoryName:      category.Name,
		RelatedActivities: related,
	}, true
}

// Helper: Calculate relevance score for activities
func calculateRelevanceScore(query, code, name string) float64 {
	nameLower := strings.ToLower(name)
	score := 0.0

	// Exact code match - highest priority
	if code == query {
		score += 100.0
	} else if strings.HasPrefix(code, query) {
		score += 50.0
	} else if strings.Contains(code, query) {
		score += 25.0
	}

	// Exact phrase match in name
	if strings.Contains(nameLower, query) {
		score += 40.0
	}

	// Word matches in name
	queryWords := strings.Fields(query)
	nameWords := strings.Fields(nameLower)
	matchedWords := 0
	for _, qw := range queryWords {
		for _, nw := range nameWords {
			if strings.Contains(nw, qw) || strings.Contains(qw, nw) {
				matchedWords++
				break
			}
		}
	}
	if len(queryWords) > 0 {
		score += float64(matchedWords) / float64(len(queryWords)) * 30.0
	}

	return score
}

// Helper: Calculate relevance for categories
func calculateCategoryRelevance(query, code string, cat codigos.ActivityCategory) float64 {
	score := 0.0
	nameLower := strings.ToLower(cat.Name)

	// Code match
	if code == query {
		score += 100.0
	} else if strings.HasPrefix(code, query) {
		score += 50.0
	}

	// Name match
	if strings.Contains(nameLower, query) {
		score += 40.0
	}

	// Keyword match
	for _, keyword := range cat.Keywords {
		keywordLower := strings.ToLower(keyword)
		if keywordLower == query {
			score += 60.0
			break
		} else if strings.Contains(keywordLower, query) || strings.Contains(query, keywordLower) {
			score += 30.0
		}
	}

	return score
}

// Helper: Extract category code from activity code
// Activities like "46201" belong to category "46"
func getCategoryCodeFromActivity(activityCode string) string {
	if len(activityCode) >= 2 {
		return activityCode[:2]
	}
	return ""
}

// Helper: Filter out a specific activity from a list
func filterOutActivity(activities []codigos.EconomicActivity, codeToRemove string) []codigos.EconomicActivity {
	filtered := make([]codigos.EconomicActivity, 0)
	for _, act := range activities {
		if act.Code != codeToRemove {
			filtered = append(filtered, act)
		}
	}
	return filtered
}

// DTOs for service responses
type SearchResult struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	Type           string  `json:"type"` // "activity" or "category"
	RelevanceScore float64 `json:"relevance_score,omitempty"`
}

type ActivityDetails struct {
	Code              string                     `json:"code"`
	Name              string                     `json:"name"`
	CategoryCode      string                     `json:"category_code"`
	CategoryName      string                     `json:"category_name"`
	RelatedActivities []codigos.EconomicActivity `json:"related_activities"`
}
