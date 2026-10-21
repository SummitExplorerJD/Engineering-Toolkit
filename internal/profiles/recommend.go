package profiles

import (
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/catalog"
	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/workspace"
)

const RecommendationSchema = "engkit.profile-recommendation/v1alpha1"

type Recommendation struct {
	ID         string   `json:"id"`
	Version    string   `json:"version"`
	Compatible bool     `json:"compatible"`
	Reasons    []string `json:"reasons"`
}

type RecommendationResult struct {
	SchemaVersion   string           `json:"schema_version"`
	Root            string           `json:"root"`
	CatalogID       string           `json:"catalog_id"`
	CatalogVersion  string           `json:"catalog_version"`
	Evidence        []string         `json:"evidence"`
	Recommendations []Recommendation `json:"recommendations"`
}

func Recommend(root string, inventory catalog.Catalog, catalogFS fs.FS) (RecommendationResult, error) {
	snapshot, err := workspace.Inspect(root)
	if err != nil {
		return RecommendationResult{}, err
	}
	ecosystems := observedEcosystems(snapshot)
	evidence := []string{"workspace exists"}
	if snapshot.GoModule != "" {
		evidence = append(evidence, "go.mod declares module "+snapshot.GoModule)
	}
	items := make([]Recommendation, 0, len(inventory.Profiles))
	for _, entry := range inventory.Profiles {
		profile, err := Load(catalogFS, path.Join(entry.Path, "profile.yaml"))
		if err != nil {
			return RecommendationResult{}, fmt.Errorf("load profile %s@%s: %w", entry.ID, entry.Version, err)
		}
		if profile.ID != entry.ID || profile.Version != entry.Version {
			return RecommendationResult{}, fmt.Errorf("profile %s@%s identity does not match its catalog entry", entry.ID, entry.Version)
		}
		reasons := []string{}
		compatible := contains(profile.Compatibility.Modes, "existing-workspace")
		if compatible {
			reasons = append(reasons, "supports existing-workspace mode")
		} else {
			reasons = append(reasons, "does not support existing-workspace mode")
		}
		if len(profile.Compatibility.Ecosystems) != 0 {
			matched := intersection(profile.Compatibility.Ecosystems, ecosystems)
			if len(matched) == 0 {
				compatible = false
				reasons = append(reasons, "no declared ecosystem matches observed evidence")
			} else {
				reasons = append(reasons, "matches ecosystem "+matched[0])
			}
		}
		if profile.SourceLayout.Policy == "src" && !contains(ecosystems, "go") {
			compatible = false
			reasons = append(reasons, "src layout lacks compatible observed ecosystem evidence")
		}
		items = append(items, Recommendation{ID: entry.ID, Version: entry.Version, Compatible: compatible, Reasons: reasons})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}
		return items[i].Version < items[j].Version
	})
	return RecommendationResult{SchemaVersion: RecommendationSchema, Root: snapshot.Root, CatalogID: inventory.ID, CatalogVersion: inventory.Version, Evidence: evidence, Recommendations: items}, nil
}

func observedEcosystems(snapshot workspace.Inspection) []string {
	result := []string{}
	if snapshot.GoModule != "" {
		result = append(result, "go")
	}
	return result
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func intersection(left, right []string) []string {
	result := []string{}
	for _, value := range left {
		if contains(right, value) {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}
