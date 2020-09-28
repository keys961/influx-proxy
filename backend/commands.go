// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

var (
	ForbidCommands   = "(?i:^\\s*delete|^\\s*drop|^\\s*grant|^\\s*revoke|^\\s*alter|^\\s*create)"
	SupportCommands  = "(?i:where.*time|show.*from|^\\s*select)"
	ExecutorCommands = "(?i:show.*measurements)"
)
