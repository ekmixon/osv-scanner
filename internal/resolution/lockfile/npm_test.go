package lockfile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"deps.dev/util/resolve"
	"deps.dev/util/resolve/dep"
	"github.com/google/go-cmp/cmp"
	"github.com/google/osv-scanner/internal/resolution/lockfile"
	"github.com/google/osv-scanner/internal/testutility"
	lf "github.com/google/osv-scanner/pkg/lockfile"
)

func npmVK(t *testing.T, name, version string) resolve.VersionKey {
	t.Helper()
	return resolve.VersionKey{
		PackageKey: resolve.PackageKey{
			System: resolve.NPM,
			Name:   name,
		},
		Version:     version,
		VersionType: resolve.Concrete,
	}
}

func TestNpmReadV2(t *testing.T) {
	t.Parallel()

	// This lockfile was generated using a private registry with https://verdaccio.org/
	// Mock packages were published to it and installed with npm.
	df, err := lf.OpenLocalDepFile("./fixtures/npm_v2/package-lock.json")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer df.Close()

	npmIO := lockfile.NpmLockfileIO{}
	got, err := npmIO.Read(df)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if err := got.Canon(); err != nil {
		t.Fatalf("failed canonicalizing got graph: %v", err)
	}

	want := new(resolve.Graph)
	//nolint:errcheck // AddEdge only errors if the nodes do not exist
	{
		root := want.AddNode(npmVK(t, "r", "1.0.0"))
		workspace := want.AddNode(npmVK(t, "w", "1.0.0"))
		a1 := want.AddNode(npmVK(t, "@fake-registry/a", "1.2.3"))
		a2 := want.AddNode(npmVK(t, "@fake-registry/a", "2.3.4"))
		a2A := want.AddNode(npmVK(t, "@fake-registry/a", "2.3.4"))
		b1 := want.AddNode(npmVK(t, "@fake-registry/b", "1.0.1"))
		b2 := want.AddNode(npmVK(t, "@fake-registry/b", "2.0.0"))
		b2A := want.AddNode(npmVK(t, "@fake-registry/b", "2.0.0"))
		c := want.AddNode(npmVK(t, "@fake-registry/c", "1.1.1"))
		d := want.AddNode(npmVK(t, "@fake-registry/d", "2.2.2"))

		want.AddEdge(root, a1, "^1.2.3", dep.NewType())
		want.AddEdge(root, b1, "^1.0.1", dep.NewType())

		aliasType := dep.NewType(dep.Dev)
		aliasType.AddAttr(dep.KnownAs, "a-dev")
		want.AddEdge(root, a2A, "^2.3.4", aliasType)

		want.AddEdge(root, workspace, "*", dep.NewType())
		want.AddEdge(a1, b1, "^1.0.0", dep.NewType(dep.Opt))
		want.AddEdge(a2A, b2A, "^2.0.0", dep.NewType())
		want.AddEdge(workspace, a2, "^2.3.4", dep.NewType(dep.Dev))
		want.AddEdge(a2, b2, "^2.0.0", dep.NewType())
		want.AddEdge(b2, c, "^1.0.0", dep.NewType())
		want.AddEdge(b2A, c, "^1.0.0", dep.NewType())
		want.AddEdge(b2, d, "^2.0.0", dep.NewType())
		want.AddEdge(b2A, d, "^2.0.0", dep.NewType())
		want.AddEdge(c, d, "^2.0.0", dep.NewType(dep.Opt)) // peerDependency becomes optional
	}

	if err := want.Canon(); err != nil {
		t.Fatalf("failed canonicalizing want graph: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("npm lockfile mismatch (-want/+got):\n%s", diff)
	}
}

func TestNpmReadV1(t *testing.T) {
	t.Parallel()

	// This lockfile was generated using a private registry with https://verdaccio.org/
	// Mock packages were published to it and installed with npm.
	df, err := lf.OpenLocalDepFile("./fixtures/npm_v1/package-lock.json")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer df.Close()

	npmIO := lockfile.NpmLockfileIO{}
	got, err := npmIO.Read(df)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if err := got.Canon(); err != nil {
		t.Fatalf("failed canonicalizing got graph: %v", err)
	}

	want := new(resolve.Graph)
	//nolint:errcheck // AddEdge only errors if the nodes do not exist
	{
		root := want.AddNode(npmVK(t, "r", "1.0.0"))
		a1 := want.AddNode(npmVK(t, "@fake-registry/a", "1.2.3"))
		a2 := want.AddNode(npmVK(t, "@fake-registry/a", "2.3.4"))
		b1 := want.AddNode(npmVK(t, "@fake-registry/b", "1.0.1"))
		b2 := want.AddNode(npmVK(t, "@fake-registry/b", "2.0.0"))
		c := want.AddNode(npmVK(t, "@fake-registry/c", "1.1.1"))
		d := want.AddNode(npmVK(t, "@fake-registry/d", "2.2.2"))
		// v1 does not support workspaces

		want.AddEdge(root, a1, "^1.2.3", dep.NewType())
		want.AddEdge(root, b1, "^1.0.1", dep.NewType())

		aliasType := dep.NewType(dep.Dev)
		aliasType.AddAttr(dep.KnownAs, "a-dev")
		want.AddEdge(root, a2, "^2.3.4", aliasType)

		// all indirect dependencies are optional because it's impossible to tell in v1
		optType := dep.NewType(dep.Opt)
		want.AddEdge(a1, b1, "^1.0.0", optType)
		want.AddEdge(a2, b2, "^2.0.0", optType)
		want.AddEdge(b2, c, "^1.0.0", optType)
		want.AddEdge(b2, d, "^2.0.0", optType)
		// peerDependencies are not in v1
	}

	if err := want.Canon(); err != nil {
		t.Fatalf("failed canonicalizing want graph: %v", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("npm lockfile mismatch (-want/+got):\n%s", diff)
	}
}

func TestNpmWrite(t *testing.T) {
	t.Parallel()

	// Set up mock npm registry
	srv := testutility.NewMockHTTPServer(t)
	srv.SetResponseFromFile(t, "/@fake-registry%2fa/1.2.4", "./fixtures/npm_registry/@fake-registry-a-1.2.4.json")
	srv.SetResponseFromFile(t, "/@fake-registry%2fa/2.3.5", "./fixtures/npm_registry/@fake-registry-a-2.3.5.json")

	// Copy package-lock.json to temporary directory
	dir := testutility.CreateTestDir(t)
	b, err := os.ReadFile("./fixtures/npm_v2/package-lock.json")
	if err != nil {
		t.Fatalf("could not read test file: %v", err)
	}
	file := filepath.Join(dir, "package-lock.json")
	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatalf("could not copy test file: %v", err)
	}

	// create an npmrc file in temp directory pointing to mock registry
	npmrcFile, err := os.Create(filepath.Join(dir, ".npmrc"))
	if err != nil {
		t.Fatalf("could not create .npmrc file: %v", err)
	}
	if _, err := npmrcFile.WriteString("registry=" + srv.URL); err != nil {
		t.Fatalf("failed writing npmrc file: %v", err)
	}

	patches := []lockfile.DependencyPatch{
		{
			Pkg: resolve.PackageKey{
				System: resolve.NPM,
				Name:   "@fake-registry/a",
			},
			OrigVersion: "1.2.3",
			NewVersion:  "1.2.4",
		},
		{
			Pkg: resolve.PackageKey{
				System: resolve.NPM,
				Name:   "@fake-registry/a",
			},
			OrigVersion: "2.3.4",
			NewVersion:  "2.3.5",
		},
	}

	df, err := lf.OpenLocalDepFile(file)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer df.Close()

	buf := new(bytes.Buffer)
	npmIO := lockfile.NpmLockfileIO{}
	if err := npmIO.Write(df, buf, patches); err != nil {
		t.Fatalf("unable to update npm package-lock.json: %v", err)
	}
	testutility.NewSnapshot().WithCRLFReplacement().MatchText(t, buf.String())
}
