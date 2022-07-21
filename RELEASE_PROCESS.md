# Release Process

This document outlines the process for releasing a new version of the Nibiru binary.

## Step 1) Create a new release branch off master

```sh
git checkout master
git branch releases/v0.x.y
git switch releases/v0.x.y
```

## Step 2) Update the changelog

- Move the changelog notes from `Unreleased` to a new section titled `v0.x.y`
- Add the date to `v0.x.y`

## Step 3) Create a tag

```sh
git tag -a -s -m "Create new release v0.x.y" v0.x.y
git push origin v0.x.y
```

## Step 4) Go to the [GitHub Releases](https://github.com/NibiruChain/nibiru/releases) page

- Create a new release from the tag v0.x.y you just pushed
- Make sure you save the release as a draft release
- Make sure you check the `This is a pre-release` checkbox

## Step 5) Build the binaries locally

```sh
cd nibiru/

ignite chain build --release -t linux:amd64 -t linux:arm64 -t darwin:amd64 -t darwin:arm64
```

## Step 6) Upload the files in the release/ directory to GitHub

Upload all the files under `releases/` to the [GitHub Release](https://github.com/NibiruChain/nibiru/releases) you just made

## Step 7) Merge your release branch into master

After merging the release branch `releases/v0.x.y` into master, uncheck the `This is a pre-release` checkbox for the release.
