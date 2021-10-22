package mod

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport/ssh"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/go-version"
)

type repo struct {
	contents *git.Repository
}

func clone(require *Require, dir string, privateKeyFile, privateKeyPassword string) (*repo, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("error cleaning up tmp directory")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("error creating tmp dir for cloning package")
	}

	o := git.CloneOptions{
		URL: fmt.Sprintf("https://%s", require.cloneRepo),
	}

	if privateKeyFile != "" {
		publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyFile, privateKeyPassword)
		if err != nil {
			return nil, err
		}

		o.Auth = publicKeys
		o.URL = fmt.Sprintf("git@%s", strings.Replace(require.cloneRepo, "/", ":", 1))
	}

	r, err := git.PlainClone(dir, false, &o)
	if err != nil {
		return nil, err
	}

	rr := &repo{
		contents: r,
	}

	if require.version == "" {
		latestTag, err := rr.latestTag(require.versionConstraint)
		if err != nil {
			return nil, err
		}

		require.version = latestTag
	}

	if err := rr.checkout(require.version); err != nil {
		return nil, err
	}

	return rr, nil
}

func (r *repo) checkout(version string) error {
	h, err := r.contents.ResolveRevision(plumbing.Revision(version))
	if err != nil {
		return err
	}

	w, err := r.contents.Worktree()
	if err != nil {
		return err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: *h,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) listTagVersions(versionConstraint string) ([]string, error) {
	if versionConstraint == "" {
		versionConstraint = ">= 0"
	}

	constraint, err := version.NewConstraint(versionConstraint)
	if err != nil {
		return nil, err
	}

	iter, err := r.contents.Tags()
	if err != nil {
		return nil, err
	}

	var tags []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tagV := ref.Name().Short()

		if !strings.HasPrefix(tagV, "v") {
			// ignore wrong formatted tags
			return nil
		}

		v, err := version.NewVersion(tagV)
		if err != nil {
			// ignore invalid tag
			return nil
		}

		if constraint.Check(v) {
			// Add tag if it matches the version constraint
			tags = append(tags, ref.Name().Short())
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (r *repo) latestTag(versionConstraint string) (string, error) {
	versionsRaw, err := r.listTagVersions(versionConstraint)
	if err != nil {
		return "", err
	}

	versions := make([]*version.Version, len(versionsRaw))
	for i, raw := range versionsRaw {
		v, _ := version.NewVersion(raw)
		versions[i] = v
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("repo doesn't have any tags matching the required version")
	}

	sort.Sort(sort.Reverse(version.Collection(versions)))
	return versions[0].Original(), nil
}
