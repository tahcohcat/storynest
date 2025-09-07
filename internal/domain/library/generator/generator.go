package generator

import (
	"context"
	"storynest/internal/domain/story"
)

type StoryGenerator interface {
	ListOnlineResources() ([]*story.OnlineResource, error)
	LoadResource(context.Context, *story.OnlineResource) (*story.Item, error)
}
