package main

import (
	"fmt"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/opensourceways/community-robot-lib/giteeclient"
)

const (
	retestCommand     = "/retest"
	msgNotSetReviewer = "**@%s** Thank you for submitting a PullRequest. It is detected that you have not set a reviewer, please set a one."
)

func (bot *robot) doRetest(e *sdk.PullRequestEvent) error {
	if giteeclient.GetPullRequestAction(e) != giteeclient.PRActionChangedSourceBranch {
		return nil
	}

	pr := giteeclient.GetPRInfoByPREvent(e)

	return bot.cli.CreatePRComment(pr.Org, pr.Repo, pr.Number, retestCommand)
}

func (bot *robot) checkReviewer(e *sdk.PullRequestEvent, cfg *botConfig) error {
	if cfg.UnableCheckingReviewerForPR || giteeclient.GetPullRequestAction(e) != giteeclient.PRActionOpened {
		return nil
	}

	if e.GetPullRequest() != nil && len(e.GetPullRequest().Assignees) > 0 {
		return nil
	}

	pr := giteeclient.GetPRInfoByPREvent(e)

	return bot.cli.CreatePRComment(pr.Org, pr.Repo, pr.Number, fmt.Sprintf(msgNotSetReviewer, pr.Author))
}

func (bot *robot) clearLabel(e *sdk.PullRequestEvent) error {
	if giteeclient.GetPullRequestAction(e) != giteeclient.PRActionChangedSourceBranch {
		return nil
	}

	pr := giteeclient.GetPRInfoByPREvent(e)
	v := getLGTMLabelsOnPR(pr.Labels)

	if pr.Labels.Has(approvedLabel) {
		v = append(v, approvedLabel)
	}

	if len(v) > 0 {
		if err := bot.cli.RemovePRLabels(pr.Org, pr.Repo, pr.Number, v); err != nil {
			return err
		}

		return bot.cli.CreatePRComment(
			pr.Org, pr.Repo, pr.Number,
			fmt.Sprintf(commentClearLabel, strings.Join(v, ", ")),
		)
	}

	return nil
}
