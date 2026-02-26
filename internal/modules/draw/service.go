package draw

import (
	"context"
	"log"
	"math/rand"
	"time"

	"copasoftware/internal/modules/participants"
	"copasoftware/internal/modules/ranking"
	"copasoftware/internal/modules/teams"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Result struct {
	Remaining     []participants.Participant
	TotalEligible int
}

type Service struct {
	participantSvc *participants.Service
	teamSvc        *teams.Service
	rankingSvc     *ranking.Service
}

func NewService(
	participantSvc *participants.Service,
	teamSvc *teams.Service,
	rankingSvc *ranking.Service,
) *Service {
	return &Service{
		participantSvc: participantSvc,
		teamSvc:        teamSvc,
		rankingSvc:     rankingSvc,
	}
}

func (s *Service) RunDraw(ctx context.Context, isFinal bool) (*Result, error) {
	eligible, err := s.getEligibleParticipants(ctx)
	if err != nil {
		return nil, err
	}
	totalEligible := len(eligible)
	if totalEligible == 0 {
		return &Result{Remaining: []participants.Participant{}, TotalEligible: 0}, nil
	}

	bySemester := s.groupBySemester(eligible)

	if !isFinal {
		bySemester = s.tryFormGroupsWithDifferentSemesters(ctx, bySemester, false)
		return &Result{
			Remaining:     s.flattenRemaining(bySemester),
			TotalEligible: totalEligible,
		}, nil
	}

	bySemester = s.tryFormGroupsWithDifferentSemesters(ctx, bySemester, true)
	bySemester = s.tryFormGroupsWithTwoSemesters(ctx, bySemester)
	bySemester = s.formGroupsSameSemester(ctx, bySemester)

	return &Result{
		Remaining:     s.flattenRemaining(bySemester),
		TotalEligible: totalEligible,
	}, nil
}

func (s *Service) getEligibleParticipants(ctx context.Context) ([]participants.Participant, error) {
	all, err := s.participantSvc.List(ctx)
	if err != nil {
		return nil, err
	}
	var eligible []participants.Participant
	for _, p := range all {
		teamsList, err := s.teamSvc.GetTeamsByParticipant(ctx, p.ID, p.Matricula)
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
		team, err := s.teamSvc.CreateDraw(ctx, ids)
		if err != nil {
			continue
		}
		if s.rankingSvc != nil && team != nil {
			if err := s.rankingSvc.InitializeTeam(ctx, team.ID); err != nil {
				log.Printf("erro ao inicializar ranking para time sorteado %s: %v", team.ID.Hex(), err)
			}
		}
		if !exhaustive {
			break
		}
	}
	return bySemester
}

func (s *Service) tryFormGroupsWithTwoSemesters(ctx context.Context, bySemester map[int][]participants.Participant) map[int][]participants.Participant {
	for {
		var semesters []int
		for sem, list := range bySemester {
			if len(list) > 0 {
				semesters = append(semesters, sem)
			}
		}
		if len(semesters) < 2 {
			break
		}
		if len(semesters) == 2 {
			if len(bySemester[semesters[0]])+len(bySemester[semesters[1]]) < 3 {
				break
			}
			sem1 := semesters[0]
			sem2 := semesters[1]
			list1 := bySemester[sem1]
			list2 := bySemester[sem2]

			if len(list1) >= 2 && len(list2) >= 1 {
				group := []participants.Participant{
					list1[0], list1[1], list2[0],
				}
				bySemester[sem1] = list1[2:]
				bySemester[sem2] = list2[1:]
				s.createTeamFromGroup(ctx, group)
				continue
			}
			if len(list1) >= 1 && len(list2) >= 2 {
				group := []participants.Participant{
					list1[0], list2[0], list2[1],
				}
				bySemester[sem1] = list1[1:]
				bySemester[sem2] = list2[2:]
				s.createTeamFromGroup(ctx, group)
				continue
			}
			break
		}
		chosenSemesters := make([]int, 2)
		indices := rand.Perm(len(semesters))[:2]
		for i, idx := range indices {
			chosenSemesters[i] = semesters[idx]
		}
		semA := chosenSemesters[0]
		semB := chosenSemesters[1]
		listA := bySemester[semA]
		listB := bySemester[semB]

		if len(listA) >= 2 && len(listB) >= 1 {
			group := []participants.Participant{
				listA[0], listA[1], listB[0],
			}
			bySemester[semA] = listA[2:]
			bySemester[semB] = listB[1:]
			s.createTeamFromGroup(ctx, group)
		} else if len(listA) >= 1 && len(listB) >= 2 {
			group := []participants.Participant{
				listA[0], listB[0], listB[1],
			}
			bySemester[semA] = listA[1:]
			bySemester[semB] = listB[2:]
			s.createTeamFromGroup(ctx, group)
		} else {
			continue
		}
	}
	return bySemester
}

func (s *Service) formGroupsSameSemester(ctx context.Context, bySemester map[int][]participants.Participant) map[int][]participants.Participant {
	for sem, list := range bySemester {
		for len(list) >= 3 {
			group := list[:3]
			list = list[3:]
			s.createTeamFromGroup(ctx, group)
		}
		bySemester[sem] = list
	}
	return bySemester
}

func (s *Service) createTeamFromGroup(ctx context.Context, group []participants.Participant) {
	ids := make([]primitive.ObjectID, 3)
	for i, p := range group {
		ids[i] = p.ID
	}
	team, err := s.teamSvc.CreateDraw(ctx, ids)
	if err != nil {
		log.Printf("erro ao criar time sorteado: %v", err)
		return
	}
	if s.rankingSvc != nil && team != nil {
		if err := s.rankingSvc.InitializeTeam(ctx, team.ID); err != nil {
			log.Printf("erro ao inicializar ranking para time sorteado %s: %v", team.ID.Hex(), err)
		}
	}
}
