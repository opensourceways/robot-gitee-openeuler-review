package main

import (
	"fmt"
	"regexp"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/opensourceways/community-robot-lib/giteeclient"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// the gitee platform limits the maximum length of label to 20.
	labelLenLimit = 20
	lgtmLabel     = "lgtm"

	commentAddLGTMBySelf            = "***lgtm*** can not be added in your self-own pull request. :astonished:"
	commentClearLabel               = `New code changes of pr are detected and remove these labels ***%s***. :flushed: `
	commentNoPermissionForLgtmLabel = `Thanks for your review, ***%s***, your opinion is very important to us.:wave:
The maintainers will consider your advice carefully.`
	commentNoPermissionForLabel = `
***@%s*** has no permission to %s ***%s*** label in this pull request. :astonished:
Please contact to the collaborators in this repository.`
	commentAddLabel = `***%s*** was added to this pull request by: ***%s***. :wave: 
**NOTE:** If this pull request is not merged while all conditions are met, comment "/check-pr" to try again. :smile: `
	commentRemovedLabel = `***%s*** was removed in this pull request by: ***%s***. :flushed: `
)

var (
	regAddLgtm    = regexp.MustCompile(`(?mi)^/lgtm\s*$`)
	regRemoveLgtm = regexp.MustCompile(`(?mi)^/lgtm cancel\s*$`)
)

func (bot *robot) handleLGTM(e *sdk.NoteEvent, cfg *botConfig, log *logrus.Entry) error {
	ne := giteeclient.NewPRNoteEvent(e)

	if !ne.IsPullRequest() || !ne.IsPROpen() || !ne.IsCreatingCommentEvent() {
		return nil
	}

	if regAddLgtm.MatchString(ne.GetComment()) {
		return bot.addLGTM(cfg, ne, log)
	}

	if regRemoveLgtm.MatchString(ne.GetComment()) {
		return bot.removeLGTM(cfg, ne, log)
	}

	return nil
}

func (bot *robot) addLGTM(cfg *botConfig, e giteeclient.PRNoteEvent, log *logrus.Entry) error {
	pr := e.GetPRInfo()
	org, repo, number := pr.Org, pr.Repo, pr.Number

	commenter := e.GetCommenter()
	if pr.Author == commenter {
		return bot.cli.CreatePRComment(org, repo, number, commentAddLGTMBySelf)
	}

	v, err := bot.hasPermission(commenter, pr, cfg, log)
	if err != nil {
		return err
	}
	if !v {
		return bot.cli.CreatePRComment(
			org, repo, number,
			fmt.Sprintf(commentNoPermissionForLgtmLabel, commenter),
		)
	}

	label := genLGTMLabel(commenter, cfg.LgtmCountsRequired)
	if label != lgtmLabel {
		if err := bot.createLabelIfNeed(org, repo, label); err != nil {
			log.WithError(err).Errorf("create repo label: %s", label)
		}
	}

	if err := bot.cli.AddPRLabel(org, repo, number, label); err != nil {
		return err
	}

	err = bot.cli.CreatePRComment(
		org, repo, number, fmt.Sprintf(commentAddLabel, label, commenter),
	)
	if err != nil {
		log.Error(err)
	}

	return bot.tryMerge(e, cfg, false, log)
}

func (bot *robot) removeLGTM(cfg *botConfig, e giteeclient.PRNoteEvent, log *logrus.Entry) error {
	pr := e.GetPRInfo()
	org, repo, number := pr.Org, pr.Repo, pr.Number

	if commenter := e.GetCommenter(); pr.Author != commenter {
		v, err := bot.hasPermission(commenter, pr, cfg, log)
		if err != nil {
			return err
		}
		if !v {
			return bot.cli.CreatePRComment(org, repo, number, fmt.Sprintf(
				commentNoPermissionForLabel, commenter, "remove", lgtmLabel,
			))
		}

		l := genLGTMLabel(commenter, cfg.LgtmCountsRequired)
		if err = bot.cli.RemovePRLabel(org, repo, number, l); err != nil {
			return err
		}

		return bot.cli.CreatePRComment(
			org, repo, number, fmt.Sprintf(commentRemovedLabel, l, commenter),
		)
	}

	// the author of pr can remove all of lgtm[-login name] kind labels
	if v := getLGTMLabelsOnPR(pr.Labels); len(v) > 0 {
		return bot.cli.RemovePRLabels(org, repo, number, v)
	}

	return nil
}

func (bot *robot) createLabelIfNeed(org, repo, label string) error {
	repoLabels, err := bot.cli.GetRepoLabels(org, repo)
	if err != nil {
		return err
	}

	for _, v := range repoLabels {
		if v.Name == label {
			return nil
		}
	}

	return bot.cli.CreateRepoLabel(org, repo, label, "")
}

func genLGTMLabel(commenter string, lgtmCount uint) string {
	if lgtmCount <= 1 {
		return lgtmLabel
	}

	l := fmt.Sprintf("%s-%s", lgtmLabel, strings.ToLower(commenter))
	if len(l) > labelLenLimit {
		return l[:labelLenLimit]
	}

	return l
}

func getLGTMLabelsOnPR(labels sets.String) []string {
	var r []string

	for l := range labels {
		if strings.HasPrefix(l, lgtmLabel) {
			r = append(r, l)
		}
	}

	return r
}
