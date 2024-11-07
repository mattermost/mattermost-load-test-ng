package userentity

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

// GetChannelBookmarks fetches bookmarks for the given channel since a specific timestamp.
func (ue *UserEntity) GetChannelBookmarks(channelId string, since int64) error {
	bookmarks, _, err := ue.client.ListChannelBookmarksForChannel(context.Background(), channelId, since)
	if err != nil {
		return err
	}

	return ue.store.SetChannelBookmarks(bookmarks)
}

// AddChannelBookmark creates a bookmark on the given channel
func (ue *UserEntity) AddChannelBookmark(bookmark *model.ChannelBookmark) error {
	bookmarkResp, _, err := ue.client.CreateChannelBookmark(context.Background(), bookmark)
	if err != nil {
		return err
	}

	return ue.store.AddChannelBookmark(bookmarkResp)
}

// UpdateChannelBookmark updates a given bookmark.
func (ue *UserEntity) UpdateChannelBookmark(bookmark *model.ChannelBookmarkWithFileInfo) error {
	patch := &model.ChannelBookmarkPatch{
		FileId:      &bookmark.FileId,
		DisplayName: &bookmark.DisplayName,
		SortOrder:   &bookmark.SortOrder,
		LinkUrl:     &bookmark.LinkUrl,
		ImageUrl:    &bookmark.ImageUrl,
		Emoji:       &bookmark.Emoji,
	}

	result, _, err := ue.client.UpdateChannelBookmark(context.Background(), bookmark.ChannelId, bookmark.Id, patch)
	if err != nil {
		return err
	}

	if result.Deleted != nil {
		bId := result.Deleted.Id
		if err := ue.store.DeleteChannelBookmark(bId); err != nil {
			return err
		}
	}

	return ue.store.UpdateChannelBookmark(result.Updated)
}

// DeleteChannelBookmark deletes a given bookmarkId from a given channelId.
func (ue *UserEntity) DeleteChannelBookmark(channelId, bookmarkId string) error {
	result, _, err := ue.client.DeleteChannelBookmark(context.Background(), channelId, bookmarkId)
	if err != nil {
		return err
	}

	return ue.store.DeleteChannelBookmark(result.Id)
}

// UpdateChannelBookmarkSortOrder sets the new position of a bookmark for the given channel
func (ue *UserEntity) UpdateChannelBookmarkSortOrder(channelId, bookmarkId string, sortOrder int64) error {
	result, _, err := ue.client.UpdateChannelBookmarkSortOrder(context.Background(), channelId, bookmarkId, sortOrder)
	if err != nil {
		return err
	}

	return ue.store.SetChannelBookmarks(result)
}
