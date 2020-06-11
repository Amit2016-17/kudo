package install

import (
	"fmt"

	"github.com/thoas/go-funk"
	"github.com/yourbasic/graph"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
)

// dependencyGraph is modeled after 'graph.Mutable' but allows to add vertices.
type dependencyGraph struct {
	edges []map[int]struct{}
}

// AddVertex adds a new vertex to the dependency graph.
func (g *dependencyGraph) AddVertex() {
	g.edges = append(g.edges, map[int]struct{}{})
}

// AddEdge adds an edge from vertex v to w to the dependency graph.
func (g *dependencyGraph) AddEdge(v, w int) {
	g.edges[v][w] = struct{}{}
}

// Order returns the number of vertices of the dependency graph.
func (g *dependencyGraph) Order() int {
	return len(g.edges)
}

func (g *dependencyGraph) Visit(v int, do func(w int, c int64) bool) bool {
	for w := range g.edges[v] {
		if do(w, 1) {
			return true
		}
	}

	return false
}

// gatherDependencies resolved all dependencies of a package.
// Dependencies are resolved recursively.
// Cyclic dependencies are detected and result in an error.
func gatherDependencies(root packages.Resources, resolver pkgresolver.Resolver) ([]packages.Resources, error) {
	pkgs := []packages.Resources{
		root,
	}

	// Each vertex in 'g' matches an index in 'pkgs'.
	g := dependencyGraph{
		edges: []map[int]struct{}{{}},
	}

	if err := dependencyWalk(&pkgs, &g, root, 0, resolver); err != nil {
		return nil, err
	}

	// Remove 'root' from the list of dependencies.
	pkgs = funk.Drop(pkgs, 1).([]packages.Resources) //nolint:errcheck

	return pkgs, nil
}

func dependencyWalk(
	pkgs *[]packages.Resources,
	g *dependencyGraph,
	parent packages.Resources,
	parentIndex int,
	resolver pkgresolver.Resolver) error {
	//nolint:errcheck
	childrenTasks := funk.Filter(parent.OperatorVersion.Spec.Tasks, func(task v1beta1.Task) bool {
		return task.Kind == engtask.KudoOperatorTaskKind
	}).([]v1beta1.Task)

	for _, childTask := range childrenTasks {
		childPkg, err := resolver.Resolve(
			childTask.Spec.KudoOperatorTaskSpec.Package,
			childTask.Spec.KudoOperatorTaskSpec.AppVersion,
			childTask.Spec.KudoOperatorTaskSpec.OperatorVersion)
		if err != nil {
			return fmt.Errorf(
				"failed to resolve package %s, dependency of package %s: %v", fullyQualifiedName(childTask.Spec.KudoOperatorTaskSpec), parent.OperatorVersion.FullyQualifiedName(), err)
		}

		newPackage := false
		childIndex := indexOf(pkgs, childPkg.Resources)
		if childIndex == -1 {
			clog.Printf("Adding new dependency %s", childPkg.Resources.OperatorVersion.FullyQualifiedName())
			newPackage = true

			*pkgs = append(*pkgs, *childPkg.Resources)
			childIndex = len(*pkgs) - 1

			// The number of vertices in 'g' has to match the number of packages we're tracking.
			g.AddVertex()
		}

		// This is a directed graph. The edge represents a dependency of the parent package on the current package.
		g.AddEdge(parentIndex, childIndex)

		if !graph.Acyclic(g) {
			return fmt.Errorf(
				"cyclic package dependency found when adding package %s -> %s", parent.OperatorVersion.FullyQualifiedName(), childPkg.Resources.OperatorVersion.FullyQualifiedName())
		}

		// We only need to walk the dependencies if the package is new
		if newPackage {
			if err := dependencyWalk(pkgs, g, *childPkg.Resources, childIndex, resolver); err != nil {
				return err
			}
		}
	}

	return nil
}

// indexOf method searches for the pkg in pkgs that has the same OperatorVersion/AppVersion (using
// EqualOperatorVersion method) and returns its index or -1 if not found.
func indexOf(pkgs *[]packages.Resources, pkg *packages.Resources) int {
	for i, p := range *pkgs {
		if p.OperatorVersion.EqualOperatorVersion(pkg.OperatorVersion) {
			return i
		}
	}
	return -1
}

func fullyQualifiedName(kt v1beta1.KudoOperatorTaskSpec) string {
	return fmt.Sprintf("%s-%s", v1beta1.OperatorVersionName(kt.Package, kt.OperatorVersion), kt.AppVersion)
}