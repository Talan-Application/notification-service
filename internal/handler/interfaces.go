package handler

import (
	"context"

	"github.com/Talan-Application/notification-service/internal/domain"
)

type EventHandler interface {
	Handle(ctx context.Context, event domain.Event) error
}
