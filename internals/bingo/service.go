package bingo

import "context"

type BingoService interface {
	GetBingo(ctx context.Context) error
}

type svc struct {
	// repository
}

func NewService() BingoService {
	return &svc{}
}

func (s *svc) GetBingo(ctx context.Context) error {
	return nil
}
