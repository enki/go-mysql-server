// Copyright 2026 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package analyzer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dolthub/go-mysql-server/memory"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/planbuilder"
)

// A write reads the target of every stored trigger of its database. The target is parsed once per statement text and
// SQL mode, then read from the memo without a parse; the same text under another SQL mode is its own entry.
func TestStoredTriggerTarget(t *testing.T) {
	db := memory.NewDatabase("mydb")
	pro := memory.NewDBProvider(db)
	ctx := sql.NewContext(context.Background(), sql.WithSession(memory.NewSession(sql.NewBaseSession(), pro)))
	ctx.SetCurrentDatabase("mydb")
	b := planbuilder.New(ctx, newTestCatalog(db), nil)

	const statement = `CREATE TRIGGER "t_ai" AFTER UPDATE ON "Logged" FOR EACH ROW SET @x = 1`
	ansi := sql.TriggerDefinition{Name: "t_ai", CreateStatement: statement, SqlMode: "ANSI_QUOTES"}
	plain := sql.TriggerDefinition{Name: "t_ai", CreateStatement: statement, SqlMode: "STRICT_TRANS_TABLES"}

	got := storedTriggerTarget(b, ansi)
	require.NoError(t, got.err)
	require.Equal(t, triggerTarget{table: "logged", event: "update", isTrigger: true}, got)
	// A remembered target needs no builder.
	require.Equal(t, got, storedTriggerTarget(nil, ansi))

	// Without ANSI_QUOTES the quoted name is a string, which doesn't parse as a table; that is remembered too.
	require.Error(t, storedTriggerTarget(b, plain).err)
	require.Error(t, storedTriggerTarget(nil, plain).err)
	require.Equal(t, got, storedTriggerTarget(nil, ansi))

	notTrigger := sql.TriggerDefinition{Name: "v", CreateStatement: "CREATE VIEW v AS SELECT 1", SqlMode: "ANSI_QUOTES"}
	require.Equal(t, triggerTarget{}, storedTriggerTarget(b, notTrigger))
}
