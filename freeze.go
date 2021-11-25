package main

import (
	"encoding/base64"

	"sigs.k8s.io/yaml"
)

type freezeContent struct {
	Release []freezeItem `json:"release"`
}

func (fc freezeContent) getFreezeItem(org, branch string) *freezeItem {
	for _, v := range fc.Release {
		if v.Branch == branch && v.hasOrg(org) {
			return &v
		}
	}

	return nil
}

type freezeItem struct {
	Branch    string   `json:"branch"`
	Community []string `json:"community"`
	Frozen    bool     `json:"frozen"`
	Owner     []string `json:"owner"`
}

func (fi freezeItem) isFrozen(org, branch, owner string) bool {
	if branch != fi.Branch {
		return false
	}

	return fi.hasOrg(org) && fi.hasOwner(owner)
}

func (fi freezeItem) hasOrg(org string) bool {
	for _, v := range fi.Community {
		if v == org {
			return true
		}
	}

	return false
}

func (fi freezeItem) hasOwner(owner string) bool {
	for _, v := range fi.Owner {
		if v == owner {
			return true
		}
	}

	return false
}

func (bot *robot) getFreezeInfo(org, branch string, cfg []freezeFile) (freezeItem, error) {
	var fi freezeItem
	for _, v := range cfg {
		fc, err := bot.getFreezeContent(v)
		if err != nil {
			return fi, err
		}

		if v := fc.getFreezeItem(org, branch); v != nil {
			fi = *v
		}
	}

	return fi, nil
}

func (bot *robot) getFreezeContent(f freezeFile) (freezeContent, error) {
	var fc freezeContent
	c, err := bot.cli.GetPathContent(f.Owner, f.Repo, f.Branch, f.Path)
	if err != nil {
		return fc, err
	}

	b, err := base64.StdEncoding.DecodeString(c.Content)
	if err != nil {
		return fc, err
	}

	err = yaml.Unmarshal(b, &fc)
	return fc, err
}
