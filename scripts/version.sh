#!/bin/sh
# Computes the app version per SemVer 2.0.0, e.g. 0.7.0 or 0.7.0-dev.3+gA1b2c3d.
#
# The X.Y.Z itself comes from the nearest reachable git tag (vX.Y.Z) and is a
# human decision made by cutting a new tag — semver bumps aren't derivable
# automatically the way a calendar version is, since only a person can judge
# whether a change is breaking/feature/fix.
#
# What *does* update on every commit automatically: builds made between tags
# get a "-dev.<commits-since-tag>+g<sha>" suffix, so the build identifier
# always changes even though X.Y.Z stays pinned to the last release tag.
# A dirty worktree (uncommitted changes) adds ".dirty".
set -e
cd "$(dirname "$0")/.."

if ! git rev-parse --git-dir >/dev/null 2>&1; then
  echo "0.1.0-dev"
  exit 0
fi

if ! git describe --tags >/dev/null 2>&1; then
  echo "0.1.0-dev"
  exit 0
fi

desc=$(git describe --tags --long)
sha=${desc##*-g}
rest=${desc%-*}
distance=${rest##*-}
tag=${rest%-*}
ver=${tag#v}

dirty=""
if ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null; then
  dirty=".dirty"
fi

if [ "$distance" = "0" ]; then
  if [ -z "$dirty" ]; then
    echo "$ver"
  else
    echo "${ver}+dirty"
  fi
  exit 0
fi

echo "${ver}-dev.${distance}+g${sha}${dirty}"
