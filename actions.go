package main

import (
	"fmt"
	"strings"

	sdk "github.com/opensourceways/go-gitee/gitee"
)

const (
	retestCommand     = "/retest"
	msgNotSetReviewer = "**@%s** Thank you for submitting a PullRequest. It is detected that you have not set a reviewer, please set a one."
)

func (bot *robot) doRetest(e *sdk.PullRequestEvent) error {
	if sdk.GetPullRequestAction(e) != sdk.PRActionChangedSourceBranch {
		return nil
	}

	org, repo := e.GetOrgRepo()

	return bot.cli.CreatePRComment(org, repo, e.GetPRNumber(), retestCommand)
}

func (bot *robot) checkReviewer(e *sdk.PullRequestEvent, cfg *botConfig) error {
	if cfg.UnableCheckingReviewerForPR || sdk.GetPullRequestAction(e) != sdk.ActionOpen {
		return nil
	}

	if e.GetPullRequest() != nil && len(e.GetPullRequest().Assignees) > 0 {
		return nil
	}

	org, repo := e.GetOrgRepo()

	return bot.cli.CreatePRComment(
		org, repo, e.GetPRNumber(),
		fmt.Sprintf(msgNotSetReviewer, e.GetPRAuthor()),
	)
}

func (bot *robot) clearLabel(e *sdk.PullRequestEvent) error {
	if sdk.GetPullRequestAction(e) != sdk.PRActionChangedSourceBranch {
		return nil
	}

	labels := e.GetPRLabelSet()
	v := getLGTMLabelsOnPR(labels)

	if labels.Has(approvedLabel) {
		v = append(v, approvedLabel)
	}

	if len(v) > 0 {
		org, repo := e.GetOrgRepo()
		number := e.GetPRNumber()

		if err := bot.cli.RemovePRLabels(org, repo, number, v); err != nil {
			return err
		}

		return bot.cli.CreatePRComment(
			org, repo, number,
			fmt.Sprintf(commentClearLabel, strings.Join(v, ", ")),
		)
	}

	return nil
}
