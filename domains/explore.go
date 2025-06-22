package domains

import (
	"sort"
)

// ExploreResult represents the results of a domain exploration across multiple TLDs
type ExploreResult struct {
	BaseDomain string               `json:"base_domain"`
	Results    []DomainSearchResult `json:"results"`
	Available  []DomainSearchResult `json:"available"`
	Taken      []DomainSearchResult `json:"taken"`
	Errors     []DomainSearchResult `json:"errors"`
}

// OrganizeExploreResults categorizes domain search results into available, taken, and error categories
func OrganizeExploreResults(baseDomain string, results []DomainSearchResult) ExploreResult {
	exploreResult := ExploreResult{
		BaseDomain: baseDomain,
		Results:    results,
		Available:  make([]DomainSearchResult, 0),
		Taken:      make([]DomainSearchResult, 0),
		Errors:     make([]DomainSearchResult, 0),
	}
	
	for _, result := range results {
		if result.Error != "" {
			exploreResult.Errors = append(exploreResult.Errors, result)
		} else if result.Available {
			exploreResult.Available = append(exploreResult.Available, result)
		} else {
			exploreResult.Taken = append(exploreResult.Taken, result)
		}
	}
	
	// Sort each category by domain name
	sort.Slice(exploreResult.Available, func(i, j int) bool {
		return exploreResult.Available[i].Domain < exploreResult.Available[j].Domain
	})
	sort.Slice(exploreResult.Taken, func(i, j int) bool {
		return exploreResult.Taken[i].Domain < exploreResult.Taken[j].Domain
	})
	sort.Slice(exploreResult.Errors, func(i, j int) bool {
		return exploreResult.Errors[i].Domain < exploreResult.Errors[j].Domain
	})
	
	return exploreResult
}