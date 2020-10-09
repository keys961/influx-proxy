// Copyright 2016 Eleme. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package backend

var (
	ForbidCommands  = "(?i:^\\s*grant|^\\s*revoke|^\\s*alter|^\\s*create|^\\s*drop|^\\s*select.*into|.*?;.+)"
	SupportCommands = "(?i:^\\s*show.*from|^\\s*select.*from|^\\s*delete.*from)"
)
