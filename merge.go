package main

import (
	"fmt"
	"regexp"
	"strings"

	sdk "gitee.com/openeuler/go-gitee/gitee"
	"github.com/opensourceways/community-robot-lib/giteeclient"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	msgPRConflicts        = "PR conflicts to the target branch."
	msgMissingLabels      = "PR does not have these lables: %s"
	msgInvalidLabels      = "PR should remove these labels: %s"
	msgNotEnoughLGTMLabel = "PR needs %d lgtm labels and now gets %d"
	msgFrozenWithOwner    = "PR merge target has been frozen, and can merge only by branch owners: %s"
)

var regCheckPr = regexp.MustCompile(`(?mi)^/check-pr\s*$`)

func (bot *robot) handleCheckPR(e *sdk.NoteEvent, cfg *botConfig) error {
	ne := giteeclient.NewPRNoteEvent(e)

	if !ne.IsPullRequest() ||
		!ne.IsPROpen() ||
		!ne.IsCreatingCommentEvent() ||
		!regCheckPr.MatchString(ne.GetComment()) {
		return nil
	}

	pr := ne.PullRequest
	org, repo := ne.GetOrgRep()

	if r, err := bot.canMerge(pr.Mergeable, ne.GetCommenter(), ne.GetPRInfo(), cfg); err != nil {
		return err
	} else if len(r) > 0 {
		return bot.cli.CreatePRComment(
			org, repo, ne.GetPRNumber(),
			fmt.Sprintf(
				"@%s , this pr is not mergeable and the reasons are below:\n%s",
				ne.GetCommenter(), strings.Join(r, "\n"),
			),
		)
	}

	return bot.mergePR(
		pr.NeedReview || pr.NeedTest,
		org, repo, ne.GetPRNumber(), string(cfg.MergeMethod),
	)
}

func (bot *robot) tryMerge(e *sdk.PullRequestEvent, cfg *botConfig) error {
	if giteeclient.GetPullRequestAction(e) != giteeclient.PRActionUpdatedLabel {
		return nil
	}

	pr := e.PullRequest
	info := giteeclient.GetPRInfoByPREvent(e)

	if r, err := bot.canMerge(pr.Mergeable, e.Author.Name, info, cfg); err != nil || len(r) > 0 {
		return err
	}

	return bot.mergePR(
		pr.NeedReview || pr.NeedTest,
		info.Org, info.Repo, info.Number, string(cfg.MergeMethod),
	)
}

func (bot *robot) mergePR(needReviewOrTest bool, org, repo string, number int32, method string) error {
	if needReviewOrTest {
		v := int32(0)
		p := sdk.PullRequestUpdateParam{
			AssigneesNumber: &v,
			TestersNumber:   &v,
		}
		if _, err := bot.cli.UpdatePullRequest(org, repo, number, p); err != nil {
			return err
		}
	}

	return bot.cli.MergePR(
		org, repo, number,
		sdk.PullRequestMergePutParam{
			MergeMethod: method,
		},
	)
}

func (bot *robot) canMerge(
	mergeable bool,
	owner string,
	pr giteeclient.PRInfo,
	cfg *botConfig,
) ([]string, error) {
	if !mergeable {
		return []string{msgPRConflicts}, nil
	}

	var reasons []string

	needs := sets.NewString(approvedLabel)
	needs.Insert(cfg.LabelsForMerge...)

	if ln := cfg.LgtmCountsRequired; ln == 1 {
		needs.Insert(lgtmLabel)
	} else {
		v := getLGTMLabelsOnPR(pr.Labels)
		if n := uint(len(v)); n < ln {
			reasons = append(reasons, fmt.Sprintf(msgNotEnoughLGTMLabel, ln, n))
		}
	}

	if v := needs.Difference(pr.Labels); v.Len() > 0 {
		reasons = append(reasons, fmt.Sprintf(
			msgMissingLabels, strings.Join(v.UnsortedList(), ", "),
		))
	}

	if len(cfg.MissingLabelsForMerge) > 0 {
		missing := sets.NewString(cfg.MissingLabelsForMerge...)
		if v := missing.Intersection(pr.Labels); v.Len() > 0 {
			reasons = append(reasons, fmt.Sprintf(
				msgInvalidLabels, strings.Join(v.UnsortedList(), ", "),
			))
		}
	}

	freeze, err := bot.getFreezeInfo(pr.Org, pr.BaseRef, cfg.FreezeFile)
	if err != nil {
		return reasons, err
	}

	if freeze.isFrozen(pr.Org, pr.BaseRef, owner) {
		reasons = append(reasons, fmt.Sprintf(
			msgFrozenWithOwner, strings.Join(freeze.Owner, ", "),
		))
	}

	return reasons, nil
}
