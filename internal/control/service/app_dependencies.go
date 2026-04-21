package service

import (
	runtimepkg "mockinterview/internal/control/runtime"
	interviewexec "mockinterview/internal/executors/interview"
)

type AppDependencies struct {
	Engine     runtimepkg.Engine
	Tools      runtimepkg.ToolGateway
	Scorecards ScorecardGenerator
}

func (d AppDependencies) withDefaults() AppDependencies {
	if d.Scorecards == nil {
		d.Scorecards = interviewexec.GenerateScorecard
	}
	return d
}
