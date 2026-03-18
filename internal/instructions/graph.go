package instructions

type Artifact struct {
	ID       string
	File     string
	Requires []string
	Unlocks  []string
}

var ArtifactGraph = []Artifact{
	{ID: "proposal", File: "proposal.md", Requires: []string{}, Unlocks: []string{"specs", "design"}},
	{ID: "specs", File: "specs/", Requires: []string{"proposal"}, Unlocks: []string{"tasks"}},
	{ID: "design", File: "design.md", Requires: []string{"proposal"}, Unlocks: []string{"tasks"}},
	{ID: "tasks", File: "tasks.md", Requires: []string{"specs", "design"}, Unlocks: []string{"verify"}},
	{ID: "verify", File: "verify.md", Requires: []string{"tasks"}, Unlocks: []string{}},
}

func DependenciesOf(artifact string) []string {
	for _, a := range ArtifactGraph {
		if a.ID == artifact {
			return a.Requires
		}
	}
	return nil
}

func UnlocksOf(artifact string) []string {
	for _, a := range ArtifactGraph {
		if a.ID == artifact {
			return a.Unlocks
		}
	}
	return nil
}
