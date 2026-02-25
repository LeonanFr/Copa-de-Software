package draw

import (
	"context"
	"math/rand"
	"time"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/teamnames"
	"copasoftware/internal/modules/teams"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service struct {
	participantSvc *participants.Service
	teamSvc        *teams.Service
	teamNameSvc    *teamnames.Service
}

func NewService(
	participantSvc *participants.Service,
	teamSvc *teams.Service,
	teamNameSvc *teamnames.Service,
) *Service {
	return &Service{
		participantSvc: participantSvc,
		teamSvc:        teamSvc,
		teamNameSvc:    teamNameSvc,
	}
}

func (s *Service) RunDraw(ctx context.Context, isFinal bool) ([]participants.Participant, error) {
	eligible, err := s.getEligibleParticipants(ctx)
	if err != nil {
		return nil, err
	}
	if len(eligible) == 0 {
		return nil, nil
	}

	bySemester := s.groupBySemester(eligible)

	if !isFinal {
		bySemester = s.tryFormGroupsWithDifferentSemesters(ctx, bySemester, false)
		return s.flattenRemaining(bySemester), nil
	}

	bySemester = s.tryFormGroupsWithDifferentSemesters(ctx, bySemester, true)
	bySemester = s.formAnyGroups(ctx, bySemester)

	return s.flattenRemaining(bySemester), nil
}

func (s *Service) getEligibleParticipants(ctx context.Context) ([]participants.Participant, error) {
	all, err := s.participantSvc.List(ctx)
	if err != nil {
		return nil, err
	}
	var eligible []participants.Participant
	for _, p := range all {
		teamsList, err := s.teamSvc.GetTeamsByParticipant(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		active := false
		for _, t := range teamsList {
			if t.Status == teams.TeamStatusPending || t.Status == teams.TeamStatusApproved {
				active = true
				break
			}
		}
		if !active {
			eligible = append(eligible, p)
		}
	}
	return eligible, nil
}

func (s *Service) groupBySemester(allParticipants []participants.Participant) map[int][]participants.Participant {
	bySemester := make(map[int][]participants.Participant)
	for _, p := range allParticipants {
		bySemester[p.Semestre] = append(bySemester[p.Semestre], p)
	}
	return bySemester
}

func (s *Service) flattenRemaining(bySemester map[int][]participants.Participant) []participants.Participant {
	var remaining []participants.Participant
	for _, list := range bySemester {
		remaining = append(remaining, list...)
	}
	return remaining
}

func (s *Service) tryFormGroupsWithDifferentSemesters(ctx context.Context, bySemester map[int][]participants.Participant, exhaustive bool) map[int][]participants.Participant {
	rand.Seed(time.Now().UnixNano())

	for {
		var semesters []int
		for sem, list := range bySemester {
			if len(list) > 0 {
				semesters = append(semesters, sem)
			}
		}
		if len(semesters) < 3 {
			break
		}

		chosenSemesters := make([]int, 3)
		indices := rand.Perm(len(semesters))[:3]
		for i, idx := range indices {
			chosenSemesters[i] = semesters[idx]
		}

		ok := true
		for _, sem := range chosenSemesters {
			if len(bySemester[sem]) == 0 {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		group := make([]participants.Participant, 3)
		for i, sem := range chosenSemesters {
			list := bySemester[sem]
			idx := rand.Intn(len(list))
			group[i] = list[idx]
			bySemester[sem] = append(list[:idx], list[idx+1:]...)
		}

		ids := make([]primitive.ObjectID, 3)
		for i, p := range group {
			ids[i] = p.ID
		}
		if _, err := s.teamSvc.CreateDraw(ctx, ids); err != nil {
			return bySemester
		}

		if !exhaustive {
			break
		}
	}
	return bySemester
}

func (s *Service) formAnyGroups(ctx context.Context, bySemester map[int][]participants.Participant) map[int][]participants.Participant {
	rand.Seed(time.Now().UnixNano())

	var all []participants.Participant
	for _, list := range bySemester {
		all = append(all, list...)
	}

	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})

	for len(all) >= 3 {
		group := all[:3]
		all = all[3:]

		ids := make([]primitive.ObjectID, 3)
		for i, p := range group {
			ids[i] = p.ID
		}
		if _, err := s.teamSvc.CreateDraw(ctx, ids); err != nil {
			break
		}
	}

	newMap := make(map[int][]participants.Participant)
	for _, p := range all {
		newMap[p.Semestre] = append(newMap[p.Semestre], p)
	}
	return newMap
}
