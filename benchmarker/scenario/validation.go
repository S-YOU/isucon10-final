package scenario

import (
	"context"
	"net/url"
	"sort"
	"time"

	"github.com/isucon/isucon10-final/benchmarker/proto/xsuportal/resources"

	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/random/useragent"

	"github.com/isucon/isucandar/worker"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucon10-final/benchmarker/model"
)

type validateLeaderbord struct {
}

func (s *Scenario) Validation(ctx context.Context, step *isucandar.BenchmarkStep) error {
	if s.NoLoad {
		return nil
	}

	<-time.After(s.Contest.ContestEndsAt.Add(5 * time.Second).Sub(time.Now()))
	ContestantLogger.Printf("===> VALIDATION")

	s.validationLeaderboard(ctx, step)
	return nil
}

func (s *Scenario) validationLeaderboard(ctx context.Context, step *isucandar.BenchmarkStep) {
	at := s.Contest.ContestEndsAt.Add(10 * time.Second)
	teams := s.Contest.Teams
	generalTeams := []*model.Team{}
	studentTeams := []*model.Team{}

	sort.Slice(teams, func(i int, j int) bool {
		ils, ilt := teams[i].LatestScore(at)
		jls, jlt := teams[j].LatestScore(at)
		if ils == jls {
			return jlt.After(ilt)
		}
		return ils > jls
	})

	// ContestantLogger.Println("=> Expected ranking")
	tidMap := make(map[int64]*model.Team)
	for _, team := range teams {
		// ContestantLogger.Printf("%d: %s\n", i+1, team.TeamName)
		tidMap[team.ID] = team

		if team.IsStudent {
			studentTeams = append(studentTeams, team)
		} else {
			generalTeams = append(generalTeams, team)
		}
	}

	w, err := worker.NewWorker(func(ctx context.Context, idx int) {
		var leaderboard *resources.Leaderboard

		if idx < len(teams) {
			t := teams[idx]
			_, res, err := GetDashboardAction(ctx, t, t.Leader)
			if err != nil {
				step.AddError(failure.NewError(ErrCritical, err))
				return
			}
			leaderboard = res.GetLeaderboard()
		} else {
			admin, err := model.NewAdmin()
			if err != nil {
				step.AddError(failure.NewError(ErrCritical, err))
				return
			}
			admin.Agent.BaseURL, _ = url.Parse(s.BaseURL)
			admin.Agent.Name = useragent.Chrome()

			_, err = LoginAction(ctx, admin)
			if err != nil {
				step.AddError(failure.NewError(ErrCritical, err))
				return
			}

			_, res, err := AudienceGetDashboardAction(ctx, admin.Agent)
			if err != nil {
				step.AddError(failure.NewError(ErrCritical, err))
				return
			}
			leaderboard = res.GetLeaderboard()
		}

		s.validationLeaderboardWithTeams(step, teams, leaderboard.GetTeams())
		s.validationLeaderboardWithTeams(step, generalTeams, leaderboard.GetGeneralTeams())
		s.validationLeaderboardWithTeams(step, studentTeams, leaderboard.GetStudentTeams())

	}, worker.WithLoopCount(int32(len(teams)+1)))
	if err != nil {
		step.AddError(failure.NewError(ErrCritical, err))
		return
	}

	w.Process(ctx)
}

func (s *Scenario) validationLeaderboardWithTeams(step *isucandar.BenchmarkStep, teams []*model.Team, aTeams []*resources.Leaderboard_LeaderboardItem) {
	at := s.Contest.ContestEndsAt.Add(10 * time.Second)

	errTeamId := failure.NewError(ErrCritical, errorInvalidResponse("ランキング上の最終 ID 検証に失敗しました"))
	errStudent := failure.NewError(ErrCritical, errorInvalidResponse("ランキング上の最終学生チーム検証に失敗しました"))
	errBestScore := failure.NewError(ErrCritical, errorInvalidResponse("ランキング上の最終ベストスコア検証に失敗しました"))
	errLatestScore := failure.NewError(ErrCritical, errorInvalidResponse("ランキング上の最終最新スコア検証に失敗しました"))
	errScoreCount := failure.NewError(ErrCritical, errorInvalidResponse("最終検証でのスコア数不一致"))
	errScore := failure.NewError(ErrCritical, errorInvalidResponse("最終検証でのスコアの情報不一致"))

	for idx, ateamResult := range aTeams {
		eteam := teams[idx]

		ateam := ateamResult.GetTeam()
		if !AssertEqual("Validate leaderboard team id", eteam.ID, ateam.GetId()) {
			step.AddError(errTeamId)
			return
		}

		if !AssertEqual("Validate leaderboard student", eteam.IsStudent, ateam.GetStudent().GetStatus()) {
			step.AddError(errStudent)
			return
		}

		eb, _ := eteam.BestScore(at)
		if !AssertEqual("Validate leaderboard best", eb, ateamResult.GetBestScore().GetScore()) {
			step.AddError(errBestScore)
			return
		}

		el, _ := eteam.LatestScore(at)
		if !AssertEqual("Validate leaderboard latest", el, ateamResult.GetLatestScore().GetScore()) {
			step.AddError(errLatestScore)
			return
		}

		escores := eteam.BenchmarkResults(at)
		if !AssertEqual("Validate leaderboard score count", int64(len(escores)), ateamResult.GetFinishCount()) {
			step.AddError(errScoreCount)
			return
		}

		for idx, ascore := range ateamResult.GetScores() {
			escore := escores[idx]

			if !AssertEqual("validate score", escore.Score, ascore.GetScore()) || !AssertEqual("validate score", escore.MarkedAt(), ascore.GetMarkedAt().AsTime()) {
				step.AddError(errScore)
				return
			}
		}
	}
}

func (s *Scenario) validationClarification(ctx context.Context, step *isucandar.BenchmarkStep) {
	errNotFound := failure.NewError(ErrCritical, errorInvalidResponse("最終検証にて存在しないはずの Clarification が見つかりました"))
	errNotMatch := failure.NewError(ErrCritical, errorInvalidResponse("最終検証にて Clarification の不一致が検出されました"))

	admin, err := model.NewAdmin()
	if err != nil {
		step.AddError(failure.NewError(ErrCritical, err))
		return
	}
	admin.Agent.BaseURL, _ = url.Parse(s.BaseURL)
	admin.Agent.Name = useragent.Chrome()

	_, err = LoginAction(ctx, admin)
	if err != nil {
		step.AddError(failure.NewError(ErrCritical, err))
		return
	}

	res, err := AdminGetClarificationsAction(ctx, admin)
	if err != nil {
		step.AddError(failure.NewError(ErrCritical, err))
		return
	}

	eclarID := make(map[int64]*model.Clarification)
	for _, eclar := range s.Contest.Clarifications() {
		eclarID[eclar.ID()] = eclar
	}

	for _, aclar := range res.GetClarifications() {
		eclar, ok := eclarID[aclar.GetId()]
		if !ok {
			step.AddError(errNotFound)
			return
		}

		if !AssertEqual("validate clar team id", eclar.TeamID, aclar.GetTeamId()) ||
			!AssertEqual("validate clar question", eclar.Question, aclar.GetQuestion()) ||
			!AssertEqual("validate clar answer", eclar.Answer, aclar.GetAnswer) ||
			!AssertEqual("validate clar answered", eclar.IsAnswered(), aclar.GetAnswered()) ||
			!AssertEqual("validate clar disclose", eclar.Disclose, aclar.GetDisclosed()) {
			step.AddError(errNotMatch)
			return
		}
	}
}
