// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"fmt"
	"math/rand"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

var (
	bookmarkNames = []string{"this is a file", "this is a link", "this is another file", "this is another link"}
	bookmarkType  = []model.ChannelBookmarkType{model.ChannelBookmarkLink, model.ChannelBookmarkFile}
)

func (c *SimulController) addBookmark(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	emoji := ""
	// 10% of the times bookmarks will have an emoji assigned.
	if rand.Float64() < 0.1 {
		emoji = control.RandomEmoji()
	}

	bookmark := &model.ChannelBookmark{
		ChannelId:   channel.Id,
		DisplayName: control.PickRandomString(bookmarkNames),
		Emoji:       emoji,
		Type:        bookmarkType[rand.Intn(len(bookmarkType))],
	}

	if bookmark.Type == model.ChannelBookmarkFile {
		control.AttachFileToBookmark(u, bookmark)
	} else {
		bookmark.LinkUrl = control.RandomLink()
	}

	err = u.AddChannelBookmark(channel.Id, bookmark)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("bookmark created in channel id %v", channel.Id)}
}

func (c *SimulController) AddChannelBookmark(u user.User) control.UserActionResponse {
	if ok, resp := control.ChannelBookmarkEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "channel bookmarks not enabled"}
	}

	return c.addBookmark(u)
}

func (c *SimulController) UpdateOrAddBookmark(u user.User) control.UserActionResponse {
	if ok, resp := control.ChannelBookmarkEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "channel bookmarks not enabled"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	currentBookmarks := u.Store().ChannelBookmarks(channel.Id)
	if len(currentBookmarks) > 0 {
		// here we update
		bookmark := currentBookmarks[rand.Intn(len(currentBookmarks))]
		if bookmark != nil {
			bookmarkWithFileInfo := bookmark.Clone()
			bookmarkWithFileInfo.DisplayName = control.PickRandomString(bookmarkNames)

			// 10% of the times bookmarks will have an emoji assigned.
			if bookmarkWithFileInfo.Emoji == "" && rand.Float64() < 0.1 {
				bookmarkWithFileInfo.Emoji = control.RandomEmoji()
			}

			if bookmarkWithFileInfo.Type == model.ChannelBookmarkFile {
				control.AttachFileToBookmark(u, bookmarkWithFileInfo.ChannelBookmark)
			} else {
				bookmarkWithFileInfo.LinkUrl = control.RandomLink()
			}

			err = u.UpdateChannelBookmark(bookmarkWithFileInfo)

			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			return control.UserActionResponse{Info: fmt.Sprintf("bookmark %v updated in channel id %v", bookmarkWithFileInfo.Id, channel.Id)}
		}
	}

	return c.addBookmark(u)
}

func (c *SimulController) DeleteBookmark(u user.User) control.UserActionResponse {
	if ok, resp := control.ChannelBookmarkEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "channel bookmarks not enabled"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	currentBookmarks := u.Store().ChannelBookmarks(channel.Id)
	if len(currentBookmarks) > 0 {
		bookmark := currentBookmarks[rand.Intn(len(currentBookmarks))]
		if bookmark != nil {
			err = u.DeleteChannelBookmark(bookmark.ChannelId, bookmark.Id)
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			return control.UserActionResponse{Info: fmt.Sprintf("bookmark id %v deleted in channel id %v", bookmark.Id, channel.Id)}
		}
	}

	return control.UserActionResponse{Info: "no channel bookmarks deleted"}
}

func (c *SimulController) UpdateBookmarksSortOrder(u user.User) control.UserActionResponse {
	if ok, resp := control.ChannelBookmarkEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "channel bookmarks not enabled"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	currentBookmarks := u.Store().ChannelBookmarks(channel.Id)
	if len(currentBookmarks) > 1 {
		bookmark := currentBookmarks[rand.Intn(len(currentBookmarks))]
		newIndex := rand.Int63n(int64(len(currentBookmarks)))
		if bookmark != nil {
			err = u.UpdateChannelBookmarkSortOrder(channel.Id, bookmark.Id, newIndex)
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			return control.UserActionResponse{Info: fmt.Sprintf("bookmark id %v in channel id %v sorted at index %d", bookmark.Id, channel.Id, newIndex)}
		}
	}

	return control.UserActionResponse{Info: "not enough channel bookmarks to sort"}
}
