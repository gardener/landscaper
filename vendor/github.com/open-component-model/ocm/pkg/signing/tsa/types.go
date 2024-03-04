// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Open Component Model contributors.
//
// SPDX-License-Identifier: Apache-2.0

package tsa

import (
	cms "github.com/InfiniteLoopSpace/go_S-MIME/cms/protocol"
	tsa "github.com/InfiniteLoopSpace/go_S-MIME/timestamp"
)

type TimeStamp = cms.SignedData

type MessageImprint = tsa.MessageImprint
