// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export interface IReply {
  200: {
    success: boolean;
    message: string;
  };
  400: {
    success: boolean;
    error: {
      code: string;
      message: string;
    };
  };
}
