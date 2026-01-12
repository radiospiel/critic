package critic_integration

import (
	"fmt"
	"os"
	"testing"

	"git.15b.it/eno/critic/simple-go/assert"
	"git.15b.it/eno/critic/internal/git"
	"git.15b.it/eno/critic/simple-go/must"
)

func TestLineDisplacement(t *testing.T) {
	SetupFixtureGitRepo(t)

	expectations := map[string]string{
		"v1:v2": `LineDisplacement{
  {add lines=1 old=0 new=4}
  {del lines=4 old=4 new=0}
}`,
		"v1:v3": `LineDisplacement{
  {add lines=4 old=0 new=1}
  {del lines=4 old=4 new=0}
  {add lines=1 old=0 new=8}
}`,
		"v1:v4": `LineDisplacement{
  {add lines=8 old=0 new=1}
  {del lines=4 old=4 new=0}
  {add lines=1 old=0 new=12}
}`,
		"v1:v5": `LineDisplacement{
  {add lines=8 old=0 new=1}
  {del lines=2 old=6 new=0}
  {del lines=4 old=13 new=0}
}`,
		"v1:v6": `LineDisplacement{
  {add lines=8 old=0 new=1}
}`,
		"v1:v7": `LineDisplacement{
  {add lines=3 old=0 new=1}
  {move lines=4 old=10 new=25}
  {add lines=5 old=0 new=20}
}`,
		"v1:v8": `LineDisplacement{
  {add lines=8 old=0 new=1}
}`,
		"v2:v3": `LineDisplacement{
  {add lines=4 old=0 new=1}
}`,
		"v2:v4": `LineDisplacement{
  {add lines=8 old=0 new=1}
}`,
		"v2:v5": `LineDisplacement{
  {add lines=8 old=0 new=1}
  {del lines=1 old=4 new=0}
  {del lines=4 old=10 new=0}
  {add lines=2 old=0 new=12}
}`,
		"v2:v6": `LineDisplacement{
  {add lines=8 old=0 new=1}
  {del lines=1 old=4 new=0}
  {add lines=4 old=0 new=12}
}`,
		"v2:v7": `LineDisplacement{
  {add lines=3 old=0 new=1}
  {del lines=1 old=4 new=0}
  {add lines=4 old=0 new=7}
  {move lines=4 old=7 new=25}
  {add lines=5 old=0 new=20}
}`,
		"v2:v8": `LineDisplacement{
  {add lines=8 old=0 new=1}
  {del lines=1 old=4 new=0}
  {add lines=4 old=0 new=12}
}`,
		"v3:v4": `LineDisplacement{
  {add lines=5 old=0 new=1}
  {del lines=2 old=2 new=0}
  {add lines=1 old=0 new=7}
}`,
		"v3:v5": `LineDisplacement{
  {add lines=5 old=0 new=1}
  {del lines=2 old=2 new=0}
  {add lines=1 old=0 new=7}
  {del lines=1 old=8 new=0}
  {add lines=2 old=0 new=12}
  {del lines=4 old=14 new=0}
}`,
		"v3:v6": `LineDisplacement{
  {add lines=5 old=0 new=1}
  {del lines=2 old=2 new=0}
  {add lines=1 old=0 new=7}
  {del lines=1 old=8 new=0}
  {add lines=4 old=0 new=12}
}`,
		"v3:v7": `LineDisplacement{
  {add lines=3 old=0 new=1}
  {del lines=4 old=1 new=0}
  {add lines=4 old=0 new=7}
  {del lines=1 old=8 new=0}
  {move lines=4 old=11 new=25}
  {add lines=5 old=0 new=20}
}`,
		"v3:v8": `LineDisplacement{
  {add lines=5 old=0 new=1}
  {del lines=2 old=2 new=0}
  {add lines=1 old=0 new=7}
  {del lines=1 old=8 new=0}
  {add lines=4 old=0 new=12}
}`,
		"v4:v5": `LineDisplacement{
  {add lines=2 old=0 new=12}
  {del lines=1 old=12 new=0}
  {del lines=4 old=18 new=0}
}`,
		"v4:v6": `LineDisplacement{
  {add lines=4 old=0 new=12}
  {del lines=1 old=12 new=0}
}`,
		"v4:v7": `LineDisplacement{
  {move lines=5 old=4 new=20}
  {add lines=4 old=0 new=7}
  {del lines=1 old=12 new=0}
  {move lines=4 old=15 new=25}
}`,
		"v4:v8": `LineDisplacement{
  {add lines=4 old=0 new=12}
  {del lines=1 old=12 new=0}
}`,
		"v5:v6": `LineDisplacement{
  {add lines=2 old=0 new=14}
  {add lines=4 old=0 new=21}
}`,
		"v5:v7": `LineDisplacement{
  {move lines=5 old=4 new=20}
  {add lines=2 old=0 new=9}
  {add lines=3 old=0 new=13}
  {del lines=3 old=16 new=0}
  {add lines=4 old=0 new=25}
}`,
		"v5:v8": `LineDisplacement{
  {add lines=2 old=0 new=14}
  {add lines=4 old=0 new=21}
}`,
		"v6:v7": `LineDisplacement{
  {move lines=5 old=4 new=20}
  {move lines=4 old=18 new=25}
}`,
		"v6:v8": `LineDisplacement{
}`,
		"v7:v8": `LineDisplacement{
  {move lines=5 old=20 new=4}
  {move lines=4 old=25 new=18}
}`,
	}

	versions := []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8"}
	for i, v1 := range versions {
		for _, v2 := range versions[i+1:] {
			key := fmt.Sprintf("%s:%s", v1, v2)
			ld, err := git.BuildLineDisplacement("data.txt", v1, v2)
			assert.NoError(t, err, "%s: git failed", key)

			expected := expectations[key]
			assert.Equals(t, ld.String(), expected, "%s mismatch", key)
		}
	}
}

// SetupFixtureGitRepo creates a temporary git repository for testing and changes into it.
// IMPORTANT: Tests using this function cannot run in parallel due to os.Chdir().
// Use -p 1 flag when running tests: go test -p 1 -v
func SetupFixtureGitRepo(t *testing.T) {
	t.Helper()

	originalDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(originalDir) })

	must.Chdir("fixtures/repo")
}
