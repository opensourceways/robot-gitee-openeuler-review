package main

import (
	"fmt"
	"regexp"

	sdk "github.com/opensourceways/go-gitee/gitee"
	"github.com/sirupsen/logrus"
)

const approvedLabel = "approved"

var (
	regAddApprove    = regexp.MustCompile(`(?mi)^/approve\s*$`)
	regRemoveApprove = regexp.MustCompile(`(?mi)^/approve cancel\s*$`)
)

func (bot *robot) handleApprove(e *sdk.NoteEvent, cfg *botConfig, log *logrus.Entry) error {
	if !e.IsPullRequest() || !e.IsPROpen() || !e.IsCreatingCommentEvent() {
		return nil
	}

	comment := e.GetComment().GetBody()
	if regAddApprove.MatchString(comment) {
		return bot.AddApprove(cfg, e, log)
	}

	if regRemoveApprove.MatchString(comment) {
		return bot.removeApprove(cfg, e, log)
	}

	return nil
}

func (bot *robot) AddApprove(cfg *botConfig, e *sdk.NoteEvent, log *logrus.Entry) error {
	org, repo := e.GetOrgRepo()
	commenter := e.GetCommenter()

	v, err := bot.hasPermission(org, repo, commenter, false, e.GetPullRequest(), cfg, log)
	if err != nil {
		return err
	}

	if !v {
		return bot.cli.CreatePRComment(org, repo, e.GetPRNumber(), fmt.Sprintf(
			commentNoPermissionForLabel, commenter, "add", approvedLabel,
		))
	}

	if err := bot.cli.AddPRLabel(org, repo, e.GetPRNumber(), approvedLabel); err != nil {
		return err
	}

	err = bot.cli.CreatePRComment(
		org, repo, e.GetPRNumber(),
		fmt.Sprintf(commentAddLabel, approvedLabel, commenter),
	)
	if err != nil {
		log.Error(err)
	}

	return bot.tryMerge(e, cfg, false, log)
}

func (bot *robot) removeApprove(cfg *botConfig, e *sdk.NoteEvent, log *logrus.Entry) error {
	org, repo := e.GetOrgRepo()
	commenter := e.GetCommenter()

	v, err := bot.hasPermission(org, repo, commenter, false, e.GetPullRequest(), cfg, log)
	if err != nil {
		return err
	}

	if !v {
		return bot.cli.CreatePRComment(org, repo, e.GetPRNumber(), fmt.Sprintf(
			commentNoPermissionForLabel, commenter, "remove", approvedLabel,
		))
	}

	err = bot.cli.RemovePRLabel(org, repo, e.GetPRNumber(), approvedLabel)
	if err != nil {
		return err
	}

	return bot.cli.CreatePRComment(
		org, repo, e.GetPRNumber(),
		fmt.Sprintf(commentRemovedLabel, approvedLabel, commenter),
	)
}
